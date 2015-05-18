package checks

import (
	"errors"
	"net"

	"github.com/zmap/zgrab/ztools/ztls"
)

func CheckExport(c net.Conn, name string) (*ztls.DHParams, error) {
	config := new(ztls.Config)
	config.CipherSuites = ztls.DHEExportCiphers
	config.ServerName = name
	config.InsecureSkipVerify = true
	config.MinVersion = ztls.VersionSSL30
	config.ForceSuites = true
	tlsConn := ztls.Client(c, config)
	tlsConn.Handshake()
	hl := tlsConn.GetHandshakeLog()
	if hl == nil {
		return nil, nil
	}
	return hl.DHExportParams, nil
}

func CheckDHE(c net.Conn, name string) (*ztls.DHParams, error) {
	config := new(ztls.Config)
	config.ServerName = name
	config.CipherSuites = ztls.DHECiphers
	config.InsecureSkipVerify = true
	config.MinVersion = ztls.VersionSSL30
	config.ForceSuites = true
	tlsConn := ztls.Client(c, config)
	tlsConn.Handshake()
	hl := tlsConn.GetHandshakeLog()
	if hl == nil {
		return nil, nil
	}
	return hl.DHParams, nil
}

func CheckChrome(c net.Conn, name string) (*ztls.DHParams, string, error) {
	config := new(ztls.Config)
	config.ServerName = name
	config.CipherSuites = ztls.ChromeCiphers
	config.InsecureSkipVerify = true
	config.MinVersion = ztls.VersionSSL30
	config.ForceSuites = true
	tlsConn := ztls.Client(c, config)
	tlsConn.Handshake()
	hl := tlsConn.GetHandshakeLog()
	if hl == nil || hl.ServerCertificates == nil {
		return nil, "", errors.New("does not support TLS")
	}
	var cipher string
	if hl.ServerHello != nil {
		cipher = hl.ServerHello.CipherSuite.String()
	}
	return hl.DHParams, cipher, nil
}
