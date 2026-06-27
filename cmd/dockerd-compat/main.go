package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bytepengfei/container-docker-adapter/internal/api"
	"github.com/bytepengfei/container-docker-adapter/internal/backend"
	"github.com/bytepengfei/container-docker-adapter/internal/backend/apple"
	"github.com/bytepengfei/container-docker-adapter/internal/backend/memory"
)

var (
	version           = "dev"
	commit            = "unknown"
	buildTime         = "unknown"
	pingAdapterSocket = pingSocket
)

const defaultContextName = "apple-container"

type commandRunner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type osCommandRunner struct{}

func (osCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (osCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runCLI(ctx, os.Args[1:], os.Stdout, os.Stderr, osCommandRunner{}); err != nil {
		log.Fatal(err)
	}
}

func runCLI(ctx context.Context, args []string, stdout, stderr io.Writer, runner commandRunner) error {
	command := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "serve":
		return runServe(ctx, args, stderr, runner)
	case "setup":
		return runSetup(ctx, args, stdout, runner)
	case "context":
		if len(args) == 0 || args[0] != "remove" {
			return errors.New("usage: container-docker-adapter context remove [options]")
		}
		return runContextRemove(ctx, args[1:], stdout, runner)
	case "version":
		_, err := fmt.Fprintf(stdout, "container-docker-adapter %s (commit: %s, built: %s)\n", version, commit, buildTime)
		return err
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func runServe(ctx context.Context, args []string, stderr io.Writer, runner commandRunner) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	socketPath := flags.String("socket", defaultSocketPath(), "Unix socket path to listen on")
	backendName := flags.String("backend", "apple", "backend to use: apple or memory")
	containerBinary := flags.String("container-bin", "", "path to the Apple container CLI")
	waitForBackend := flags.Bool("wait-for-backend", false, "wait until the backend becomes available")
	if err := flags.Parse(args); err != nil {
		return err
	}

	binary := *containerBinary
	if *backendName == "apple" && binary == "" {
		resolved, err := runner.LookPath("container")
		if err != nil {
			return errors.New("Apple container CLI was not found in PATH; install it with `brew install container`")
		}
		binary = resolved
	}
	selected, err := selectBackend(*backendName, binary)
	if err != nil {
		return err
	}
	if err := waitForBackendReady(ctx, selected, *waitForBackend, stderr); err != nil {
		return err
	}

	server := api.NewServer(selected, api.ServerOptions{SocketPath: *socketPath})
	return server.ListenAndServe(ctx)
}

func runSetup(ctx context.Context, args []string, stdout io.Writer, runner commandRunner) error {
	flags := flag.NewFlagSet("setup", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	socketPath := flags.String("socket", defaultSocketPath(), "adapter Unix socket path")
	contextName := flags.String("context", defaultContextName, "Docker context name")
	dockerBinary := flags.String("docker-bin", "", "path to Docker CLI")
	containerBinary := flags.String("container-bin", "", "path to Apple container CLI")
	useContext := flags.Bool("use", true, "switch to the configured context")
	if err := flags.Parse(args); err != nil {
		return err
	}

	dockerPath, err := resolveBinary(runner, *dockerBinary, "docker", "brew install docker")
	if err != nil {
		return err
	}
	containerPath, err := resolveBinary(runner, *containerBinary, "container", "brew install container")
	if err != nil {
		return err
	}
	if _, err := runner.Run(ctx, containerPath, "system", "status"); err != nil {
		return fmt.Errorf("Apple Container system is unavailable; run `container system start`: %w", err)
	}
	if err := pingAdapterSocket(ctx, *socketPath); err != nil {
		return fmt.Errorf("adapter is unavailable at unix://%s; start it with `brew services start container-docker-adapter`: %w", *socketPath, err)
	}

	host := "host=unix://" + *socketPath
	description := "Docker Engine compatibility context backed by Apple Container"
	if _, err := runner.Run(ctx, dockerPath, "context", "inspect", *contextName); err == nil {
		if _, err := runner.Run(ctx, dockerPath, "context", "update", *contextName, "--description", description, "--docker", host); err != nil {
			return err
		}
	} else {
		if _, err := runner.Run(ctx, dockerPath, "context", "create", *contextName, "--description", description, "--docker", host); err != nil {
			return err
		}
	}
	if *useContext {
		if _, err := runner.Run(ctx, dockerPath, "context", "use", *contextName); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(stdout, "Docker context %q is configured for unix://%s\n", *contextName, *socketPath)
	return err
}

func runContextRemove(ctx context.Context, args []string, stdout io.Writer, runner commandRunner) error {
	flags := flag.NewFlagSet("context remove", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contextName := flags.String("context", defaultContextName, "Docker context name")
	dockerBinary := flags.String("docker-bin", "", "path to Docker CLI")
	if err := flags.Parse(args); err != nil {
		return err
	}
	dockerPath, err := resolveBinary(runner, *dockerBinary, "docker", "brew install docker")
	if err != nil {
		return err
	}
	current, err := runner.Run(ctx, dockerPath, "context", "show")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(current)) == *contextName {
		if _, err := runner.Run(ctx, dockerPath, "context", "use", "default"); err != nil {
			return err
		}
	}
	if _, err := runner.Run(ctx, dockerPath, "context", "rm", "-f", *contextName); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "Docker context %q was removed\n", *contextName)
	return err
}

func waitForBackendReady(ctx context.Context, selected backend.Backend, wait bool, output io.Writer) error {
	delay := 500 * time.Millisecond
	for {
		err := selected.Ping(ctx)
		if err == nil {
			return nil
		}
		if !wait {
			return fmt.Errorf("backend is unavailable: %w", err)
		}
		_, _ = fmt.Fprintf(output, "waiting for Apple Container backend: %v\n", err)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if delay < 5*time.Second {
			delay *= 2
		}
	}
}

func resolveBinary(runner commandRunner, configured, name, installHint string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	path, err := runner.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s was not found in PATH; install it with `%s`", name, installHint)
	}
	return path, nil
}

func pingSocket(ctx context.Context, socketPath string) error {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/_ping", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("ping returned HTTP %d", response.StatusCode)
	}
	return nil
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

func printUsage(output io.Writer) {
	_, _ = fmt.Fprintln(output, `Usage:
  container-docker-adapter serve [options]
  container-docker-adapter setup [options]
  container-docker-adapter context remove [options]
  container-docker-adapter version

Running without a subcommand remains equivalent to "serve".`)
}
