package apple

import "time"

type containerDTO struct {
	ID            string                 `json:"id"`
	Configuration containerConfiguration `json:"configuration"`
	Status        containerStatus        `json:"status"`
}

type containerConfiguration struct {
	CreationDate time.Time         `json:"creationDate"`
	ID           string            `json:"id"`
	Image        containerImage    `json:"image"`
	InitProcess  containerInit     `json:"initProcess"`
	Labels       map[string]string `json:"labels"`
	Mounts       []containerMount  `json:"mounts"`
}

type containerImage struct {
	Reference  string     `json:"reference"`
	Descriptor descriptor `json:"descriptor"`
}

type descriptor struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type containerInit struct {
	Executable       string   `json:"executable"`
	Arguments        []string `json:"arguments"`
	Terminal         bool     `json:"terminal"`
	WorkingDirectory string   `json:"workingDirectory"`
}

type containerMount struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type containerStatus struct {
	State       string    `json:"state"`
	StartedDate time.Time `json:"startedDate"`
}

type imageDTO struct {
	ID            string             `json:"id"`
	Configuration imageConfiguration `json:"configuration"`
	Variants      []imageVariant     `json:"variants"`
}

type imageConfiguration struct {
	CreationDate time.Time  `json:"creationDate"`
	Name         string     `json:"name"`
	Descriptor   descriptor `json:"descriptor"`
}

type imageVariant struct {
	Size int64 `json:"size"`
}

type versionDTO struct {
	AppName   string `json:"appName"`
	BuildType string `json:"buildType"`
	Commit    string `json:"commit"`
	Version   string `json:"version"`
}

type volumeDTO struct {
	ID            string              `json:"id"`
	Configuration volumeConfiguration `json:"configuration"`
}

type volumeConfiguration struct {
	CreationDate time.Time         `json:"creationDate"`
	Driver       string            `json:"driver"`
	Labels       map[string]string `json:"labels"`
	Name         string            `json:"name"`
	Options      map[string]string `json:"options"`
	Source       string            `json:"source"`
}

type networkDTO struct {
	ID            string               `json:"id"`
	Configuration networkConfiguration `json:"configuration"`
}

type networkConfiguration struct {
	CreationDate time.Time         `json:"creationDate"`
	Labels       map[string]string `json:"labels"`
	Mode         string            `json:"mode"`
	Name         string            `json:"name"`
	Options      map[string]string `json:"options"`
	Plugin       string            `json:"plugin"`
}
