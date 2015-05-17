package handlers

import (
	"log"
	"net"
	"net/http"

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
		c.Next()
	}
}

func checkServer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"server": "yolo"})
}

func UseServerCheck(g *gin.RouterGroup) {
	g.Use(addServerCheck(), bindParams(), dnsLookup())
	g.GET("/", checkServer)
}
