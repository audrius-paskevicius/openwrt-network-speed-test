package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUploadConnectionReuseAndPipelining(t *testing.T) {
	f := newFixture(t)
	token := grantForTest(t, f)
	path := filepath.Join(f.s.dir, "grant.json")
	data, _ := os.ReadFile(path)
	var grant accessGrant
	json.Unmarshal(data, &grant)
	grant.Peer = "127.0.0.1"
	data, _ = json.Marshal(grant)
	os.WriteFile(path, data, 0600)
	direct := *f.s
	direct.native = true
	direct.payload = filepath.Join(f.s.dir, "payload")
	server := httptest.NewUnstartedServer(&direct)
	pool := newReuseListener(server.Listener)
	server.Listener = pool
	direct.recycle = pool.recycle
	server.Start()
	defer server.Close()
	req, _ := http.NewRequest("POST", server.URL+"?op=start", bytes.NewReader(make([]byte, payloadSize)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal("start")
	}
	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	receive := func(status int) []byte {
		t.Helper()
		response, err := http.ReadResponse(reader, nil)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != status {
			t.Fatalf("response %d: %s %v", response.StatusCode, body, err)
		}
		return body
	}
	// Tiny upload plus already-buffered next request exercises preservation of
	// pipelined bytes when the socket is handed back to net/http.
	fmt.Fprintf(conn, "POST /?op=upload HTTP/1.1\r\nHost: router.test\r\nAuthorization: Bearer %s\r\nContent-Type: application/octet-stream\r\nContent-Length: 1\r\n\r\nxPOST /?op=ping HTTP/1.1\r\nHost: router.test\r\nAuthorization: Bearer %s\r\nContent-Length: 0\r\n\r\n", token, token)
	if string(receive(200)) != `{"bytes":1}` {
		t.Fatal("upload byte count")
	}
	receive(200)
	for i := 0; i < 3; i++ {
		fmt.Fprintf(conn, "POST /?op=upload HTTP/1.1\r\nHost: router.test\r\nAuthorization: Bearer %s\r\nContent-Type: application/octet-stream\r\nContent-Length: 8192\r\n\r\n%s", token, strings.Repeat("a", 8192))
		if string(receive(200)) != `{"bytes":8192}` {
			t.Fatal("reused upload byte count")
		}
	}
	fmt.Fprintf(conn, "POST /?op=ping HTTP/1.1\r\nHost: router.test\r\nAuthorization: Bearer %s\r\nContent-Length: 0\r\n\r\n", strings.Repeat("f", 64))
	receive(403)
	fmt.Fprintf(conn, "POST /?op=finish HTTP/1.1\r\nHost: router.test\r\nAuthorization: Bearer %s\r\nContent-Length: 0\r\n\r\n", token)
	receive(200)
}

func TestReuseListenerClose(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	pool := newReuseListener(listener)
	if err = pool.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Accept(); err == nil {
		t.Fatal("closed listener accepted a connection")
	}
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	if pool.recycle(a, nil) {
		t.Fatal("closed listener retained a connection")
	}
}
