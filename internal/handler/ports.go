package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/davydany/ClawIDE/internal/docker"
	"github.com/davydany/ClawIDE/internal/middleware"
	"github.com/davydany/ClawIDE/internal/portdetect"
)

// portsResponse is the JSON envelope returned by DetectPorts.
type portsResponse struct {
	OSPorts         []portdetect.Port            `json:"os_ports"`
	ComposeServices []docker.ComposeServiceDetail `json:"compose_services"`
}

// DetectPorts combines an OS-level port scan with service details extracted
// from the project's docker-compose.yml and returns the result as JSON.
func (h *Handlers) DetectPorts(w http.ResponseWriter, r *http.Request) {
	resp := portsResponse{}

	// OS-level port scan.
	osPorts, err := portdetect.ScanPorts()
	if err != nil {
		log.Printf("port scan error: %v", err)
		// Non-fatal: we still try to return compose services.
	}
	if osPorts != nil {
		resp.OSPorts = osPorts
	} else {
		resp.OSPorts = []portdetect.Port{}
	}

	// Compose service extraction using the project path from middleware.
	project := middleware.GetProject(r)
	if project.Path != "" {
		cfg, err := docker.ParseComposeFile(project.Path)
		if err != nil {
			log.Printf("compose parse error: %v", err)
			// Non-fatal: compose file may not exist.
		} else {
			resp.ComposeServices = docker.ExtractServiceDetails(cfg)
		}
	}

	if resp.ComposeServices == nil {
		resp.ComposeServices = []docker.ComposeServiceDetail{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("port detection JSON encode error: %v", err)
	}
}
