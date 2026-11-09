package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type accessGrant struct {
	Token   string    `json:"token_hash"`
	Session string    `json:"session"`
	Peer    string    `json:"peer"`
	Expires time.Time `json:"expires"`
}

const nativePort = "18080"
const payloadSize = 1 << 20

func preparePayload(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "payload")
	return path, nil
}

func storePayload(path string, source io.Reader, size int64) error {
	f, err := os.OpenFile(path+".new", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.CopyN(f, source, size)
	closeErr := f.Close()
	if err != nil {
		os.Remove(path + ".new")
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(path+".new", path)
}

func (s *server) sendPayload(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(s.payload)
	if err != nil {
		fail(w, 503, "Cannot open test data.")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		fail(w, 503, "Cannot read test data.")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(s.streamDuration))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(s.streamLimit, 10))
	// A known length keeps HTTP unchunked. Go's TCP/file path uses sendfile on Linux.
	for remaining := s.streamLimit; remaining > 0; {
		if r.Context().Err() != nil {
			return
		}
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return
		}
		n := min(remaining, info.Size())
		copied, err := io.CopyN(w, f, n)
		remaining -= copied
		if err != nil {
			return
		}
	}
}

func receivePayload(w http.ResponseWriter, r *http.Request, release func(), recycle func(net.Conn, []byte) bool) {
	sink, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		fail(w, 503, "Cannot receive test data.")
		return
	}
	defer sink.Close()
	h, ok := w.(http.Hijacker)
	if !ok {
		fail(w, 503, "Direct transport unavailable.")
		return
	}
	conn, buffer, err := h.Hijack()
	if err != nil {
		return
	}
	reused := false
	defer func() {
		if !reused {
			conn.Close()
		}
	}()
	_ = conn.SetDeadline(time.Now().Add(18 * time.Second))
	// Drain only bytes already buffered with the request headers, then let Go use
	// Linux splice from the TCP socket through a kernel pipe into /dev/null.
	first := min(int64(buffer.Reader.Buffered()), r.ContentLength)
	copied, err := io.CopyN(sink, buffer.Reader, first)
	if err == nil {
		var n int64
		n, err = io.CopyN(sink, conn, r.ContentLength-copied)
		copied += n
	}
	status := 200
	var body []byte
	if err != nil || copied != r.ContentLength {
		status = 400
		body = []byte(`{"error":"Upload did not complete."}`)
	} else {
		body, _ = json.Marshal(map[string]int64{"bytes": copied})
	}
	// Release before publishing the response so the next upload can start as
	// soon as the browser receives confirmation of this one.
	release()
	connection := "close"
	if status == 200 && !r.Close && r.ProtoMajor == 1 && r.ProtoMinor == 1 && recycle != nil {
		connection = "keep-alive"
	}
	fmt.Fprintf(buffer.Writer, "HTTP/1.1 %d %s\r\nContent-Type: application/json\r\nCache-Control: no-store\r\nX-Content-Type-Options: nosniff\r\nConnection: %s\r\nContent-Length: %d\r\n\r\n", status, http.StatusText(status), connection, len(body))
	buffer.Writer.Write(body)
	if buffer.Writer.Flush() == nil && connection == "keep-alive" {
		pending, _ := buffer.Reader.Peek(buffer.Reader.Buffered())
		reused = recycle(conn, append([]byte(nil), pending...))
	}
}

func nativeHandler(s *server, assets string) http.Handler {
	static := http.FileServer(http.Dir("/www/luci-static"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		if r.URL.Path == "/api" {
			s.ServeHTTP(w, r)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		names := map[string]string{"/test/": "native.html", "/native.js": "native.js", "/engine.js": "engine.js", "/style.css": "style.css"}
		if name, ok := names[r.URL.Path]; ok {
			http.ServeFile(w, r, filepath.Join(assets, name))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/luci-static/") {
			http.StripPrefix("/luci-static", static).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

func serveNative(device string) error {
	if device == "" {
		return fmt.Errorf("LAN device is required")
	}
	dir := "/var/run/network-speed-test"
	path, err := preparePayload(dir)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	s := &server{native: true, payload: path, dir: dir, authorize: authorized, inspect: inspect, cpu: readCPU, lock: lockFile, now: time.Now, streamLimit: maxDownload, streamDuration: 13 * time.Second}
	listener, err := listenLAN(context.Background(), device, net.JoinHostPort("", nativePort))
	if err != nil {
		return err
	}
	pool := newReuseListener(listener)
	s.recycle = pool.recycle
	httpServer := &http.Server{Handler: nativeHandler(s, "/usr/share/network-speed-test"), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 8192}
	return httpServer.Serve(pool)
}
