package main

import (
	"io"
	"net"
	"sync"
	"syscall"
)

// A hijacked upload socket is returned to net/http after its body and response
// have finished. This keeps TCP warm while allowing splice to consume the body.
type reuseListener struct {
	net.Listener
	queue  chan net.Conn
	done   chan struct{}
	errors chan error
	mu     sync.Mutex
	closed bool
}

func newReuseListener(listener net.Listener) *reuseListener {
	l := &reuseListener{Listener: listener, queue: make(chan net.Conn, 64), done: make(chan struct{}), errors: make(chan error, 1)}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case l.errors <- err:
				default:
				}
				return
			}
			if !l.recycle(conn, nil) {
				conn.Close()
			}
		}
	}()
	return l
}

func (l *reuseListener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, net.ErrClosed
	case err := <-l.errors:
		return nil, err
	case conn := <-l.queue:
		return conn, nil
	}
}
func (l *reuseListener) recycle(conn net.Conn, prefix []byte) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	if len(prefix) > 0 {
		conn = &prefixedConn{Conn: conn, prefix: prefix}
	}
	select {
	case l.queue <- conn:
		return true
	default:
		return false
	}
}
func (l *reuseListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	close(l.done)
	err := l.Listener.Close()
	for {
		select {
		case conn := <-l.queue:
			conn.Close()
		default:
			return err
		}
	}
}

// Preserve any already-buffered bytes, including pipelined request headers.
// Normal sequential browser uploads return the unwrapped TCP connection.
type prefixedConn struct {
	net.Conn
	prefix []byte
}

func (c *prefixedConn) Read(p []byte) (int, error) {
	if len(c.prefix) > 0 {
		n := copy(p, c.prefix)
		c.prefix = c.prefix[n:]
		return n, nil
	}
	return c.Conn.Read(p)
}
func (c *prefixedConn) SyscallConn() (syscall.RawConn, error) {
	if len(c.prefix) != 0 {
		return nil, syscall.EINVAL
	}
	if raw, ok := c.Conn.(syscall.Conn); ok {
		return raw.SyscallConn()
	}
	return nil, io.ErrNoProgress
}
