package net

import (
	"net"
	"net/http"
)

func GetIP(r *http.Request, proxied bool) net.IP {
	var ipStr string
	if proxied {
		ipStr = r.Header.Get("X-Forwarded-For")
		if ipStr == "" {
			ipStr = r.Header.Get("X-Real-Ip")
		}
	} else {
		ipStr = r.RemoteAddr
	}

	return net.ParseIP(ipStr)
}
