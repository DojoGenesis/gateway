package main

import (
	"testing"

	"github.com/DojoGenesis/gateway/pkg/netbind"
)

// DGS-136: the bridge webhook server bound ":" + port — every interface — the
// same class as the gateway's DGS-113. It must default to loopback and widen
// only when DOJO_BRIDGE_HOST says so. The rule itself is pkg/netbind's and is
// tested there; these pin that the bridge feeds it DOJO_BRIDGE_HOST.

func bridgePlan() netbind.Plan { return netbind.Resolve(bridgeBindHost(), "8090") }

func TestDGS136_BridgeDefaultBindIsLoopback(t *testing.T) {
	t.Setenv("DOJO_BRIDGE_HOST", "")
	p := bridgePlan()
	if p.Widened || p.Addrs[0] != "127.0.0.1:8090" {
		t.Fatalf("default bridge plan = %+v; want loopback starting 127.0.0.1:8090", p)
	}
}

func TestDGS136_BridgeBlankHostStaysLoopback(t *testing.T) {
	t.Setenv("DOJO_BRIDGE_HOST", "   ")
	if p := bridgePlan(); p.Widened || p.Addrs[0] != "127.0.0.1:8090" {
		t.Fatalf("blank DOJO_BRIDGE_HOST plan = %+v; want loopback", p)
	}
}

func TestDGS136_BridgeExplicitHostWidens(t *testing.T) {
	// The prod container sets this: cloudflared reaches the bridge over the
	// compose network at bridge:8090, which loopback would refuse.
	t.Setenv("DOJO_BRIDGE_HOST", "0.0.0.0")
	if p := bridgePlan(); !p.Widened || len(p.Addrs) != 1 || p.Addrs[0] != "0.0.0.0:8090" {
		t.Fatalf("DOJO_BRIDGE_HOST=0.0.0.0 plan = %+v; want exactly 0.0.0.0:8090, widened", p)
	}
}
