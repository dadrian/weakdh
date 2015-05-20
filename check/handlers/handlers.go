package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/dadrian/weakdh/check/checks"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/zmap/zgrab/ztools/ztls"
)

type ServerCheck struct {
	Domain    string             `json:"domain"`
	IP        []net.IP           `json:"ip_addresses"`
	Results   []*SingleHostCheck `json:"results"`
	Error     *string            `json:"error"`
	Timestamp string             `json:"timestamp"`
}

type SingleHostCheck struct {
	Domain         string         `json:"-"`
	IP             string         `json:"ip"`
	HasTLS         bool           `json:"has_tls"`
	DHParams       *ztls.DHParams `json:"dh_params"`
	ExportDHParams *ztls.DHParams `json:"export_dh_params"`
	ChromeDHParams *ztls.DHParams `json:"chrome_dh_params"`
	ChromeCipher   *string        `json:"chrome_cipher"`
	Error          *string        `json:"error"`
}

type ServerCheckParams struct {
	Server   string `form:"server" binding:"required"`
	Protocol string
}

type checkHandler struct {
}

func addServerCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		check := new(ServerCheck)
		check.Timestamp = time.Now().Format(time.RFC3339)
		c.Set("check", check)
		c.Next()
	}
}

func cleanDomain(d string) string {
	u, err := url.Parse(d)
	// Not a URL, just try the hostname
	if err != nil {
		return d
	}
	if u.Path == d || u.Host == "" {
		return d
	}

	hostname := u.Host
	h, _, e := net.SplitHostPort(hostname)
	// If split failed, just assume we have a hostname
	if e != nil || h == "" {
		return hostname
	}
	// Split didn't fail, return what the net library thinks the host is
	return h
}

func bindParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		check := c.MustGet("check").(*ServerCheck)
		p := new(ServerCheckParams)
		if c.BindWith(p, binding.Form) != true {
			c.Error(errBadCheckParams, nil)
			c.AbortWithStatus(400)
		}
		// Check if IP or domain
		if ip := net.ParseIP(p.Server); ip != nil {
			check.IP = []net.IP{ip}
		} else {
			check.Domain = cleanDomain(p.Server)
		}
		c.Next()
	}
}

func dnsLookup() gin.HandlerFunc {
	return func(c *gin.Context) {
		check := c.MustGet("check").(*ServerCheck)
		if check.IP == nil {
			// We have a domain, lookup IP addresses
			addrs, err := net.LookupIP(check.Domain)
			if err != nil {
				s := "DNS lookup failed"
				c.Error(err, s)
				check.Error = &s
				c.Next()
				return
			}
			hostCount := len(addrs)
			check.IP = addrs
			check.Results = make([]*SingleHostCheck, hostCount)
		} else {
			check.Results = make([]*SingleHostCheck, 1)
		}
		c.Next()
	}
}

func handshakes() gin.HandlerFunc {
	return func(c *gin.Context) {
		check := c.MustGet("check").(*ServerCheck)
		domain := check.Domain
		allHostsGroup := new(sync.WaitGroup)
		allHostsGroup.Add(len(check.IP))
		for idx, addr := range check.IP {
			go func(i int, a net.IP, wg *sync.WaitGroup) {
				defer wg.Done()
				checkGroup := new(sync.WaitGroup)
				checkGroup.Add(3)
				fullAddress := net.JoinHostPort(a.String(), "443")
				var normal, export, chrome *ztls.DHParams
				var chrome_cipher string
				var noTLS bool
				var connErr error

				setErrorOnce := new(sync.Once)

				dialer := new(net.Dialer)
				dl := time.Now().Add(time.Second * 10)
				dialer.Deadline = dl

				go func(fullAddress string, cg *sync.WaitGroup) {
					defer cg.Done()
					conn, err := dialer.Dial("tcp", fullAddress)
					if err != nil {
						setErrorOnce.Do(func() {
							connErr = err
						})
						return
					}
					conn.SetDeadline(dl)
					export, _ = checks.CheckExport(conn, domain)
				}(fullAddress, checkGroup)

				go func(fullAddress string, cg *sync.WaitGroup) {
					defer cg.Done()
					conn, err := dialer.Dial("tcp", fullAddress)
					if err != nil {
						setErrorOnce.Do(func() {
							connErr = err
						})
						return
					}
					conn.SetDeadline(dl)
					normal, _ = checks.CheckDHE(conn, domain)
				}(fullAddress, checkGroup)

				go func(fullAddress string, cg *sync.WaitGroup) {
					defer cg.Done()
					conn, err := dialer.Dial("tcp", fullAddress)
					if err != nil {
						setErrorOnce.Do(func() {
							connErr = err
						})
						return
					}
					conn.SetDeadline(dl)
					chrome, chrome_cipher, err = checks.CheckChrome(conn, domain)
					if err != nil {
						noTLS = true
					}
				}(fullAddress, checkGroup)

				checkGroup.Wait()

				var errStringPtr *string
				if connErr != nil {
					s := connErr.Error()
					errStringPtr = &s
				}
				out := &SingleHostCheck{
					IP:             a.String(),
					HasTLS:         !noTLS,
					DHParams:       normal,
					ExportDHParams: export,
					ChromeDHParams: chrome,
					ChromeCipher:   &chrome_cipher,
					Error:          errStringPtr,
				}
				check.Results[i] = out

			}(idx, addr, allHostsGroup)
		}
		allHostsGroup.Wait()
		c.Next()
	}
}

func checkServer(c *gin.Context) {
	check := c.MustGet("check").(*ServerCheck)
	c.JSON(http.StatusOK, check)
}

func UseServerCheck(g *gin.RouterGroup, outputFile *os.File) {

	outputChan := make(chan *ServerCheck, 1024)
	enc := json.NewEncoder(outputFile)

	sendToChan := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			check := c.MustGet("check").(*ServerCheck)
			outputChan <- check
			c.Next()
		}
	}

	go func() {
		for c := range outputChan {
			enc.Encode(c)
		}
	}()

	g.Use(addServerCheck(), bindParams(), dnsLookup(), handshakes(), sendToChan())
	g.GET("/", checkServer)
}
