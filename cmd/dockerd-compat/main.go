package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pengfei/container-docker-adapter/internal/api"
	"github.com/pengfei/container-docker-adapter/internal/backend/memory"
)

func main() {
	socketPath := flag.String("socket", defaultSocketPath(), "unix socket path to listen on")
	flag.Parse()

	backend := memory.New()
	server := api.NewServer(backend, api.ServerOptions{
		SocketPath: *socketPath,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
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
