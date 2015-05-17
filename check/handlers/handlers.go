package handlers

import (
	"errors"
	"log"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/zmap/zgrab/ztools/ztls"
)

type ServerCheck struct {
	Domain         string
	IP             net.IP
	DHParams       *ztls.DHParams
	ExportDHParams *ztls.DHParams
}

type ServerCheckParams struct {
	Server   string `form:"server" binding:"required"`
	Protocol string ``
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
		if check.IP = net.ParseIP(p.Server); check.IP == nil {
			check.Domain = p.Server
		}
		c.Next()
	}
}

func dnsLookup() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Print("DNS Lookup")
		check := c.MustGet("check").(*ServerCheck)
		if check.IP != nil {
			log.Print("Have an IP, no need to DNS lookup")
		} else {
			c.Error(errors.New("Only IPs supported"), nil)
			c.AbortWithStatus(400)
		}
		c.Next()
	}
}

func startHandshakes() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Print("start handshakes")
		check := c.MustGet("check").(*ServerCheck)
		config := new(ztls.Config)
		config.InsecureSkipVerify = true
		address := net.JoinHostPort(check.IP.String(), "443")
		conn, err := net.Dial("tcp", address)
		if err != nil {
			c.Error(err, nil)
			c.AbortWithStatus(500)
		}
		tlsConn := ztls.Client(conn, config)
		if err = tlsConn.Handshake(); err != nil {
			c.Error(err, nil)
			c.AbortWithStatus(500)
		}
		h := tlsConn.GetHandshakeLog()
		c.JSON(200, h)
	}
}

func checkServer(c *gin.Context) {
	//	c.JSON(http.StatusOK, gin.H{"server": "yolo"})
}

func UseServerCheck(g *gin.RouterGroup) {
	g.Use(addServerCheck(), bindParams(), dnsLookup(), startHandshakes())
	g.GET("/", checkServer)
}
