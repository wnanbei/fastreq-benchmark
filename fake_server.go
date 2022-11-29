package fastreqbenchmark

import (
	"net"
	"sync"
)

var fakeClientConnPool sync.Pool

type fakeClientConn struct {
	net.Conn
	s  []byte
	n  int
	ch chan struct{}
}

func (c *fakeClientConn) Write(b []byte) (int, error) {
	c.ch <- struct{}{}
	return len(b), nil
}

func (c *fakeClientConn) Read(b []byte) (int, error) {
	if c.n == 0 {
		// wait for request :)
		<-c.ch
	}
	n := 0
	for len(b) > 0 {
		if c.n == len(c.s) {
			c.n = 0
			return n, nil
		}
		n = copy(b, c.s[c.n:])
		c.n += n
		b = b[n:]
	}
	return n, nil
}

func (c *fakeClientConn) Close() error {
	releaseFakeServerConn(c)
	return nil
}

func (c *fakeClientConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   []byte{1, 2, 3, 4},
		Port: 8765,
	}
}

func (c *fakeClientConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   []byte{1, 2, 3, 4},
		Port: 8765,
	}
}

func releaseFakeServerConn(c *fakeClientConn) {
	c.n = 0
	fakeClientConnPool.Put(c)
}

func acquireFakeServerConn(s []byte) *fakeClientConn {
	v := fakeClientConnPool.Get()
	if v == nil {
		c := &fakeClientConn{
			s:  s,
			ch: make(chan struct{}, 1),
		}
		return c
	}
	return v.(*fakeClientConn)
}

func createFixedBody(bodySize int) []byte {
	var b []byte
	for i := 0; i < bodySize; i++ {
		b = append(b, byte(i%10)+'0')
	}
	return b
}
