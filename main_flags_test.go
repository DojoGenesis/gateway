package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The shipped systemd unit runs `dojo-gateway -config /etc/dojo/config.yaml`.
// The binary parsed no flags at all, so that argument was discarded in silence
// and /etc/dojo/config.yaml was never opened. These tests pin the parsing of
// the exact argument vectors the deployment artifacts use.
func TestParseCommandLine(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want commandLine
	}{
		{
			name: "systemd unit form",
			args: []string{"-config", "/etc/dojo/config.yaml"},
			want: commandLine{ConfigPath: "/etc/dojo/config.yaml"},
		},
		{
			name: "double dash",
			args: []string{"--config", "/etc/dojo/config.yaml"},
			want: commandLine{ConfigPath: "/etc/dojo/config.yaml"},
		},
		{
			name: "equals form",
			args: []string{"-config=/etc/dojo/config.yaml"},
			want: commandLine{ConfigPath: "/etc/dojo/config.yaml"},
		},
		{
			name: "no arguments",
			args: nil,
			want: commandLine{},
		},
		{
			name: "check-config with a path",
			args: []string{"-config", "/etc/dojo/config.yaml", "-check-config"},
			want: commandLine{ConfigPath: "/etc/dojo/config.yaml", CheckConfig: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseCommandLine(tt.args))
		})
	}
}
