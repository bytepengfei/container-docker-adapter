package main

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
)

type fakeCommandRunner struct {
	mu        sync.Mutex
	paths     map[string]string
	calls     [][]string
	inspectOK bool
	current   string
}

func (r *fakeCommandRunner) LookPath(file string) (string, error) {
	path := r.paths[file]
	if path == "" {
		return "", os.ErrNotExist
	}
	return path, nil
}

func (r *fakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)
	joined := strings.Join(args, " ")
	switch {
	case joined == "system status":
		return []byte("running"), nil
	case joined == "context inspect apple-container" && !r.inspectOK:
		return nil, errors.New("context not found")
	case joined == "context show":
		return []byte(r.current + "\n"), nil
	default:
		return nil, nil
	}
}

func TestSetupCreatesAndUsesDockerContext(t *testing.T) {
	socket := "/tmp/adapter.sock"
	stubSocketPing(t)
	runner := &fakeCommandRunner{
		paths: map[string]string{
			"docker":    "/opt/homebrew/bin/docker",
			"container": "/opt/homebrew/bin/container",
		},
	}
	var output strings.Builder
	if err := runSetup(context.Background(), []string{"--socket", socket}, &output, runner); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"/opt/homebrew/bin/container", "system", "status"},
		{"/opt/homebrew/bin/docker", "context", "inspect", "apple-container"},
		{"/opt/homebrew/bin/docker", "context", "create", "apple-container", "--description", "Docker Engine compatibility context backed by Apple Container", "--docker", "host=unix://" + socket},
		{"/opt/homebrew/bin/docker", "context", "use", "apple-container"},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

func TestSetupUpdatesExistingContext(t *testing.T) {
	socket := "/tmp/adapter.sock"
	stubSocketPing(t)
	runner := &fakeCommandRunner{
		paths:     map[string]string{"docker": "docker", "container": "container"},
		inspectOK: true,
	}
	if err := runSetup(context.Background(), []string{"--socket", socket, "--use=false"}, io.Discard, runner); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(runner.calls[2], " "); !strings.Contains(got, "context update apple-container") {
		t.Fatalf("update call = %q", got)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("calls = %v, expected no context use", runner.calls)
	}
}

func TestContextRemoveSwitchesToDefaultFirst(t *testing.T) {
	runner := &fakeCommandRunner{
		paths:   map[string]string{"docker": "docker"},
		current: "apple-container",
	}
	if err := runContextRemove(context.Background(), nil, io.Discard, runner); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"docker", "context", "show"},
		{"docker", "context", "use", "default"},
		{"docker", "context", "rm", "-f", "apple-container"},
	}
	if !reflect.DeepEqual(runner.calls, want) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, want)
	}
}

func TestResolveContainerFromHomebrewPath(t *testing.T) {
	runner := &fakeCommandRunner{paths: map[string]string{"container": "/opt/homebrew/bin/container"}}
	path, err := resolveBinary(runner, "", "container", "brew install container")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/opt/homebrew/bin/container" {
		t.Fatalf("path = %q", path)
	}
}

func TestWaitForBackendCanBeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	target := &pingBackend{Backend: nil}
	done := make(chan error, 1)
	go func() {
		done <- waitForBackendReady(ctx, target, true, io.Discard)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

type pingBackend struct {
	backend.Backend
}

func (*pingBackend) Ping(context.Context) error {
	return errors.New("not running")
}

func stubSocketPing(t *testing.T) {
	t.Helper()
	original := pingAdapterSocket
	pingAdapterSocket = func(context.Context, string) error { return nil }
	t.Cleanup(func() {
		pingAdapterSocket = original
	})
}
