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
}


// TagOptions struct to hold options for tagging a Docker image
type TagOptions struct {
	Source string
	Target string
}

// PushOptions struct to hold options for pushing a Docker image
type PushOptions struct {
	ImageName string
}
