package httpapi

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

func localGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostIsRebindSafe(r.Host) {
			writeErr(w, http.StatusForbidden, "FORBIDDEN",
				"request Host is not a local address; fizza only serves loopback clients")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !originIsLocal(origin) {
			writeErr(w, http.StatusForbidden, "FORBIDDEN",
				"cross-origin request rejected; fizza serve is local-only and unauthenticated")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func originIsLocal(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return isLoopbackHost(u.Hostname())
}

func hostIsRebindSafe(host string) bool {
	if host == "" {
		return false
	}
	name := hostnameOnly(host)
	if strings.EqualFold(name, "localhost") {
		return true
	}
	return net.ParseIP(name) != nil
}

func isLoopbackHost(name string) bool {
	if strings.EqualFold(name, "localhost") {
		return true
	}
	ip := net.ParseIP(name)
	return ip != nil && ip.IsLoopback()
}

func hostnameOnly(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.Trim(host, "[]")
}
