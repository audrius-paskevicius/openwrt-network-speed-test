//go:build linux

package main

import (
	"path/filepath"
	"testing"
)

func TestRealFileLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	unlock, e := lockFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if second, e := lockFile(path); e == nil {
		second()
		t.Fatal("concurrent lock succeeded")
	}
	unlock()
	unlock, e = lockFile(path)
	if e != nil {
		t.Fatal(e)
	}
	unlock()
}
