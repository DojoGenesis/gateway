package server

import "testing"

// TestListenAddr covers the DGS-113 default: loopback-only unless a bind
// host is explicitly supplied (which main.go sources from
// config.Config.BindHost / GATEWAY_BIND_HOST).
func TestListenAddr(t *testing.T) {
	cases := []struct {
		name     string
		bindHost string
		port     string
		want     string
	}{
		{
			name:     "empty bind host defaults to loopback",
			bindHost: "",
			port:     "7340",
			want:     "127.0.0.1:7340",
		},
		{
			name:     "explicit override widens to all interfaces",
			bindHost: "0.0.0.0",
			port:     "7340",
			want:     "0.0.0.0:7340",
		},
		{
			name:     "IPv6 override is bracketed by JoinHostPort",
			bindHost: "::",
			port:     "7340",
			want:     "[::]:7340",
		},
		{
			name:     "explicit loopback override is passed through unchanged",
			bindHost: "127.0.0.1",
			port:     "9000",
			want:     "127.0.0.1:9000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := listenAddr(tc.bindHost, tc.port)
			if got != tc.want {
				t.Errorf("listenAddr(%q, %q) = %q, want %q", tc.bindHost, tc.port, got, tc.want)
			}
		})
	}
}
