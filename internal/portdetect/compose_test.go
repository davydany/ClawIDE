package portdetect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePortSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    []ComposePort
		wantErr bool
	}{
		{
			name: "single port",
			spec: "8080",
			want: []ComposePort{{Service: "svc", HostPort: 8080, ContainerPort: 8080}},
		},
		{
			name: "host:container",
			spec: "8080:80",
			want: []ComposePort{{Service: "svc", HostPort: 8080, ContainerPort: 80}},
		},
		{
			name: "IP:host:container",
			spec: "127.0.0.1:5432:5432",
			want: []ComposePort{{Service: "svc", HostPort: 5432, ContainerPort: 5432}},
		},
		{
			name: "with protocol stripped",
			spec: "8080:80/tcp",
			want: []ComposePort{{Service: "svc", HostPort: 8080, ContainerPort: 80}},
		},
		{
			name: "port range",
			spec: "8080-8082:80-82",
			want: []ComposePort{
				{Service: "svc", HostPort: 8080, ContainerPort: 80},
				{Service: "svc", HostPort: 8081, ContainerPort: 81},
				{Service: "svc", HostPort: 8082, ContainerPort: 82},
			},
		},
		{
			name:    "range mismatch error",
			spec:    "8080-8082:80-83",
			wantErr: true,
		},
		{
			name:    "unsupported format",
			spec:    "a:b:c:d",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePortSpec("svc", tt.spec)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitRange(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStart int
		wantEnd   int
		wantErr   bool
	}{
		{
			name:      "valid range",
			input:     "8080-8082",
			wantStart: 8080,
			wantEnd:   8082,
		},
		{
			name:    "invalid format no dash",
			input:   "8080",
			wantErr: true,
		},
		{
			name:    "end less than start",
			input:   "8082-8080",
			wantErr: true,
		},
		{
			name:    "non-numeric start",
			input:   "abc-8080",
			wantErr: true,
		},
		{
			name:    "non-numeric end",
			input:   "8080-abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := splitRange(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStart, start)
			assert.Equal(t, tt.wantEnd, end)
		})
	}
}

func TestExtractComposePorts(t *testing.T) {
	t.Run("valid compose file", func(t *testing.T) {
		dir := t.TempDir()
		content := `
services:
  web:
    ports:
      - "8080:80"
      - "443:443"
  redis:
    ports:
      - "6379:6379"
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(content), 0644))

		ports, err := ExtractComposePorts(dir)
		require.NoError(t, err)
		assert.Len(t, ports, 3)
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ExtractComposePorts(dir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no docker-compose.yml found")
	})
}
