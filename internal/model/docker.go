package model

type DockerService struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	State  string `json:"state"`
	Ports  string `json:"ports"`
}

type ComposeConfig struct {
	Services map[string]ComposeService `yaml:"services"`
}

type ComposeService struct {
	Image       string   `yaml:"image"`
	Build       any      `yaml:"build"`
	Ports       []string `yaml:"ports"`
	Volumes     []string `yaml:"volumes"`
	Environment any      `yaml:"environment"`
	DependsOn   any      `yaml:"depends_on"`
}
