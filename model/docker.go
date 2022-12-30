package model

type DockerContainerContext struct {
	Path  string `yaml:"path"`
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}

type DockerUpdateContext struct {
	ContainerContext *DockerContainerContext
	NewVersion       string
	Created          string
	HubLink          string
	Digest           string
	Hostname         string
}
