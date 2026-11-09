package main

import (
	"context"
	"net"
	"syscall"
)

func listenLAN(ctx context.Context, device, address string) (net.Listener, error) {
	lc := net.ListenConfig{Control: func(network, address string, c syscall.RawConn) error {
		var bindErr error
		err := c.Control(func(fd uintptr) {
			bindErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, device)
		})
		if err != nil {
			return err
		}
		return bindErr
	}}
	return lc.Listen(ctx, "tcp", address)
}
