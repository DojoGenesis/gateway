package netbind

import (
	"net"
	"testing"
	"time"
)

func has6() bool {
	l, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func closeAll(lns []net.Listener) {
	for _, l := range lns {
		_ = l.Close()
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		host    string
		addrs   []string
		widened bool
	}{
		{"", []string{"127.0.0.1:7340", "[::1]:7340"}, false},
		{"   ", []string{"127.0.0.1:7340", "[::1]:7340"}, false},
		{"localhost", []string{"127.0.0.1:7340", "[::1]:7340"}, false},
		{"127.0.0.1", []string{"127.0.0.1:7340"}, false},
		{"::1", []string{"[::1]:7340"}, false},
		{"[::1]", []string{"[::1]:7340"}, false},
		{"0.0.0.0", []string{"0.0.0.0:7340"}, true},
		{"192.168.1.180", []string{"192.168.1.180:7340"}, true},
		{"bridge", []string{"bridge:7340"}, true},
	}
	for _, c := range cases {
		p := Resolve(c.host, "7340")
		if len(p.Addrs) != len(c.addrs) || p.Widened != c.widened {
			t.Errorf("Resolve(%q) = %+v, want addrs %v widened %v", c.host, p, c.addrs, c.widened)
			continue
		}
		for i := range c.addrs {
			if p.Addrs[i] != c.addrs[i] {
				t.Errorf("Resolve(%q).Addrs[%d] = %q, want %q", c.host, i, p.Addrs[i], c.addrs[i])
			}
		}
	}
}

func TestListenDefaultAnswersOnBothLoopbacks(t *testing.T) {
	lns, plan, err := Listen("", "0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeAll(lns)
	for _, ln := range lns {
		go func(l net.Listener) {
			for {
				c, err := l.Accept()
				if err != nil {
					return
				}
				_ = c.Close()
			}
		}(ln)
	}
	_, port, _ := net.SplitHostPort(lns[0].Addr().String())
	hosts := []string{"127.0.0.1"}
	if has6() {
		hosts = append(hosts, "::1")
		if len(lns) != 2 {
			t.Fatalf("IPv6 loopback exists but only %d listener(s) opened: %v", len(lns), plan.Addrs)
		}
	}
	for _, h := range hosts {
		c, err := net.DialTimeout("tcp", net.JoinHostPort(h, port), time.Second)
		if err != nil {
			t.Errorf("dial %s:%s: %v", h, port, err)
			continue
		}
		_ = c.Close()
	}
}

func TestListenPortClashIsAnErrorAndLeavesNothingOpen(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	_, port, _ := net.SplitHostPort(taken.Addr().String())

	lns, _, err := Listen("", port)
	if err == nil {
		closeAll(lns)
		t.Fatalf("Listen on a taken port succeeded")
	}
	if lns != nil {
		t.Fatalf("Listen returned listeners alongside an error: %v", lns)
	}
}

func TestListenIPv6ClashIsNotSwallowed(t *testing.T) {
	if !has6() {
		t.Skip("no IPv6 loopback")
	}
	// Take the port on ::1 only; 127.0.0.1 is free. A real conflict on the
	// second family must be an error, not skipped as "no IPv6".
	taken, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	_, port, _ := net.SplitHostPort(taken.Addr().String())

	lns, _, err := Listen("", port)
	if err == nil {
		closeAll(lns)
		t.Fatalf("Listen succeeded although [::1]:%s is taken", port)
	}
	// And the 127.0.0.1 listener it had opened was closed again.
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatalf("127.0.0.1:%s still held after the failed Listen: %v", port, err)
	}
	_ = l.Close()
}

func TestListenExplicitHostIsExact(t *testing.T) {
	lns, plan, err := Listen("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	defer closeAll(lns)
	if len(lns) != 1 || plan.Widened {
		t.Fatalf("explicit 127.0.0.1: %d listeners, widened=%v; want exactly 1, not widened", len(lns), plan.Widened)
	}
}
