package model

type DockerService struct {
	Name    string `json:"name"`
	Service string `json:"service"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Health  string `json:"health"`
	Ports   string `json:"ports"`
}

type ComposeConfig struct {
	Services map[string]ComposeService `yaml:"services"`
}

type HealthcheckConfig struct {
	Test        any    `yaml:"test"`
	Interval    string `yaml:"interval"`
	Timeout     string `yaml:"timeout"`
	Retries     int    `yaml:"retries"`
	StartPeriod string `yaml:"start_period"`
	Disable     bool   `yaml:"disable"`
}

type ComposeService struct {
	Image         string             `yaml:"image"`
	Build         any                `yaml:"build"`
	Ports         []string           `yaml:"ports"`
	Volumes       []string           `yaml:"volumes"`
	Environment   any                `yaml:"environment"`
	EnvFile       any                `yaml:"env_file"`
	DependsOn     any                `yaml:"depends_on"`
	Command       any                `yaml:"command"`
	ContainerName string             `yaml:"container_name"`
	Restart       string             `yaml:"restart"`
	Healthcheck   *HealthcheckConfig `yaml:"healthcheck"`
}
