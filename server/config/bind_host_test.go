package config

import (
	"testing"
)

// DGS-113: the bind host is configurable and defaults to loopback.

func TestDGS113_BindHostDefaultsToLoopback(t *testing.T) {
	t.Setenv("GATEWAY_BIND_HOST", "")
	res := LoadWithOptions(LoadOptions{Path: writeConfig(t, "port: \"7340\"\n")})
	if res.Err != nil {
		t.Fatalf("load: %v", res.Err)
	}
	if res.Config.BindHost != "127.0.0.1" {
		t.Fatalf("default BindHost = %q, want 127.0.0.1", res.Config.BindHost)
	}
}

func TestDGS113_BindHostFromEnvironment(t *testing.T) {
	t.Setenv("GATEWAY_BIND_HOST", "0.0.0.0")
	res := LoadWithOptions(LoadOptions{Path: writeConfig(t, "bind_host: 10.0.0.5\n")})
	if res.Err != nil {
		t.Fatalf("load: %v", res.Err)
	}
	if res.Config.BindHost != "0.0.0.0" {
		t.Fatalf("BindHost = %q, want the environment's 0.0.0.0 to win over the file", res.Config.BindHost)
	}
}

func TestDGS113_BindHostFromFileIsHonoured(t *testing.T) {
	t.Setenv("GATEWAY_BIND_HOST", "")
	res := LoadWithOptions(LoadOptions{Path: writeConfig(t, "bind_host: 10.0.0.5\n")})
	if res.Err != nil {
		t.Fatalf("load: %v", res.Err)
	}
	for _, w := range res.Warnings {
		t.Errorf("unexpected warning — bind_host must be a real key, not an ignored one: %s", w)
	}
	if res.Config.BindHost != "10.0.0.5" {
		t.Fatalf("BindHost = %q, want 10.0.0.5 from the file", res.Config.BindHost)
	}
}

func TestDGS113_BlankBindHostIsUnset(t *testing.T) {
	t.Setenv("GATEWAY_BIND_HOST", "   ")
	res := LoadWithOptions(LoadOptions{Path: writeConfig(t, "bind_host: \"\"\n")})
	if res.Err != nil {
		t.Fatalf("load: %v", res.Err)
	}
	if res.Config.BindHost != "127.0.0.1" {
		t.Fatalf("BindHost = %q; a blank env var or file value must fall back to loopback, not bind every interface",
			res.Config.BindHost)
	}
}
