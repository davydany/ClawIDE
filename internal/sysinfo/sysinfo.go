package sysinfo

import (
	"log"
	"math"
	"net"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/store"
	"github.com/davydany/ClawIDE/internal/tmux"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// CPUCore holds usage for a single logical core.
type CPUCore struct {
	Core         int     `json:"core"`
	UsagePercent float64 `json:"usage_percent"`
}

// Memory holds system memory stats.
type Memory struct {
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// NetworkInterface holds one network interface and its primary IP.
type NetworkInterface struct {
	Name   string `json:"name"`
	IPv4   string `json:"ipv4"`
	Status string `json:"status"`
}

// Projects holds project counts.
type Projects struct {
	Total   int `json:"total"`
	Starred int `json:"starred"`
}

// Stats is the top-level response returned by Gather.
type Stats struct {
	CPU          []CPUCore          `json:"cpu"`
	Memory       Memory             `json:"memory"`
	Network      []NetworkInterface `json:"network"`
	TmuxSessions int                `json:"tmux_sessions"`
	Projects     Projects           `json:"projects"`
}

// Gather collects all system stats. Each sub-collector is non-fatal;
// failures are logged and the field stays at its zero value.
func Gather(st *store.Store) Stats {
	s := Stats{
		CPU:     []CPUCore{},
		Network: []NetworkInterface{},
	}

	// CPU per-core usage (blocks for 200ms sample window).
	if percents, err := cpu.Percent(200*time.Millisecond, true); err != nil {
		log.Printf("sysinfo: cpu: %v", err)
	} else {
		for i, pct := range percents {
			s.CPU = append(s.CPU, CPUCore{
				Core:         i,
				UsagePercent: round2(pct),
			})
		}
	}

	// Memory.
	if vm, err := mem.VirtualMemory(); err != nil {
		log.Printf("sysinfo: memory: %v", err)
	} else {
		s.Memory = Memory{
			TotalBytes:   vm.Total,
			UsedBytes:    vm.Used,
			UsagePercent: round2(vm.UsedPercent),
		}
	}

	// Network interfaces (using stdlib net package for portability).
	if ifaces, err := net.Interfaces(); err != nil {
		log.Printf("sysinfo: network: %v", err)
	} else {
		for _, iface := range ifaces {
			// Skip loopback and interfaces with no addresses.
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil || len(addrs) == 0 {
				continue
			}
			ipv4 := ""
			for _, addr := range addrs {
				ip := stripCIDR(addr.String())
				if isIPv4(ip) {
					ipv4 = ip
					break
				}
			}
			if ipv4 == "" {
				continue
			}
			s.Network = append(s.Network, NetworkInterface{
				Name:   iface.Name,
				IPv4:   ipv4,
				Status: deriveStatus(iface.Flags),
			})
		}
	}

	// Tmux sessions.
	if sessions, err := tmux.ListClawIDESessions(); err != nil {
		log.Printf("sysinfo: tmux: %v", err)
	} else {
		s.TmuxSessions = len(sessions)
	}

	// Projects.
	if st != nil {
		projects := st.GetProjects()
		s.Projects.Total = len(projects)
		for _, p := range projects {
			if p.Starred {
				s.Projects.Starred++
			}
		}
	}

	return s
}

// round2 rounds a float to 2 decimal places.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// stripCIDR removes the /prefix from a CIDR notation address.
func stripCIDR(addr string) string {
	if idx := strings.Index(addr, "/"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// isIPv4 checks if a string is an IPv4 address.
func isIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

// deriveStatus returns "up" or "down" based on interface flags.
func deriveStatus(flags net.Flags) string {
	if flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}
