package fastreqbenchmark

import (
	"bytes"
	"fmt"
	"net"
	"sync/atomic"
	"testing"

	"github.com/valyala/fasthttp"
	"github.com/wnanbei/fastreq"
)

func Benchmark_Fastreq_Get(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	fastClient := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return acquireFakeServerConn(s), nil
		},
	}
	c := fastreq.NewClientFromFastHTTP(fastClient)

	nn := uint32(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		url := fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1))
		for pb.Next() {
			resp, err := c.Get(url, nil)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.Response.StatusCode() != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.Response.StatusCode())
			}
			if !bytes.Equal(resp.Response.Body(), body) {
				b.Fatalf("unexpected response body: %q. Expected %q", resp.Response.Body(), body)
			}
			fastreq.Release(resp)
		}
	})
}

func Benchmark_Fastreq_Do(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	fastClient := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return acquireFakeServerConn(s), nil
		},
	}
	c := fastreq.NewClientFromFastHTTP(fastClient)

	nn := uint32(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		url := fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1))
		for pb.Next() {
			req := fastreq.NewRequest(fastreq.GET, url)
			resp, err := c.Do(req)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.Response.StatusCode() != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.Response.StatusCode())
			}
			if !bytes.Equal(resp.Response.Body(), body) {
				b.Fatalf("unexpected response body: %q. Expected %q", resp.Response.Body(), body)
			}
			fastreq.Release(resp)
		}
	})
}
