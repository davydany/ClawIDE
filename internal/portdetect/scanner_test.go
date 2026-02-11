package portdetect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAddr string
		wantPort string
	}{
		{
			name:     "IPv4 standard",
			input:    "127.0.0.1:3000",
			wantAddr: "127.0.0.1",
			wantPort: "3000",
		},
		{
			name:     "wildcard",
			input:    "*:5432",
			wantAddr: "*",
			wantPort: "5432",
		},
		{
			name:     "IPv6 bracket notation",
			input:    "[::]:5432",
			wantAddr: "[::]",
			wantPort: "5432",
		},
		{
			name:     "IPv6 loopback",
			input:    "[::1]:8080",
			wantAddr: "[::1]",
			wantPort: "8080",
		},
		{
			name:     "no port",
			input:    "noport",
			wantAddr: "noport",
			wantPort: "",
		},
		{
			name:     "IPv6 bracket without port",
			input:    "[::1]",
			wantAddr: "[::1]",
			wantPort: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, port := splitHostPort(tt.input)
			assert.Equal(t, tt.wantAddr, addr)
			assert.Equal(t, tt.wantPort, port)
		})
	}
}

func TestParseSSProcess(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantPID     int
		wantProcess string
	}{
		{
			name:        "node process",
			input:       `users:(("node",pid=12345,fd=22))`,
			wantPID:     12345,
			wantProcess: "node",
		},
		{
			name:        "postgres process",
			input:       `users:(("postgres",pid=789,fd=5))`,
			wantPID:     789,
			wantProcess: "postgres",
		},
		{
			name:        "empty string",
			input:       "",
			wantPID:     0,
			wantProcess: "",
		},
		{
			name:        "no match format",
			input:       "random-text",
			wantPID:     0,
			wantProcess: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pid, process := parseSSProcess(tt.input)
			assert.Equal(t, tt.wantPID, pid)
			assert.Equal(t, tt.wantProcess, process)
		})
	}
}
