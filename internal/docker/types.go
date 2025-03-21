package docker

import "time"

// BuildOptions struct to hold options for building a Docker image
type BuildOptions struct {
	ContextDir     string
	DockerfilePath string
	NoCache        bool
	BuildArgs      map[string]string
	Target         string
	Platform       string
	Timeout        time.Duration
	BuildKit       bool
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
}