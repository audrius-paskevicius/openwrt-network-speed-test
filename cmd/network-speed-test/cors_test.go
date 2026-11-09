package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestNativeCORS(t *testing.T) {
	f := newFixture(t)
	token := grantForTest(t, f)
	f.s.native = true
	request := func(method, origin, peer, headers, bearer string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://router.test:18080/api?op=connection", nil)
		r.RemoteAddr = peer
		r.Header.Set("Origin", origin)
		if method == "OPTIONS" {
			r.Header.Set("Access-Control-Request-Method", "POST")
			r.Header.Set("Access-Control-Request-Headers", headers)
		} else {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		f.s.ServeHTTP(w, r)
		return w
	}
	w := request("OPTIONS", "http://router.test", "192.0.2.2:4567", "authorization,content-type", "")
	expect(t, w, 204)
	if w.Header().Get("Access-Control-Allow-Origin") != "http://router.test" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("CORS must allow only the issuing origin without credentials")
	}
	expect(t, request("POST", "http://router.test", "192.0.2.2:4567", "", token), 200)
	expect(t, request("POST", "http://router.test", "192.0.2.2:4567", "", testSession), 403)
	for _, origin := range []string{"http://evil.test", "null", "https://router.test"} {
		w = request("OPTIONS", origin, "192.0.2.2:4567", "authorization", "")
		expect(t, w, 403)
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("unrelated origin allowed")
		}
		expect(t, request("POST", origin, "192.0.2.2:4567", "", token), 403)
	}
	expect(t, request("OPTIONS", "http://router.test", "192.0.2.3:4567", "authorization", ""), 403)
	expect(t, request("OPTIONS", "http://router.test", "192.0.2.2:4567", "x-custom", ""), 403)
	f.now = f.now.Add(31 * time.Minute)
	expect(t, request("OPTIONS", "http://router.test", "192.0.2.2:4567", "authorization", ""), 403)
}
