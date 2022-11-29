package fastreqbenchmark

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/valyala/fasthttp"
)

func Benchmark_NetHTTP_Get(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return acquireFakeServerConn(s), nil
			},
			MaxIdleConnsPerHost: runtime.GOMAXPROCS(-1),
		},
	}

	nn := uint32(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := c.Get(fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1)))
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.StatusCode)
			}
			respBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("unexpected error when reading response body: %v", err)
			}
			if !bytes.Equal(respBody, body) {
				b.Fatalf("unexpected response body: %q. Expected %q", respBody, body)
			}
		}
	})
}

func Benchmark_NetHTTP_Do(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return acquireFakeServerConn(s), nil
			},
			MaxIdleConnsPerHost: runtime.GOMAXPROCS(-1),
		},
	}

	nn := uint32(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, err := http.NewRequest(fasthttp.MethodGet, fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1)), nil)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			resp, err := c.Do(req)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.StatusCode)
			}
			respBody, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				b.Fatalf("unexpected error when reading response body: %v", err)
			}
			if !bytes.Equal(respBody, body) {
				b.Fatalf("unexpected response body: %q. Expected %q", respBody, body)
			}
		}
	})
}
