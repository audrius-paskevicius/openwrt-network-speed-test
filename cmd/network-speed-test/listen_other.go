//go:build !linux

package main

import (
	"context"
	"errors"
	"net"
)

func listenLAN(context.Context, string, string) (net.Listener, error) {
	return nil, errors.New("native service requires Linux")
}
