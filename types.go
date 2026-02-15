package main

import "time"

// Podman API response types.
// Fields like Config.Env are intentionally omitted so secrets and
// environment variables are never parsed or displayed.

type Container struct {
	ID      string    `json:"Id"`
	Names   []string  `json:"Names"`
	Image   string    `json:"Image"`
	ImageID string    `json:"ImageID"`
	Command []string  `json:"Command"`
	Created time.Time `json:"Created"`
	State   string    `json:"State"`
	Status  string    `json:"Status"`
	Ports        []Port              `json:"Ports"`
	ExposedPorts map[string][]string `json:"ExposedPorts"`
}

type Port struct {
	HostIP        string `json:"host_ip"`
	HostPort      uint16 `json:"host_port"`
	ContainerPort uint16 `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type ContainerInspect struct {
	ID              string           `json:"Id"`
	Name            string           `json:"Name"`
	Created         time.Time        `json:"Created"`
	Image           string           `json:"Image"`
	ImageName       string           `json:"ImageName"`
	State           ContainerState   `json:"State"`
	Config          ContainerConfig  `json:"Config"`
	Mounts          []Mount          `json:"Mounts"`
	NetworkSettings *NetworkSettings `json:"NetworkSettings"`
	RestartCount    int32            `json:"RestartCount"`
	HostConfig      *HostConfig      `json:"HostConfig"`
}

type ContainerState struct {
	Status     string    `json:"Status"`
	Running    bool      `json:"Running"`
	StartedAt  time.Time `json:"StartedAt"`
	FinishedAt time.Time `json:"FinishedAt"`
	ExitCode   int32     `json:"ExitCode"`
	Health     *Health   `json:"Health,omitempty"`
}

type Health struct {
	Status string `json:"Status"`
}

type ContainerConfig struct {
	Hostname string            `json:"Hostname"`
	Image    string            `json:"Image"`
	Cmd      []string          `json:"Cmd"`
	Labels   map[string]string `json:"Labels"`
	// Env is intentionally omitted — never show environment variables.
}

type HostConfig struct {
	RestartPolicy RestartPolicy `json:"RestartPolicy"`
}

type RestartPolicy struct {
	Name              string `json:"Name"`
	MaximumRetryCount uint   `json:"MaximumRetryCount"`
}

type Mount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	RW          bool   `json:"RW"`
}

type NetworkSettings struct {
	Ports map[string][]HostPort `json:"Ports"`
}

type HostPort struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type ImageSummary struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Created  int64    `json:"Created"`
	Size     int64    `json:"Size"`
}

type ImageInspect struct {
	ID          string      `json:"Id"`
	Digest      string      `json:"Digest"`
	RepoTags    []string    `json:"RepoTags"`
	RepoDigests []string    `json:"RepoDigests"`
	Created     time.Time   `json:"Created"`
	Author      string      `json:"Author"`
	Architecture string    `json:"Architecture"`
	Os          string      `json:"Os"`
	Size        int64       `json:"Size"`
	Config      ImageConfig `json:"Config"`
	Labels      map[string]string `json:"Labels"`
}

type ImageConfig struct {
	Cmd        []string `json:"Cmd"`
	Entrypoint []string `json:"Entrypoint"`
	WorkingDir string   `json:"WorkingDir"`
	StopSignal string   `json:"StopSignal"`
	// Env is intentionally omitted — never show environment variables.
}
