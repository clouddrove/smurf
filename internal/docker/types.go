package docker

import "time"

// BuildOptions contains configuration for the Docker build
type BuildOptions struct {
	ContextDir     string
	DockerfilePath string
	BuildArgs      map[string]string
	Target         string
	Platform       string
	NoCache        bool
	BuildKit       bool
	Timeout        time.Duration
	Excludes       []string
	Labels         map[string]string
}

// ImageInfo struct to hold information about a Docker image
type ImageInfo struct {
	ID       string
	Size     int64
	Created  time.Time
	Platform string
	Layers   int
	Tag      string
}

// TagOptions struct to hold options for tagging a Docker image
type TagOptions struct {
	Source string
	Target string
}

// PushOptions struct to hold options for pushing a Docker image
type PushOptions struct {
	ImageName string
	Timeout   time.Duration
}

// PushProgress struct to hold progress information for pushing a Docker image
type jsonMessage struct {
	Status   string `json:"status"`
	Error    string `json:"error"`
	Progress string `json:"progress"`
	ID       string `json:"id"`
}
