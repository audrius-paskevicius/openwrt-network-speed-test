package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxUpload = 32 << 20
const maxDownload = 2 << 30
const runLifetime = 90 * time.Second
const cookieName = "network_speed_test_run"

type connection struct {
	Radio     string `json:"radio,omitempty"`
	Interface string `json:"interface,omitempty"`
	Band      string `json:"band,omitempty"`
	Channel   int    `json:"channel,omitempty"`
}
type cpuTimes struct{ Total, Idle, User, System, IRQ, SoftIRQ uint64 }
type measurements struct {
	DownloadBytes int64   `json:"download_bytes"`
	DownloadMS    float64 `json:"download_ms"`
	UploadBytes   int64   `json:"upload_bytes"`
	UploadMS      float64 `json:"upload_ms"`
	LatencyMS     float64 `json:"latency_ms"`
}
type resultRecord struct {
	Transport    string                    `json:"transport"`
	Time         time.Time                 `json:"time"`
	Start        connection                `json:"start_connection"`
	End          connection                `json:"end_connection"`
	Changed      bool                      `json:"connection_changed"`
	CPU          int                       `json:"router_cpu_busy"`
	CPUBreakdown map[string]map[string]int `json:"cpu_breakdown"`
	DownloadMbps float64                   `json:"download_mbps"`
	UploadMbps   float64                   `json:"upload_mbps"`
	Measurements measurements              `json:"measurements"`
}
type lease struct {
	Token      string              `json:"token_hash"`
	Owner      string              `json:"owner_hash"`
	Peer       string              `json:"peer"`
	Expires    time.Time           `json:"expires"`
	Connection connection          `json:"connection"`
	CPU        map[string]cpuTimes `json:"cpu"`
}
type server struct {
	native         bool
	payload        string
	dir            string
	authorize      func(context.Context, string) bool
	inspect        func(context.Context, string) connection
	cpu            func() map[string]cpuTimes
	lock           func(string) (func(), error)
	now            func() time.Time
	streamLimit    int64
	streamDuration time.Duration
}

