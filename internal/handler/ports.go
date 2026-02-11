package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ccmux/internal/middleware"
	"github.com/davydany/ccmux/internal/portdetect"
)

// portsResponse is the JSON envelope returned by DetectPorts.
type portsResponse struct {
	OSPorts      []portdetect.Port        `json:"os_ports"`
	ComposePorts []portdetect.ComposePort `json:"compose_ports"`
}

// DetectPorts combines an OS-level port scan with port mappings extracted
// from the project's docker-compose.yml and returns the result as JSON.
func (h *Handlers) DetectPorts(w http.ResponseWriter, r *http.Request) {
	resp := portsResponse{}

	// OS-level port scan.
	osPorts, err := portdetect.ScanPorts()
	if err != nil {
		log.Printf("port scan error: %v", err)
		// Non-fatal: we still try to return compose ports.
	}
	if osPorts != nil {
		resp.OSPorts = osPorts
	} else {
		resp.OSPorts = []portdetect.Port{}
	}

	// Compose port extraction using the project path from middleware.
	project := middleware.GetProject(r)
	if project.Path != "" {
		composePorts, err := portdetect.ExtractComposePorts(project.Path)
		if err != nil {
			log.Printf("compose port extraction error: %v", err)
			// Non-fatal: compose file may not exist.
		}
		if composePorts != nil {
			resp.ComposePorts = composePorts
		}
	}

	if resp.ComposePorts == nil {
		resp.ComposePorts = []portdetect.ComposePort{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("port detection JSON encode error: %v", err)
	}
}
