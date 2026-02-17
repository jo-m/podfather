package main

import (
	"encoding/json"
	"time"
)

// StringOrSlice handles JSON fields that may be either a string or []string.
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(data []byte) error {
	var slice []string
	if err := json.Unmarshal(data, &slice); err == nil {
		*s = slice
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = []string{str}
	return nil
}

// FlexString handles JSON fields that may be a string or a number.
type FlexString string

func (s *FlexString) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = FlexString(str)
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = FlexString(num.String())
	return nil
}

// Podman API response types.
// ContainerConfig.Env is intentionally omitted so container secrets and
// environment variables are never parsed or displayed.
// ImageConfig.Env is included because image env vars are build-time defaults,
// not runtime secrets.

type Container struct {
	ID           string              `json:"Id"`
	Names        []string            `json:"Names"`
	Image        string              `json:"Image"`
	ImageID      string              `json:"ImageID"`
	Command      []string            `json:"Command"`
	Created      time.Time           `json:"Created"`
	State        string              `json:"State"`
	Status       string              `json:"Status"`
	Ports        []Port              `json:"Ports"`
	ExposedPorts map[string][]string `json:"ExposedPorts"`
	Labels       map[string]string   `json:"Labels"`
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
	Path            string           `json:"Path"`
	Args            []string         `json:"Args"`
	Image           string           `json:"Image"`
	ImageName       string           `json:"ImageName"`
	State           ContainerState   `json:"State"`
	Config          ContainerConfig  `json:"Config"`
	Mounts          []Mount          `json:"Mounts"`
	NetworkSettings *NetworkSettings `json:"NetworkSettings"`
	RestartCount    int32            `json:"RestartCount"`
	HostConfig      *HostConfig      `json:"HostConfig"`
	OCIRuntime      string           `json:"OCIRuntime"`
	Driver          string           `json:"Driver"`
}

type ContainerState struct {
	Status     string    `json:"Status"`
	Running    bool      `json:"Running"`
	Paused     bool      `json:"Paused"`
	Pid        int       `json:"Pid"`
	OOMKilled  bool      `json:"OOMKilled"`
	StartedAt  time.Time `json:"StartedAt"`
	FinishedAt time.Time `json:"FinishedAt"`
	ExitCode   int32     `json:"ExitCode"`
	Health     *Health   `json:"Health,omitempty"`
}

type Health struct {
	Status string `json:"Status"`
}

type ContainerConfig struct {
	Hostname      string              `json:"Hostname"`
	Image         string              `json:"Image"`
	User          string              `json:"User"`
	Cmd           []string            `json:"Cmd"`
	Entrypoint    StringOrSlice       `json:"Entrypoint"`
	WorkingDir    string              `json:"WorkingDir"`
	StopSignal    FlexString          `json:"StopSignal"`
	Labels        map[string]string   `json:"Labels"`
	Annotations   map[string]string   `json:"Annotations"`
	ExposedPorts  map[string]struct{} `json:"ExposedPorts"`
	CreateCommand []string            `json:"CreateCommand"`
	// Env is intentionally omitted — never show environment variables.
}

type HostConfig struct {
	RestartPolicy  RestartPolicy `json:"RestartPolicy"`
	NetworkMode    string        `json:"NetworkMode"`
	Privileged     bool          `json:"Privileged"`
	ReadonlyRootfs bool          `json:"ReadonlyRootfs"`
	AutoRemove     bool          `json:"AutoRemove"`
	LogConfig      LogConfig     `json:"LogConfig"`
}

type RestartPolicy struct {
	Name              string `json:"Name"`
	MaximumRetryCount uint   `json:"MaximumRetryCount"`
}

type LogConfig struct {
	Type string `json:"Type"`
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
	ID           string            `json:"Id"`
	Digest       string            `json:"Digest"`
	RepoTags     []string          `json:"RepoTags"`
	RepoDigests  []string          `json:"RepoDigests"`
	Created      time.Time         `json:"Created"`
	Author       string            `json:"Author"`
	Comment      string            `json:"Comment"`
	Architecture string            `json:"Architecture"`
	Os           string            `json:"Os"`
	Size         int64             `json:"Size"`
	User         string            `json:"User"`
	Config       ImageConfig       `json:"Config"`
	Labels       map[string]string `json:"Labels"`
	RootFS       RootFS            `json:"RootFS"`
	History      []ImageHistory    `json:"History"`
	NamesHistory []string          `json:"NamesHistory"`
}

type ImageConfig struct {
	Cmd          []string            `json:"Cmd"`
	Entrypoint   StringOrSlice       `json:"Entrypoint"`
	WorkingDir   string              `json:"WorkingDir"`
	StopSignal   FlexString          `json:"StopSignal"`
	Env          []string            `json:"Env"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts"`
}

type RootFS struct {
	Type   string   `json:"Type"`
	Layers []string `json:"Layers"`
}

type ImageHistory struct {
	Created   time.Time `json:"created"`
	CreatedBy string    `json:"created_by"`
	Comment   string    `json:"comment"`
	Empty     bool      `json:"empty_layer"`
}

// App label prefix for container metadata.
const appLabelPrefix = "ch.jo-m.go.podfather.app."

// App represents a logical application composed of one or more containers
// sharing the same app name label.
type App struct {
	Name        string
	Icon        string
	Category    string
	SortIndex   int
	Subtitle    string
	Description string
	URL         string
	Containers  []Container
}

// AppCategory groups apps under a category heading.
type AppCategory struct {
	Name string
	Apps []App
}
