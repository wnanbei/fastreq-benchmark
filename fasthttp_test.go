package fastreqbenchmark

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func Benchmark_FastHTTP_Get(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return acquireFakeServerConn(s), nil
		},
	}

	nn := uint32(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		url := fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1))
		for pb.Next() {
			var bodyBuf []byte
			statusCode, bodyBuf, err := c.Get(bodyBuf[:0], url)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if statusCode != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d", statusCode)
			}
			if !bytes.Equal(bodyBuf, body) {
				b.Fatalf("unexpected response body: %q. Expected %q", bodyBuf, body)
			}
		}
	})
}

func Benchmark_FastHTTP_Do(b *testing.B) {
	body := []byte("012345678912")
	s := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(body), body))
	c := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return acquireFakeServerConn(s), nil
		},
		MaxConnsPerHost: runtime.GOMAXPROCS(-1),
	}

	nn := uint32(0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := fasthttp.AcquireRequest()
			resp := fasthttp.AcquireResponse()
			req.Header.SetRequestURI(fmt.Sprintf("http://foobar%d.com/aaa/bbb", atomic.AddUint32(&nn, 1)))

			if err := c.Do(req, resp); err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.Header.StatusCode() != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d", resp.Header.StatusCode())
			}
			if !bytes.Equal(resp.Body(), body) {
				b.Fatalf("unexpected response body: %q. Expected %q", resp.Body(), body)
			}
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}
	})
}

func fasthttpEchoHandler(ctx *fasthttp.RequestCtx) {
	ctx.Success("text/plain", ctx.RequestURI())
}

func nethttpEchoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(fasthttp.HeaderContentType, "text/plain")
	w.Write([]byte(r.RequestURI)) //nolint:errcheck
}

func Benchmark_FastHTTP_GetTCP_1(b *testing.B) {
	benchmark_FastHTTP_GetTCP(b, 1)
}

func Benchmark_FastHTTP_GetTCP_10(b *testing.B) {
	benchmark_FastHTTP_GetTCP(b, 10)
}

