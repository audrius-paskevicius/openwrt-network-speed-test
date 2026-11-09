package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const testSession = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type fixture struct {
	s      *server
	cookie *http.Cookie
	t      *testing.T
	now    time.Time
}

func newFixture(t *testing.T) *fixture {
	var mutex sync.Mutex
	f := &fixture{t: t, now: time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)}
	f.s = &server{dir: t.TempDir(), authorize: func(_ context.Context, sid string) bool { return sid == testSession }, inspect: func(context.Context, string) connection {
		return connection{Interface: "phy0-ap0", Band: "5 GHz", Channel: 36}
	}, cpu: func() map[string]cpuTimes { return map[string]cpuTimes{"cpu0": {Total: 100, Idle: 50}} }, lock: func(string) (func(), error) {
		if !mutex.TryLock() {
			return nil, errors.New("busy")
		}
		return mutex.Unlock, nil
	}, now: func() time.Time { return f.now }, streamLimit: 65536, streamDuration: time.Second}
	return f
}
func (f *fixture) request(op, session string, body io.Reader) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "http://router.test/cgi-bin/network-speed-test?op="+op, body)
	r.RemoteAddr = "192.0.2.2:1234"
	r.Header.Set("Authorization", "Bearer "+session)
	r.Header.Set("Origin", "http://router.test")
	if op == "upload" {
		r.Header.Set("Content-Type", "application/octet-stream")
	}
	if f.cookie != nil {
		r.AddCookie(f.cookie)
	}
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	return w
}
func (f *fixture) start() {
	f.t.Helper()
	w := f.request("start", testSession, nil)
	if w.Code != 200 {
		f.t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	f.cookie = w.Result().Cookies()[0]
}
func expect(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d, want %d: %s", w.Code, status, w.Body.String())
	}
}
func TestAuthenticationEveryOperation(t *testing.T) {
	for _, op := range []string{"start", "ping", "download", "upload", "finish"} {
		for _, sid := range []string{"", strings.Repeat("0", 32), strings.Repeat("b", 32), strings.Repeat("A", 32), "../../../etc/passwd"} {
			f := newFixture(t)
			expect(t, f.request(op, sid, strings.NewReader("secret")), 403)
			if _, e := os.Stat(filepath.Join(f.s.dir, "run.json")); !os.IsNotExist(e) {
				t.Fatal("unauthorized request wrote state")
			}
		}
	}
}
func TestMethodsAndOrigin(t *testing.T) {
	f := newFixture(t)
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "PUT"} {
		r := httptest.NewRequest(method, "http://router.test/?op=start", nil)
		w := httptest.NewRecorder()
		f.s.ServeHTTP(w, r)
		expect(t, w, 405)
	}
	r := httptest.NewRequest("POST", "http://router.test/?op=start", nil)
	r.Header.Set("Authorization", "Bearer "+testSession)
	r.Header.Set("Origin", "http://other.test")
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	expect(t, w, 403)
	expect(t, f.request("unknown", testSession, nil), 400)
}
func TestCompleteRunAndByteCounts(t *testing.T) {
	f := newFixture(t)
	f.start()
	if !f.cookie.HttpOnly || f.cookie.SameSite != http.SameSiteStrictMode || f.cookie.Path != "/cgi-bin/network-speed-test" {
		t.Fatal("cookie protection")
	}
	l, e := f.s.readLease()
	if e != nil || l.Token == f.cookie.Value || l.Owner == testSession {
		t.Fatal("lease must store hashes")
	}
	expect(t, f.request("ping", testSession, nil), 200)
	w := f.request("download", testSession, nil)
	expect(t, w, 200)
	if w.Body.Len() != 65536 {
		t.Fatal("download size")
	}
	if w.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatal("cache control")
	}
	w = f.request("upload", testSession, bytes.NewReader(make([]byte, 200000)))
	expect(t, w, 200)
	var result struct {
		Bytes int `json:"bytes"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Bytes != 200000 {
		t.Fatal("accepted byte count")
	}
	f.s.inspect = func(context.Context, string) connection {
		return connection{Interface: "phy1-ap0", Band: "2.4 GHz", Channel: 6}
	}
	f.s.cpu = func() map[string]cpuTimes { return map[string]cpuTimes{"cpu0": {Total: 200, Idle: 51}} }
	w = f.request("finish", testSession, nil)
	expect(t, w, 200)
	var end struct {
		Changed bool `json:"connection_changed"`
		Busy    int  `json:"router_cpu_busy"`
	}
	json.Unmarshal(w.Body.Bytes(), &end)
	if !end.Changed || end.Busy != 99 {
		t.Fatal("connection/CPU flags")
	}
	expect(t, f.request("ping", testSession, nil), 409)
	if _, e = os.Stat(filepath.Join(f.s.dir, "run.json")); !os.IsNotExist(e) {
		t.Fatal("lease not removed")
	}
}
func TestExclusiveRunExpiryAndOwnership(t *testing.T) {
	f := newFixture(t)
	f.start()
	expect(t, f.request("start", testSession, nil), 409)
	old := f.cookie
	f.cookie = &http.Cookie{Name: cookieName, Value: strings.Repeat("f", 64)}
	expect(t, f.request("download", testSession, nil), 409)
	f.cookie = old
	r := httptest.NewRequest("POST", "http://router.test/?op=ping", nil)
	r.RemoteAddr = "192.0.2.3:456"
	r.Header.Set("Authorization", "Bearer "+testSession)
	r.AddCookie(f.cookie)
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	expect(t, w, 409)
	f.s.authorize = func(context.Context, string) bool { return true }
	expect(t, f.request("ping", strings.Repeat("b", 32), nil), 409)
	f.now = f.now.Add(runLifetime)
	expect(t, f.request("ping", testSession, nil), 409)
	expect(t, f.request("start", testSession, nil), 200)
}
func TestExpiredLoginAndBusyRequest(t *testing.T) {
	f := newFixture(t)
	f.start()
	f.s.authorize = func(context.Context, string) bool { return false }
	expect(t, f.request("download", testSession, nil), 403)
	f.s.authorize = func(context.Context, string) bool { return true }
	f.s.lock = func(string) (func(), error) { return nil, errors.New("busy") }
	expect(t, f.request("upload", testSession, strings.NewReader("x")), 409)
}
func TestUploadBoundsAndTruncation(t *testing.T) {
	f := newFixture(t)
	f.start()
	for _, n := range []int64{-1, 0, maxUpload + 1} {
		r := httptest.NewRequest("POST", "http://router.test/?op=upload", strings.NewReader("x"))
		r.ContentLength = n
		r.Header.Set("Authorization", "Bearer "+testSession)
		r.Header.Set("Content-Type", "application/octet-stream")
		r.RemoteAddr = "192.0.2.2:12"
		r.AddCookie(f.cookie)
		w := httptest.NewRecorder()
		f.s.ServeHTTP(w, r)
		expect(t, w, 413)
	}
	r := httptest.NewRequest("POST", "http://router.test/?op=upload", strings.NewReader("short"))
	r.ContentLength = 100
	r.Header.Set("Authorization", "Bearer "+testSession)
	r.Header.Set("Content-Type", "application/octet-stream")
	r.RemoteAddr = "192.0.2.2:12"
	r.AddCookie(f.cookie)
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	expect(t, w, 400)
	expect(t, f.request("ping", testSession, strings.NewReader("unexpected")), 400)
}

type failedWriter struct {
	header http.Header
	writes int
}

func (w *failedWriter) Header() http.Header       { return w.header }
func (w *failedWriter) WriteHeader(int)           {}
func (w *failedWriter) Write([]byte) (int, error) { w.writes++; return 0, io.ErrClosedPipe }
func TestDownloadDisconnectReleasesLock(t *testing.T) {
	f := newFixture(t)
	f.start()
	r := httptest.NewRequest("POST", "http://router.test/?op=download", nil)
	r.RemoteAddr = "192.0.2.2:12"
	r.Header.Set("Authorization", "Bearer "+testSession)
	r.AddCookie(f.cookie)
	w := &failedWriter{header: http.Header{}}
	f.s.ServeHTTP(w, r)
	if w.writes != 1 {
		t.Fatal("kept sending after disconnect")
	}
	expect(t, f.request("finish", testSession, nil), 200)
}
func TestCPUCounters(t *testing.T) {
	if cpuPeak(map[string]cpuTimes{"cpu0": {100, 80}}, map[string]cpuTimes{"cpu0": {200, 100}}) != 80 {
		t.Fatal("CPU percentage")
	}
	if cpuPeak(nil, nil) != 0 {
		t.Fatal("missing counters")
	}
}
