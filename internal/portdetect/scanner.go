package portdetect

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Port represents a listening TCP port detected on the host OS.
type Port struct {
	Number  int    `json:"number"`
	PID     int    `json:"pid"`
	Process string `json:"process"`
	Address string `json:"address"`
}

// ScanPorts detects listening TCP ports on the system.
// On macOS it shells out to lsof; on Linux it uses ss.
func ScanPorts() ([]Port, error) {
	switch runtime.GOOS {
	case "darwin":
		return scanDarwin()
	case "linux":
		return scanLinux()
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// scanDarwin parses output from: lsof -iTCP -sTCP:LISTEN -P -n
//
// Typical output lines (after the header):
//
//	COMMAND   PID   USER   FD   TYPE  DEVICE SIZE/OFF NODE NAME
//	node    12345  user   22u  IPv4  0x1234      0t0  TCP 127.0.0.1:3000 (LISTEN)
//	postgres  789  user   5u  IPv6  0x5678      0t0  TCP *:5432 (LISTEN)
func scanDarwin() ([]Port, error) {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-P", "-n")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// lsof exits 1 when no results; treat that as empty, not an error.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof: %w", err)
	}

	seen := make(map[string]bool)
	var ports []Port

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		// Need at least 10 fields: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME [STATE]
		if len(fields) < 9 {
			continue
		}

		process := fields[0]
		pidStr := fields[1]
		name := fields[8] // e.g. "127.0.0.1:3000" or "*:5432"

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue // skip header or malformed lines
		}

		address, portStr := splitHostPort(name)
		portNum, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		// Deduplicate by address:port (lsof can report the same port multiple times
		// for different file descriptors in the same process).
		key := fmt.Sprintf("%s:%d:%d", address, portNum, pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		ports = append(ports, Port{
			Number:  portNum,
			PID:     pid,
			Process: process,
			Address: address,
		})
	}

	return ports, scanner.Err()
}

// scanLinux parses output from: ss -tlnp
//
// Typical output lines (after the header):
//
//	State   Recv-Q  Send-Q  Local Address:Port  Peer Address:Port  Process
//	LISTEN  0       128     0.0.0.0:8080        0.0.0.0:*          users:(("node",pid=12345,fd=22))
//	LISTEN  0       128     [::]:5432           [::]:*             users:(("postgres",pid=789,fd=5))
func scanLinux() ([]Port, error) {
	cmd := exec.Command("ss", "-tlnp")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ss: %w", err)
	}

	var ports []Port
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip the header line.
		if strings.HasPrefix(line, "State") {
			continue
		}

		fields := strings.Fields(line)
		// Expect at least 5 fields: State Recv-Q Send-Q LocalAddr:Port PeerAddr:Port [Process]
		if len(fields) < 5 {
			continue
		}

		localAddr := fields[3] // e.g. "0.0.0.0:8080" or "[::]:5432"
		address, portStr := splitHostPort(localAddr)
		portNum, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		pid := 0
		process := ""
		// Process info is in the last field, e.g. users:(("node",pid=12345,fd=22))
		if len(fields) >= 6 {
			pid, process = parseSSProcess(fields[5])
		}

		key := fmt.Sprintf("%s:%d:%d", address, portNum, pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		ports = append(ports, Port{
			Number:  portNum,
			PID:     pid,
			Process: process,
			Address: address,
		})
	}

	return ports, scanner.Err()
}

// splitHostPort splits a string like "127.0.0.1:3000", "*:5432", or "[::]:5432"
// into (address, port). It handles IPv6 bracket notation.
func splitHostPort(s string) (string, string) {
	// Handle IPv6 bracket notation: [::]:5432
	if strings.HasPrefix(s, "[") {
		if idx := strings.LastIndex(s, "]:"); idx != -1 {
			return s[:idx+1], s[idx+2:]
		}
		return s, ""
	}

	// Standard host:port -- find the last colon to handle addresses that contain colons.
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s, ""
	}
	return s[:idx], s[idx+1:]
}

// parseSSProcess extracts PID and process name from the ss process column.
// Input format: users:(("node",pid=12345,fd=22))
func parseSSProcess(s string) (int, string) {
	pid := 0
	process := ""

	// Extract process name from between ((" and the next "
	if start := strings.Index(s, "((\""); start != -1 {
		rest := s[start+3:]
		if end := strings.Index(rest, "\""); end != -1 {
			process = rest[:end]
		}
	}

	// Extract pid from pid=NNNN
	if pidIdx := strings.Index(s, "pid="); pidIdx != -1 {
		rest := s[pidIdx+4:]
		end := strings.IndexAny(rest, ",)")
		if end == -1 {
			end = len(rest)
		}
		if n, err := strconv.Atoi(rest[:end]); err == nil {
			pid = n
		}
	}

	return pid, process
}
