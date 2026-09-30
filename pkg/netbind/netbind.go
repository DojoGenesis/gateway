// Package netbind owns one decision for every HTTP listener in this repo:
// which addresses it binds.
//
// The rule (DGS-113, DGS-136):
//   - An unset, blank, or "localhost" host means LOOPBACK, and loopback means
//     what "localhost" means on the machine: 127.0.0.1 AND ::1. Production
//     clients dial "localhost:<port>", and on the production host localhost
//     resolves to ::1 first (observed 2026-09-30), so an IPv4-only loopback
//     bind would leave each of their connections relying on a fallback.
//   - Any other host is bound exactly as given, and is reported as widened
//     unless it is itself a loopback address, so callers can warn.
//
// ::1 is best effort: a machine or container without IPv6 loopback still
// gets 127.0.0.1. Every other bind failure (a port clash, an address that is
// not on this machine) is an error, and nothing is left listening.
package netbind

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// Plan is the resolved bind: the addresses to listen on, and whether they
// reach beyond loopback.
type Plan struct {
	Addrs   []string
	Widened bool
}

// Resolve turns a configured host and a port into a Plan. It does not bind.
func Resolve(host, port string) Plan {
	h := strings.TrimSpace(host)
	if h == "" || strings.EqualFold(h, "localhost") {
		return Plan{Addrs: []string{net.JoinHostPort("127.0.0.1", port), net.JoinHostPort("::1", port)}}
	}
	ip := net.ParseIP(strings.Trim(h, "[]"))
	return Plan{
		Addrs:   []string{net.JoinHostPort(strings.Trim(h, "[]"), port)},
		Widened: ip == nil || !ip.IsLoopback(),
	}
}

// Listen binds every address in the Plan for host and port. On success it
// returns one listener per bound address (the IPv6 loopback may be absent,
// see the package docs) and the Plan it followed. On failure it returns an
// error and closes anything it had already opened.
func Listen(host, port string) ([]net.Listener, Plan, error) {
	plan := Resolve(host, port)
	var lns []net.Listener
	for i, addr := range plan.Addrs {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			if i > 0 && !plan.Widened && ipv6Unavailable(err) {
				continue
			}
			for _, open := range lns {
				_ = open.Close()
			}
			return nil, plan, err
		}
		lns = append(lns, ln)
		// A port of "0" asks the kernel to choose; every later address in the
		// plan must reuse the choice so all listeners share one port.
		if port == "0" && i == 0 {
			_, chosen, _ := net.SplitHostPort(ln.Addr().String())
			for j := range plan.Addrs[1:] {
				host, _, _ := net.SplitHostPort(plan.Addrs[j+1])
				plan.Addrs[j+1] = net.JoinHostPort(host, chosen)
			}
		}
	}
	return lns, plan, nil
}

// ipv6Unavailable reports whether a bind failed because this machine has no
// usable IPv6 loopback, as opposed to a real conflict.
func ipv6Unavailable(err error) bool {
	return errors.Is(err, syscall.EADDRNOTAVAIL) || errors.Is(err, syscall.EAFNOSUPPORT)
}
