package model

import "time"

type Capabilities struct {
	Exec       bool
	Logs       bool
	Attach     bool
	Events     bool
	Stats      bool
	Volumes    bool
	Networks   bool
	Build      bool
	Compose    bool
	Swarm      bool
	Checkpoint bool
}

type Version struct {
	Platform      Platform    `json:"Platform"`
	Components    []Component `json:"Components,omitempty"`
	Version       string      `json:"Version"`
	APIVersion    string      `json:"ApiVersion"`
	MinAPIVersion string      `json:"MinAPIVersion"`
	GitCommit     string      `json:"GitCommit"`
	GoVersion     string      `json:"GoVersion"`
	Os            string      `json:"Os"`
	Arch          string      `json:"Arch"`
	KernelVersion string      `json:"KernelVersion"`
	BuildTime     string      `json:"BuildTime"`
}

type Platform struct {
	Name string `json:"Name"`
}

type Component struct {
	Name    string            `json:"Name"`
	Version string            `json:"Version"`
	Details map[string]string `json:"Details,omitempty"`
}

type Info struct {
	ID                string   `json:"ID"`
	Containers        int      `json:"Containers"`
	ContainersRunning int      `json:"ContainersRunning"`
	ContainersPaused  int      `json:"ContainersPaused"`
	ContainersStopped int      `json:"ContainersStopped"`
	Images            int      `json:"Images"`
	Driver            string   `json:"Driver"`
	OperatingSystem   string   `json:"OperatingSystem"`
	OSType            string   `json:"OSType"`
	Architecture      string   `json:"Architecture"`
	NCPU              int      `json:"NCPU"`
	MemTotal          int64    `json:"MemTotal"`
	DockerRootDir     string   `json:"DockerRootDir"`
	ServerVersion     string   `json:"ServerVersion"`
	Warnings          []string `json:"Warnings,omitempty"`
}

type Container struct {
	ID      string
	Names   []string
	Image   string
	ImageID string
	Command string
	Created time.Time
	State   string
	Status  string
	Ports   []Port
	Labels  map[string]string
	Mounts  []Mount
}

type Port struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	Type        string
}

type Mount struct {
	Type        string
	Name        string
	Source      string
	Destination string
	Driver      string
	Mode        string
	RW          bool
	Propagation string
}

type ContainerSpec struct {
	Name       string
	Image      string
	Cmd        []string
	Entrypoint []string
	Env        []string
	Labels     map[string]string
	WorkingDir string
	Tty        bool
	OpenStdin  bool
}

type ContainerCreateResult struct {
	ID       string   `json:"Id"`
	Warnings []string `json:"Warnings"`
}

type Image struct {
	ID          string
	RepoTags    []string
	RepoDigests []string
	Created     time.Time
	Size        int64
	SharedSize  int64
	VirtualSize int64
	Labels      map[string]string
	Containers  int64
}

type ImageDelete struct {
	Untagged string `json:"Untagged,omitempty"`
	Deleted  string `json:"Deleted,omitempty"`
}

type ListContainersOptions struct {
	All    bool
	Limit  int
	Size   bool
	Filter string
}

type ListImagesOptions struct {
	All     bool
	Filter  string
	Filters string
}

type StopOptions struct {
	TimeoutSeconds *int
}

type RemoveOptions struct {
	Force         bool
	RemoveVolumes bool
}

type RemoveImageOptions struct {
	Force   bool
	NoPrune bool
}

type RegistryAuth struct {
	Raw string
}

type LogOptions struct {
	Follow     bool
	Stdout     bool
	Stderr     bool
	Since      string
	Until      string
	Timestamps bool
	Tail       string
}
