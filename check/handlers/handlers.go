package handlers

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dadrian/weakdh/check/checks"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/zmap/zgrab/ztools/ztls"
)

type ServerCheck struct {
	Domain  string             `json:"domain"`
	IP      []net.IP           `json:"ip_addresses"`
	Results []*SingleHostCheck `json:"results"`
}

type SingleHostCheck struct {
	Domain         string         `json:"-"`
	IP             string         `json:"ip"`
	HasTLS         bool           `json:"has_tls"`
	DHParams       *ztls.DHParams `json:"dh_params"`
	ExportDHParams *ztls.DHParams `json:"export_dh_params"`
	ChromeDHParams *ztls.DHParams `json:"chrome_dh_params"`
	Error          error          `json:"error"`
}

type ServerCheckParams struct {
	Server   string `form:"server" binding:"required"`
	Protocol string
}

type checkHandler struct {
}

func addServerCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Print("Add Check Object")
		check := new(ServerCheck)
		c.Set("check", check)
		c.Next()
	}
}

func bindParams() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Print("Bind Params")
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
			check.Domain = p.Server
		}
		c.Next()
	}
}

func dnsLookup() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Print("DNS Lookup")
		check := c.MustGet("check").(*ServerCheck)
		if check.IP == nil {
			// We have a domain, lookup IP addresses
			addrs, err := net.LookupIP(check.Domain)
			if err != nil {
				c.Error(err, "DNS Lookup Failed")
				c.AbortWithStatus(500)
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
		log.Print(domain)
		allHostsGroup := new(sync.WaitGroup)
		allHostsGroup.Add(len(check.IP))
		for idx, addr := range check.IP {
			go func(i int, a net.IP, wg *sync.WaitGroup) {
				defer wg.Done()
				checkGroup := new(sync.WaitGroup)
				checkGroup.Add(3)
				fullAddress := net.JoinHostPort(a.String(), "443")
				var normal, export, chrome *ztls.DHParams
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
					chrome, err = checks.CheckChrome(conn, domain)
					if err != nil {
						noTLS = true
					}
				}(fullAddress, checkGroup)

				checkGroup.Wait()
				out := &SingleHostCheck{
					HasTLS:         !noTLS,
					DHParams:       normal,
					ExportDHParams: export,
					ChromeDHParams: chrome,
					Error:          connErr,
				}
				log.Print(i)
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

func UseServerCheck(g *gin.RouterGroup) {
	g.Use(addServerCheck(), bindParams(), dnsLookup(), handshakes())
	g.GET("/", checkServer)
}
