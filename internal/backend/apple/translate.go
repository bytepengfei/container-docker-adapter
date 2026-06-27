package apple

import (
	"time"

	"github.com/pengfei/container-docker-adapter/internal/model"
)

type Machine struct {
	MachineID string
	Name      string
	Image     string
	Created   time.Time
	Status    string
}

type Image struct {
	ID      string
	Name    string
	Created time.Time
	Size    int64
}

func ContainerFromMachine(machine Machine) model.Container {
	state := machine.Status
	if state == "" {
		state = "unknown"
	}
	return model.Container{
		ID:      machine.MachineID,
		Names:   []string{machine.Name},
		Image:   machine.Image,
		Created: machine.Created,
		State:   state,
		Status:  dockerStatus(state, machine.Created),
		Labels:  map[string]string{},
	}
}

func ImageFromApple(image Image) model.Image {
	return model.Image{
		ID:          image.ID,
		RepoTags:    []string{image.Name},
		RepoDigests: []string{},
		Created:     image.Created,
		Size:        image.Size,
		VirtualSize: image.Size,
		Labels:      map[string]string{},
		Containers:  -1,
	}
}

func dockerStatus(state string, created time.Time) string {
	switch state {
	case "running":
		return "Up " + time.Since(created).Round(time.Second).String()
	case "stopped", "exited":
		return "Exited (0)"
	case "created":
		return "Created"
	default:
		return state
	}
}
