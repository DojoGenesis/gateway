package main

import (
	"net"
	"testing"
)

// DGS-136: the bridge webhook server bound ":" + port — every interface — the
// same class as the gateway's DGS-113. It must default to loopback and widen
// only when DOJO_BRIDGE_HOST says so.

func TestDGS136_BridgeDefaultBindIsLoopback(t *testing.T) {
	t.Setenv("DOJO_BRIDGE_HOST", "")
	addr := bridgeListenAddr("8090")
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("bridge addr %q is not host:port: %v", addr, err)
	}
	if port != "8090" {
		t.Fatalf("port = %q, want 8090", port)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		t.Fatalf("default bridge bind host = %q (addr %q); want loopback — an empty host binds every interface", host, addr)
	}
}

func TestDGS136_BridgeBlankHostStaysLoopback(t *testing.T) {
	t.Setenv("DOJO_BRIDGE_HOST", "   ")
	if got, want := bridgeListenAddr("8090"), "127.0.0.1:8090"; got != want {
		t.Fatalf("addr = %q, want %q", got, want)
	}
}

func TestDGS136_BridgeExplicitHostWidens(t *testing.T) {
	// The prod container sets this: cloudflared reaches the bridge over the
	// compose network at bridge:8090, which loopback would refuse.
	t.Setenv("DOJO_BRIDGE_HOST", "0.0.0.0")
	if got, want := bridgeListenAddr("8090"), "0.0.0.0:8090"; got != want {
		t.Fatalf("addr = %q, want %q", got, want)
	}
}
