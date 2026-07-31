package router

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSConfig struct {
	pattern          string
	enabled          bool
	allowedOrigins   string
	allowedMethods   string
	allowedHeaders   string
	allowCredentials bool
	maxAge           string
}

func (c *CORSConfig) Enable() *CORSConfig {
	c.enabled = true
	return c
}

func (c *CORSConfig) Origin(origins ...string) *CORSConfig {
	c.allowedOrigins = strings.Join(origins, ", ")
	return c
}

func (c *CORSConfig) Methods(methods ...string) *CORSConfig {
	c.allowedMethods = strings.Join(methods, ", ")
	return c
}

func (c *CORSConfig) Headers(headers ...string) *CORSConfig {
	c.allowedHeaders = strings.Join(headers, ", ")
	return c
}

func (c *CORSConfig) Credentials(allow bool) *CORSConfig {
	c.allowCredentials = allow
	return c
}

func (c *CORSConfig) MaxAge(seconds int) *CORSConfig {
	c.maxAge = strconv.Itoa(seconds)
	return c
}

func (c *CORSConfig) applyHeaders(w http.ResponseWriter) {
	h := w.Header()
	if c.allowedOrigins != "" {
		h.Set("Access-Control-Allow-Origin", c.allowedOrigins)
	}
	if c.allowedMethods != "" {
		h.Set("Access-Control-Allow-Methods", c.allowedMethods)
	}
	if c.allowedHeaders != "" {
		h.Set("Access-Control-Allow-Headers", c.allowedHeaders)
	}
	if c.allowCredentials {
		h.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge != "" {
		h.Set("Access-Control-Max-Age", c.maxAge)
	}
}
