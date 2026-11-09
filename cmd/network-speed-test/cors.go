package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Only the authenticated LuCI origin that issued this peer's grant may read
// cross-origin responses. Preflight grants no access to a test or its data.
func (s *server) allowNativeOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "http://"+r.Host {
		return true
	}
	w.Header().Add("Vary", "Origin")
	var grant accessGrant
	data, err := os.ReadFile(filepath.Join(s.dir, "grant.json"))
	if err != nil || json.Unmarshal(data, &grant) != nil || origin != grant.Origin || grant.Peer != peer(r) || !grant.Expires.After(s.now()) {
		fail(w, 403, "Use Start on the LuCI Speed Test page.")
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	if r.Method != "OPTIONS" {
		return true
	}
	if r.Header.Get("Access-Control-Request-Method") != "POST" {
		fail(w, 405, "Only POST is allowed.")
		return false
	}
	for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
		header = strings.ToLower(strings.TrimSpace(header))
		if header != "" && header != "authorization" && header != "content-type" {
			fail(w, 403, "Unsupported request header.")
			return false
		}
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.WriteHeader(http.StatusNoContent)
	return false
}
