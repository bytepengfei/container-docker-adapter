package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bytepengfei/container-docker-adapter/internal/api"
	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/backend/apple"
	"github.com/bytepengfei/container-docker-adapter/internal/backend/memory"
)

func main() {
	socketPath := flag.String("socket", defaultSocketPath(), "unix socket path to listen on")
	backendName := flag.String("backend", "apple", "backend to use: apple or memory")
	containerBinary := flag.String("container-bin", "/usr/local/bin/container", "path to the Apple container CLI")
	flag.Parse()

	selected, err := selectBackend(*backendName, *containerBinary)
	if err != nil {
		log.Fatal(err)
	}
	if err := selected.Ping(context.Background()); err != nil {
		log.Fatalf("%s backend is unavailable: %v", *backendName, err)
	}
	server := api.NewServer(selected, api.ServerOptions{
		SocketPath: *socketPath,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}

func selectBackend(name, containerBinary string) (backend.Backend, error) {
	switch name {
	case "apple":
		return apple.New(&apple.Client{Binary: containerBinary}), nil
	case "memory":
		return memory.New(), nil
	default:
		return nil, fmt.Errorf("unsupported backend %q (expected apple or memory)", name)
	}
}

func defaultSocketPath() string {
	if value := os.Getenv("DOCKER_COMPAT_SOCKET"); value != "" {
		return value
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "/tmp/docker-compat.sock"
	}
	return home + "/.docker-compat/docker.sock"
}
