package main

import (
	"bytes"
	"context"
	"encoding/json"
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

func grantForTest(t *testing.T, f *fixture) string {
	t.Helper()
	w := f.request("prepare", testSession, nil)
	expect(t, w, 200)
	var body struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if !validHex(body.Token, 64) || strings.Contains(w.Body.String(), testSession) || len(w.Result().Cookies()) != 0 {
		t.Fatal("handoff must expose only a scoped token")
	}
	return body.Token
}

func TestScopedGrant(t *testing.T) {
	f := newFixture(t)
	token := grantForTest(t, f)
	f.s.native = true
	f.s.payload = filepath.Join(f.s.dir, "payload")
	expect(t, f.request("connection", token, nil), 200)
	expect(t, f.request("connection", testSession, nil), 403)
	expect(t, f.request("prepare", token, nil), 403)
	expect(t, f.request("connection", strings.Repeat("f", 64), nil), 403)
	expect(t, f.request("start", token, nil), 400)
	f.now = f.now.Add(31 * time.Minute)
	expect(t, f.request("connection", token, nil), 403)
}

func TestNativeTransfersAndRevocation(t *testing.T) {
	f := newFixture(t)
	control := httptest.NewServer(f.s)
	defer control.Close()
	request := func(base, op, token string, body []byte) *http.Response {
		t.Helper()
		r, _ := http.NewRequest("POST", base+"?op="+op, bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			r.Header.Set("Content-Type", "application/octet-stream")
		}
		response, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	response := request(control.URL, "prepare", testSession, nil)
	var handoff struct {
		Token string `json:"token"`
	}
	json.NewDecoder(response.Body).Decode(&handoff)
	response.Body.Close()
	direct := *f.s
	direct.native = true
	direct.payload = filepath.Join(f.s.dir, "payload")
	native := httptest.NewServer(&direct)
	defer native.Close()
	sample := bytes.Repeat([]byte{17, 91, 204, 8}, payloadSize/4)
	response = request(native.URL, "start", handoff.Token, sample)
	if response.StatusCode != 200 {
		data, _ := io.ReadAll(response.Body)
		t.Fatal(string(data))
	}
	response.Body.Close()
	response = request(control.URL, "prepare", testSession, nil)
	if response.StatusCode != 409 {
		t.Fatal("handoff invalidated an active run")
	}
	response.Body.Close()
	response = request(native.URL, "download", handoff.Token, nil)
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || !bytes.Equal(data, sample) || response.ContentLength != int64(len(sample)) {
		t.Fatalf("download mismatch: %d %v", len(data), err)
	}
	upload := bytes.Repeat([]byte{81}, 200000)
	response = request(native.URL, "upload", handoff.Token, upload)
	var received struct {
		Bytes int `json:"bytes"`
	}
	json.NewDecoder(response.Body).Decode(&received)
	response.Body.Close()
	if response.StatusCode != 200 || received.Bytes != len(upload) {
		t.Fatal("upload confirmation")
	}
	response = request(native.URL, "finish", handoff.Token, nil)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("finish")
	}
	direct.authorize = func(_ context.Context, _ string) bool { return false }
	response = request(native.URL, "start", handoff.Token, sample)
	response.Body.Close()
	if response.StatusCode != 403 {
		t.Fatal("revoked router login can start another run")
	}
}

func TestGrantBoundToClient(t *testing.T) {
	f := newFixture(t)
	token := grantForTest(t, f)
	f.s.native = true
	r := httptest.NewRequest("POST", "http://router.test/api?op=connection", nil)
	r.RemoteAddr = net.JoinHostPort("192.0.2.99", "1234")
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	expect(t, w, 403)
}

func TestIncompleteSampleDoesNotReplacePayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payload")
	os.WriteFile(path, []byte("existing"), 0600)
	if storePayload(path, strings.NewReader("short")) == nil {
		t.Fatal("accepted truncated sample")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "existing" {
		t.Fatal("partial sample replaced payload")
	}
}
