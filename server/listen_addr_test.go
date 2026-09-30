package server

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// freePort asks the kernel for an unused loopback port and releases it.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	_, port, _ := net.SplitHostPort(l.Addr().String())
	_ = l.Close()
	return port
}

func startTestListener(t *testing.T, cfg *ServerConfig) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 2 * time.Second
	}
	s := &Server{cfg: cfg, router: router}
	if err := s.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = s.Stop(context.Background()) })
	return s
}

// TestDGS113_DefaultBindIsLoopback pins the default: a gateway started with no
// bind host configured must listen on loopback only. The pre-fix server built
// its listener as ":" + port — every interface — so a laptop or a bare binary
// answered on the LAN (verified 2026-08-05 at 192.168.1.180:7341).
func TestDGS113_DefaultBindIsLoopback(t *testing.T) {
	port := freePort(t)
	s := startTestListener(t, &ServerConfig{Port: port, Environment: "test"})

	host, gotPort, err := net.SplitHostPort(s.httpServer.Addr)
	if err != nil {
		t.Fatalf("listener addr %q is not host:port: %v", s.httpServer.Addr, err)
	}
	if gotPort != port {
		t.Fatalf("listener port = %q, want %q", gotPort, port)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		t.Fatalf("default listener host = %q (addr %q); want a loopback address — an empty host binds every interface",
			host, s.httpServer.Addr)
	}

	// And it actually serves there. Start used to return before the listener
	// existed, so poll briefly rather than assume.
	var resp *http.Response
	for i := 0; i < 50; i++ {
		resp, err = http.Get("http://127.0.0.1:" + port + "/health")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET loopback /health: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("loopback /health = %d, want 200", resp.StatusCode)
	}
}

// TestDGS113_BlankBindHostStaysLoopback: a blank or whitespace-only value (an
// empty line in an env file) must not widen the bind.
func TestDGS113_BlankBindHostStaysLoopback(t *testing.T) {
	for _, blank := range []string{"", "   ", "\t"} {
		port := freePort(t)
		s := startTestListener(t, &ServerConfig{Port: port, BindHost: blank})
		if want := "127.0.0.1:" + port; s.httpServer.Addr != want {
			t.Errorf("BindHost %q: addr = %q, want %q", blank, s.httpServer.Addr, want)
		}
	}
}

// TestDGS113_ExplicitBindHostWidens: widening is possible, but only by naming it.
// Containers depend on this — loopback inside a container is unreachable from
// the published port.
func TestDGS113_ExplicitBindHostWidens(t *testing.T) {
	port := freePort(t)
	s := startTestListener(t, &ServerConfig{Port: port, BindHost: "0.0.0.0"})
	if want := "0.0.0.0:" + port; s.httpServer.Addr != want {
		t.Fatalf("addr = %q, want %q", s.httpServer.Addr, want)
	}
}

// TestDGS113_BindFailureIsReturned: Start used to call ListenAndServe in a
// goroutine and only log its error, so a port clash or a bad bind host left a
// running process with nothing listening. It must now fail loudly.
func TestDGS113_BindFailureIsReturned(t *testing.T) {
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("hold port: %v", err)
	}
	defer held.Close()
	_, port, _ := net.SplitHostPort(held.Addr().String())

	gin.SetMode(gin.TestMode)
	s := &Server{cfg: &ServerConfig{Port: port, ShutdownTimeout: time.Second}, router: gin.New()}
	if err := s.Start(); err == nil {
		_ = s.Stop(context.Background())
		t.Fatalf("Start on an occupied port returned nil; want a bind error")
	}
}

// ipv6LoopbackAvailable reports whether this machine can bind [::1] at all.
func ipv6LoopbackAvailable() bool {
	l, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

// TestDefaultBindAnswersOnIPv6Loopback: in production both consumers of the
// gateway, Caddy and PDI, dial "localhost:7340", and on pdi-dojo-1 localhost
// resolves to ::1 FIRST (observed 2026-09-30: the one live peer on :7340 was
// [::1]). A default bind of 127.0.0.1 alone would leave every one of their
// requests depending on the dialer's IPv4 fallback. Loopback must mean what
// localhost means on the box: both families.
func TestDefaultBindAnswersOnIPv6Loopback(t *testing.T) {
	if !ipv6LoopbackAvailable() {
		t.Skip("no IPv6 loopback on this machine")
	}
	port := freePort(t)
	startTestListener(t, &ServerConfig{Port: port, Environment: "test"})

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("::1", port), time.Second)
	if err != nil {
		t.Fatalf("dial [::1]:%s with the default bind: %v — localhost clients that try ::1 first get refused", port, err)
	}
	_ = conn.Close()
}