func digest(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func validHex(s string, n int) bool {
	if len(s) != n || strings.Trim(s, "0") == "" {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func command(ctx context.Context, program string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, program, args...).Output()
}
func authorized(ctx context.Context, sid string) bool {
	request, _ := json.Marshal(map[string]string{"ubus_rpc_session": sid, "scope": "network-speed-test", "object": "test", "function": "run"})
	b, e := command(ctx, "/bin/ubus", "-S", "call", "session", "access", string(request))
	var result struct {
		Access bool `json:"access"`
	}
	return e == nil && json.Unmarshal(b, &result) == nil && result.Access
}
func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}
func peer(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func (s *server) readLease() (lease, error) {
	var l lease
	b, e := os.ReadFile(filepath.Join(s.dir, "run.json"))
	if e == nil {
		e = json.Unmarshal(b, &l)
	}
	return l, e
}
func (s *server) saveLease(l lease) error {
	b, e := json.Marshal(l)
	if e != nil {
		return e
	}
	path := filepath.Join(s.dir, "run.new")
	if e = os.WriteFile(path, b, 0600); e != nil {
		return e
	}
	return os.Rename(path, filepath.Join(s.dir, "run.json"))
}
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		fail(w, 405, "Use Start on the LuCI Speed Test page.")
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != scheme+"://"+r.Host {
		fail(w, 403, "Use the Speed Test page on this router.")
		return
	}
	op := r.URL.Query().Get("op")
	sid, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	capability := sid
	allowed := false
	if s.native {
		var grant accessGrant
		if data, err := os.ReadFile(filepath.Join(s.dir, "grant.json")); err == nil && json.Unmarshal(data, &grant) == nil && validHex(capability, 64) && grant.Token == digest(capability) && grant.Peer == peer(r) && grant.Expires.After(s.now()) {
			sid = grant.Session
			allowed = op != "prepare" && (op != "start" || s.authorize(r.Context(), sid))
		}
	} else {
		allowed = validHex(sid, 32) && s.authorize(r.Context(), sid)
	}
	if !ok || !allowed {
		if s.native {
			fail(w, 403, "Test access expired. Return to the router UI and press Start.")
			return
		}
		fail(w, 403, "Your router login has expired or does not allow speed tests. Sign in again.")
		return
	}
	var measured *measurements
	if op != "prepare" && op != "connection" && op != "start" && op != "ping" && op != "download" && op != "upload" && op != "finish" {
		fail(w, 400, "Unknown test operation.")
		return
	}
	if s.native && op == "start" {
		if r.ContentLength != payloadSize || r.Header.Get("Content-Type") != "application/octet-stream" {
			fail(w, 400, "A client-generated data sample is required.")
			return
		}
	} else if op == "upload" {
		if r.ContentLength < 1 || r.ContentLength > maxUpload {
			fail(w, 413, "Upload must contain 1 byte to 32 MiB.")
			return
		}
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			fail(w, 415, "Invalid test data.")
			return
		}
	} else if op == "finish" && r.ContentLength != 0 {
		if r.ContentLength < 1 || r.ContentLength > 1024 || r.Header.Get("Content-Type") != "application/json" {
			fail(w, 400, "Invalid test measurements.")
			return
		}
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1025))
		decoder.DisallowUnknownFields()
		var m measurements
		if err := decoder.Decode(&m); err != nil || m.DownloadBytes < 1 || m.DownloadBytes > maxDownload || m.UploadBytes < 1 || m.UploadBytes > 16*maxDownload || m.DownloadMS <= 0 || m.DownloadMS > 90000 || m.UploadMS <= 0 || m.UploadMS > 90000 || m.LatencyMS < 0 || m.LatencyMS > 18000 {
			fail(w, 400, "Invalid test measurements.")
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			fail(w, 400, "Invalid test measurements.")
			return
		}
		measured = &m
	} else if r.ContentLength != 0 {
		fail(w, 400, "Unexpected request body.")
		return
	}
	if op == "connection" {
		reply(w, 200, map[string]any{"connection": s.inspect(r.Context(), peer(r))})
		return
	}
	if e := os.MkdirAll(s.dir, 0700); e != nil {
		fail(w, 503, "Cannot prepare test.")
		return
	}
	unlock, e := s.lock(filepath.Join(s.dir, "request.lock"))
	if e != nil {
		fail(w, 409, "Another transfer is running. Try again shortly.")
		return
	}
	releaseLock := unlock
	unlock = func() {
		if releaseLock != nil {
			releaseLock()
			releaseLock = nil
		}
	}
	defer unlock()
	if s.native {
		var current accessGrant
		data, err := os.ReadFile(filepath.Join(s.dir, "grant.json"))
		if err != nil || json.Unmarshal(data, &current) != nil || current.Token != digest(capability) || current.Session != sid {
			fail(w, 403, "Test access changed. Return to the router UI and press Start.")
			return
		}
		if op == "start" {
			current.Expires = s.now().Add(30 * time.Minute)
			data, _ = json.Marshal(current)
			if os.WriteFile(filepath.Join(s.dir, "grant.new"), data, 0600) != nil || os.Rename(filepath.Join(s.dir, "grant.new"), filepath.Join(s.dir, "grant.json")) != nil {
				fail(w, 503, "Cannot renew local test access.")
				return
			}
		}
	}
	l, leaseError := s.readLease()
	if op == "start" || op == "prepare" {
		if leaseError == nil && l.Expires.After(s.now()) {
			fail(w, 409, "Another test is running. Wait a moment, then try again.")
			return
		}
		if leaseError != nil && !errors.Is(leaseError, os.ErrNotExist) {
			fail(w, 503, "Cannot read test state.")
			return
		}
		token := make([]byte, 32)
		if _, e = rand.Read(token); e != nil {
			fail(w, 503, "Cannot prepare test.")
			return
		}
		value := hex.EncodeToString(token)
		if op == "prepare" {
			grant := accessGrant{Token: digest(value), Session: sid, Peer: peer(r), Expires: s.now().Add(30 * time.Minute)}
			data, _ := json.Marshal(grant)
			path := filepath.Join(s.dir, "grant.new")
			if os.WriteFile(path, data, 0600) != nil || os.Rename(path, filepath.Join(s.dir, "grant.json")) != nil {
				fail(w, 503, "Cannot prepare local test access.")
				return
			}
			reply(w, 200, map[string]any{"token": value})
			return
		}
		if s.native {
			value = capability
			if err := storePayload(s.payload, r.Body); err != nil {
				fail(w, 400, "The test data sample did not arrive completely.")
				return
			}
		}
		l = lease{Token: digest(value), Owner: digest(sid), Peer: peer(r), Expires: s.now().Add(runLifetime), Connection: s.inspect(r.Context(), peer(r)), CPU: s.cpu()}
		if e = s.saveLease(l); e != nil {
			fail(w, 503, "Cannot prepare test.")
			return
		}
		if !s.native {
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/cgi-bin/network-speed-test", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: 90})
		}
		reply(w, 200, map[string]any{"connection": l.Connection})
		return
	}
	c, e := r.Cookie(cookieName)
	if s.native {
		c = &http.Cookie{Value: capability}
		e = nil
	}
	if leaseError != nil || e != nil || !validHex(c.Value, 64) || !l.Expires.After(s.now()) || l.Token != digest(c.Value) || l.Owner != digest(sid) || l.Peer != peer(r) {
		fail(w, 409, "This test has expired. Press Start to try again.")
		return
	}
	switch op {
	case "ping":
		reply(w, 200, map[string]bool{"ok": true})
	case "download":
		if s.native {
			s.sendPayload(w, r)
			return
		}
		buffer := make([]byte, 64<<10)
		if _, e = rand.Read(buffer); e != nil {
			fail(w, 503, "Cannot prepare test data.")
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		until := s.now().Add(s.streamDuration)
		for sent := int64(0); sent < s.streamLimit && s.now().Before(until); {
			if r.Context().Err() != nil {
				return
			}
			chunk := buffer
			if remaining := s.streamLimit - sent; remaining < int64(len(chunk)) {
				chunk = chunk[:remaining]
			}
			n, err := w.Write(chunk)
			if err != nil || n != len(chunk) {
				return
			}
			sent += int64(n)
		}
	case "upload":
		if s.native {
			receivePayload(w, r, unlock)
			return
		}
		// Read exactly the declared length; do not retain, echo, or write payloads.
		n, err := io.CopyBuffer(io.Discard, io.LimitReader(r.Body, r.ContentLength), make([]byte, 64<<10))
		if err != nil || n != r.ContentLength {
			fail(w, 400, "Upload did not complete.")
			return
		}
		reply(w, 200, map[string]int64{"bytes": n})
	case "finish":
		end := s.inspect(r.Context(), peer(r))
		changed := l.Connection.Interface != "" && end != l.Connection
		cpuEnd := s.cpu()
		peak := cpuPeak(l.CPU, cpuEnd)
		saved := false
		if measured != nil {
			record := resultRecord{Transport: scheme, Time: s.now().UTC(), Start: l.Connection, End: end, Changed: changed, CPU: peak, Measurements: *measured, DownloadMbps: float64(measured.DownloadBytes) * 8 / measured.DownloadMS / 1000, UploadMbps: float64(measured.UploadBytes) * 8 / measured.UploadMS / 1000}
			record.CPUBreakdown = cpuBreakdown(l.CPU, cpuEnd)
			data, err := json.Marshal(record)
			if err == nil {
				path := filepath.Join(s.dir, "last-result.new")
				if os.WriteFile(path, data, 0600) == nil {
					saved = os.Rename(path, filepath.Join(s.dir, "last-result.json")) == nil
				}
			}
		}
		if e = os.Remove(filepath.Join(s.dir, "run.json")); e != nil {
			fail(w, 503, "Cannot finish test.")
			return
		}
		if !s.native {
			http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/cgi-bin/network-speed-test", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		}
		reply(w, 200, map[string]any{"connection": end, "connection_changed": changed, "router_cpu_busy": peak, "result_saved": saved})
	}
}
func readCPU() map[string]cpuTimes {
	b, e := os.ReadFile("/proc/stat")
	if e != nil {
		return nil
	}
	out := map[string]cpuTimes{}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) < 5 || !strings.HasPrefix(f[0], "cpu") || f[0] == "cpu" {
			continue
		}
		var c cpuTimes
		for i := 1; i < len(f) && i <= 8; i++ {
			v, _ := strconv.ParseUint(f[i], 10, 64)
			c.Total += v
			switch i {
			case 1, 2:
				c.User += v
			case 3:
				c.System = v
			case 6:
				c.IRQ = v
			case 7:
				c.SoftIRQ = v
			}
			if i == 4 || i == 5 {
				c.Idle += v
			}
		}
		out[f[0]] = c
	}
	return out
}
func cpuPeak(before, after map[string]cpuTimes) int {
	peak := 0
	for core, b := range before {
		a, ok := after[core]
		if !ok || a.Total <= b.Total || a.Idle < b.Idle {
			continue
		}
		total, idle := a.Total-b.Total, a.Idle-b.Idle
		if idle > total {
			continue
		}
		p := int((total - idle) * 100 / total)
		if p > peak {
			peak = p
		}
	}
	return peak
}
func cpuBreakdown(before, after map[string]cpuTimes) map[string]map[string]int {
	out := map[string]map[string]int{}
	for core, b := range before {
		a, ok := after[core]
		if !ok || a.Total <= b.Total {
			continue
		}
		total := a.Total - b.Total
		percent := func(start, end uint64) int {
			if end < start || end-start > total {
				return 0
			}
			return int((end - start) * 100 / total)
		}
		out[core] = map[string]int{"user_percent": percent(b.User, a.User), "system_percent": percent(b.System, a.System), "irq_percent": percent(b.IRQ, a.IRQ), "softirq_percent": percent(b.SoftIRQ, a.SoftIRQ)}
	}
	return out
}
func radioForInterface(data []byte, iface string) string {
	var radios map[string]struct {
		Interfaces []struct {
			Ifname string `json:"ifname"`
		} `json:"interfaces"`
	}
	if json.Unmarshal(data, &radios) != nil {
		return ""
	}
	for name, radio := range radios {
		for _, network := range radio.Interfaces {
			if network.Ifname == iface {
				return name
			}
		}
	}
	return ""
}
func inspect(ctx context.Context, address string) connection {
	// Only inspect the requesting client's directly connected neighbour and association.
	if net.ParseIP(address) == nil {
		return connection{}
	}
	b, e := command(ctx, "/sbin/ip", "neigh", "show", "to", address)
	if e != nil {
		return connection{}
	}
	fields := strings.Fields(string(b))
	mac := ""
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "lladdr" {
			mac = strings.ToLower(fields[i+1])
			break
		}
	}
	if _, e = net.ParseMAC(mac); e != nil {
		return connection{}
	}
	b, e = command(ctx, "/bin/ubus", "-S", "list", "hostapd.*")
	if e != nil {
		return connection{}
	}
	for _, object := range strings.Fields(string(b)) {
		data, err := command(ctx, "/bin/ubus", "-S", "call", object, "get_clients", "{}")
		if err != nil {
			continue
		}
		var result struct {
			Frequency int                        `json:"freq"`
			Clients   map[string]json.RawMessage `json:"clients"`
		}
		if json.Unmarshal(data, &result) != nil {
			continue
		}
		found := false
		for key := range result.Clients {
			if strings.EqualFold(key, mac) {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		iface := strings.TrimPrefix(object, "hostapd.")
		freq := result.Frequency
		if freq == 0 {
			info, err := command(ctx, "/usr/sbin/iw", "dev", iface, "info")
			if err == nil {
				words := strings.Fields(string(info))
				for i := 0; i+2 < len(words); i++ {
					if words[i] == "channel" {
						freq, _ = strconv.Atoi(strings.TrimPrefix(words[i+2], "("))
						break
					}
				}
			}
		}
		c := connection{Interface: iface}
		if status, err := command(ctx, "/bin/ubus", "-S", "call", "network.wireless", "status"); err == nil {
			c.Radio = radioForInterface(status, iface)
		}
		switch {
		case freq == 2484:
			c.Band = "2.4 GHz"
			c.Channel = 14
		case freq >= 2412 && freq <= 2472:
			c.Band = "2.4 GHz"
			c.Channel = (freq - 2407) / 5
		case freq >= 5000 && freq < 5925:
			c.Band = "5 GHz"
			c.Channel = (freq - 5000) / 5
		case freq >= 5955 && freq <= 7115:
			c.Band = "6 GHz"
			c.Channel = (freq - 5950) / 5
		}
		return c
	}
	return connection{}
}
func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if len(os.Args) != 3 {
			os.Exit(2)
		}
		if err := serveNative(os.Args[2]); err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		return
	}
	// A client that disconnects or stops uploading cannot leave a CGI process behind.
	timer := time.AfterFunc(20*time.Second, func() { os.Exit(1) })
	defer timer.Stop()
	s := &server{dir: "/var/run/network-speed-test", authorize: authorized, inspect: inspect, cpu: readCPU, lock: lockFile, now: time.Now, streamLimit: maxDownload, streamDuration: 13 * time.Second}
	if cgi.Serve(s) != nil {
		os.Exit(1)
	}
}