func benchmark_FastHTTP_GetTCP(b *testing.B, parallelism int) {
	addr := "127.0.0.1:8543"

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		b.Fatalf("cannot listen %q: %v", addr, err)
	}

	ch := make(chan struct{})
	go func() {
		if err := fasthttp.Serve(ln, fasthttpEchoHandler); err != nil {
			b.Errorf("error when serving requests: %v", err)
		}
		close(ch)
	}()

	c := &fasthttp.Client{
		MaxConnsPerHost: runtime.GOMAXPROCS(-1) * parallelism,
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://" + addr + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var buf []byte
		for pb.Next() {
			statusCode, body, err := c.Get(buf, url)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if statusCode != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", statusCode, fasthttp.StatusOK)
			}
			if string(body) != requestURI {
				b.Fatalf("unexpected response %q. Expecting %q", body, requestURI)
			}
			buf = body
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func Benchmark_FastHTTP_GetInmemory_1(b *testing.B) {
	benchmark_FastHTTP_GetInmemory(b, 1)
}

func Benchmark_FastHTTP_GetInmemory_10(b *testing.B) {
	benchmark_FastHTTP_GetInmemory(b, 10)
}

func Benchmark_FastHTTP_GetInmemory_1000(b *testing.B) {
	benchmark_FastHTTP_GetInmemory(b, 1000)
}

func Benchmark_FastHTTP_GetInmemory_10K(b *testing.B) {
	benchmark_FastHTTP_GetInmemory(b, 10000)
}

func benchmark_FastHTTP_GetInmemory(b *testing.B, parallelism int) {
	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := fasthttp.Serve(ln, fasthttpEchoHandler); err != nil {
			b.Errorf("error when serving requests: %v", err)
		}
		close(ch)
	}()

	c := &fasthttp.Client{
		MaxConnsPerHost: runtime.GOMAXPROCS(-1) * parallelism,
		Dial:            func(addr string) (net.Conn, error) { return ln.Dial() },
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var buf []byte
		for pb.Next() {
			statusCode, body, err := c.Get(buf, url)
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if statusCode != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", statusCode, fasthttp.StatusOK)
			}
			if string(body) != requestURI {
				b.Fatalf("unexpected response %q. Expecting %q", body, requestURI)
			}
			buf = body
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func Benchmark_FastHTTP_DoTimeout_BigResponse_Inmemory_1(b *testing.B) {
	benchmark_FastHTTP_DoTimeout_BigResponse_Inmemory(b, 1)
}

func Benchmark_FastHTTP_DoTimeout_BigResponse_Inmemory_10(b *testing.B) {
	benchmark_FastHTTP_DoTimeout_BigResponse_Inmemory(b, 10)
}

func benchmark_FastHTTP_DoTimeout_BigResponse_Inmemory(b *testing.B, parallelism int) {
	bigResponse := createFixedBody(1024 * 1024)
	h := func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/plain")
		ctx.Write(bigResponse) //nolint:errcheck
	}

	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := fasthttp.Serve(ln, h); err != nil {
			b.Errorf("error when serving requests: %v", err)
		}
		close(ch)
	}()

	c := &fasthttp.Client{
		MaxConnsPerHost: runtime.GOMAXPROCS(-1) * parallelism,
		Dial:            func(addr string) (net.Conn, error) { return ln.Dial() },
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var req fasthttp.Request
		req.SetRequestURI(url)
		var resp fasthttp.Response
		for pb.Next() {
			if err := c.DoTimeout(&req, &resp, 5*time.Second); err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode() != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
			}
			body := resp.Body()
			if !bytes.Equal(bigResponse, body) {
				b.Fatalf("unexpected response %q. Expecting %q", body, bigResponse)
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}

func Benchmark_FastHTTP_PipelineClient_1(b *testing.B) {
	benchmark_FastHTTP_PipelineClient(b, 1)
}

func Benchmark_FastHTTP_PipelineClient_10(b *testing.B) {
	benchmark_FastHTTP_PipelineClient(b, 10)
}

func Benchmark_FastHTTP_PipelineClient_100(b *testing.B) {
	benchmark_FastHTTP_PipelineClient(b, 100)
}

func Benchmark_FastHTTP_PipelineClient_1000(b *testing.B) {
	benchmark_FastHTTP_PipelineClient(b, 1000)
}

func benchmark_FastHTTP_PipelineClient(b *testing.B, parallelism int) {
	h := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("foobar") //nolint:errcheck
	}
	ln := fasthttputil.NewInmemoryListener()

	ch := make(chan struct{})
	go func() {
		if err := fasthttp.Serve(ln, h); err != nil {
			b.Errorf("error when serving requests: %v", err)
		}
		close(ch)
	}()

	maxConns := runtime.GOMAXPROCS(-1)
	c := &fasthttp.PipelineClient{
		Dial:               func(addr string) (net.Conn, error) { return ln.Dial() },
		ReadBufferSize:     1024 * 1024,
		WriteBufferSize:    1024 * 1024,
		MaxConns:           maxConns,
		MaxPendingRequests: parallelism * maxConns,
	}

	requestURI := "/foo/bar?baz=123"
	url := "http://unused.host" + requestURI
	b.SetParallelism(parallelism)
	b.RunParallel(func(pb *testing.PB) {
		var req fasthttp.Request
		req.SetRequestURI(url)
		var resp fasthttp.Response
		for pb.Next() {
			if err := c.Do(&req, &resp); err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode() != fasthttp.StatusOK {
				b.Fatalf("unexpected status code: %d. Expecting %d", resp.StatusCode(), fasthttp.StatusOK)
			}
			body := resp.Body()
			if string(body) != "foobar" {
				b.Fatalf("unexpected response %q. Expecting %q", body, "foobar")
			}
		}
	})

	ln.Close()
	select {
	case <-ch:
	case <-time.After(time.Second):
		b.Fatalf("server wasn't stopped")
	}
}
