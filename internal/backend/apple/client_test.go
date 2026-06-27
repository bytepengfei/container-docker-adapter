package apple

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	apperrors "github.com/bytepengfei/container-docker-adapter/internal/errors"
	"github.com/bytepengfei/container-docker-adapter/internal/model"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls [][]string
	run   func([]string) (CommandResult, error)
	start func([]string, io.Writer, io.Writer) (func() error, error)
}

func (r *fakeRunner) Run(_ context.Context, args ...string) (CommandResult, error) {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string(nil), args...))
	r.mu.Unlock()
	return r.run(args)
}

func (r *fakeRunner) RunInput(_ context.Context, _ io.Reader, args ...string) (CommandResult, error) {
	return r.Run(context.Background(), args...)
}

func (r *fakeRunner) Start(_ context.Context, stdout, stderr io.Writer, args ...string) (func() error, error) {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string(nil), args...))
	r.mu.Unlock()
	return r.start(args, stdout, stderr)
}

func TestContainerLifecycleUsesAppleCLI(t *testing.T) {
	runner := &fakeRunner{}
	runner.run = func(args []string) (CommandResult, error) {
		switch args[0] {
		case "create":
			return CommandResult{Stdout: []byte("demo\n")}, nil
		case "delete":
			return CommandResult{}, nil
		default:
			t.Fatalf("unexpected run command: %v", args)
			return CommandResult{}, nil
		}
	}
	runner.start = func(args []string, stdout, _ io.Writer) (func() error, error) {
		if !reflect.DeepEqual(args, []string{"start", "--attach", "demo"}) {
			t.Fatalf("start args = %v", args)
		}
		done := make(chan struct{})
		go func() {
			_, _ = io.WriteString(stdout, "Hello from Apple Container\n")
			close(done)
		}()
		return func() error {
			<-done
			return nil
		}, nil
	}
	backend := New(&Client{Runner: runner})

	created, err := backend.CreateContainer(context.Background(), model.ContainerSpec{
		Name:       "demo",
		Image:      "hello-world",
		Env:        []string{"A=B"},
		Labels:     map[string]string{"project": "adapter"},
		WorkingDir: "/work",
		Tty:        false,
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := backend.AttachContainer(context.Background(), created.ID, model.LogOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := backend.StartContainer(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	result, err := backend.WaitContainer(context.Background(), created.ID, "not-running")
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != 0 || string(output) != "Hello from Apple Container\n" {
		t.Fatalf("wait=%+v output=%q", result, output)
	}
	if err := backend.RemoveContainer(context.Background(), created.ID, model.RemoveOptions{}); err != nil {
		t.Fatal(err)
	}

	wantCreate := []string{"create", "--name", "demo", "--env", "A=B", "--label", "project=adapter", "--workdir", "/work", "hello-world"}
	if !reflect.DeepEqual(runner.calls[0], wantCreate) {
		t.Fatalf("create args = %v, want %v", runner.calls[0], wantCreate)
	}
}

func TestListContainersParsesAppleJSON(t *testing.T) {
	runner := &fakeRunner{
		run: func(args []string) (CommandResult, error) {
			return CommandResult{Stdout: []byte(`[{"id":"demo","configuration":{"creationDate":"2026-06-27T16:20:08Z","id":"demo","image":{"reference":"docker.io/library/hello-world:latest","descriptor":{"digest":"sha256:abc"}},"initProcess":{"executable":"/hello","arguments":[],"terminal":false,"workingDirectory":"/"},"labels":{}},"status":{"state":"running","startedDate":"2026-06-27T16:20:09Z"}}]`)}, nil
		},
		start: func([]string, io.Writer, io.Writer) (func() error, error) {
			return nil, errors.New("unexpected start")
		},
	}
	backend := New(&Client{Runner: runner})
	containers, err := backend.ListContainers(context.Background(), model.ListContainersOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 || containers[0].ID != "demo" || containers[0].Command != "/hello" {
		t.Fatalf("containers = %+v", containers)
	}
}

func TestCommandErrorsAreTranslated(t *testing.T) {
	runner := &fakeRunner{
		run: func([]string) (CommandResult, error) {
			return CommandResult{}, errors.New("container does not exist")
		},
		start: func([]string, io.Writer, io.Writer) (func() error, error) {
			return nil, errors.New("unexpected start")
		},
	}
	backend := New(&Client{Runner: runner})
	_, err := backend.InspectContainer(context.Background(), "missing")
	if !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestExecRunnerHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := (ExecRunner{Binary: "/bin/sleep"}).Run(ctx, "5")
	if err == nil || !strings.Contains(err.Error(), "signal: killed") {
		t.Fatalf("error = %v, want killed process", err)
	}
}

func TestBuildImageMapsOptionsAndRejectsUnsafeContext(t *testing.T) {
	runner := &fakeRunner{
		run: func(args []string) (CommandResult, error) {
			if args[0] != "build" {
				t.Fatalf("command = %v, want build", args)
			}
			return CommandResult{Stdout: []byte("built\n")}, nil
		},
		start: func([]string, io.Writer, io.Writer) (func() error, error) {
			return nil, errors.New("unexpected start")
		},
	}
	backend := New(&Client{Runner: runner})
	var output bytes.Buffer
	if err := backend.BuildImage(context.Background(), buildContext(t, "Dockerfile", tar.TypeReg), model.BuildOptions{
		Tags:      []string{"demo:latest"},
		NoCache:   true,
		BuildArgs: map[string]string{"VERSION": "1"},
	}, &output); err != nil {
		t.Fatal(err)
	}
	call := runner.calls[0]
	if !containsArgs(call, "--tag", "demo:latest") || !containsArgs(call, "--build-arg", "VERSION=1") {
		t.Fatalf("build args = %v", call)
	}
	if !strings.Contains(output.String(), `"stream":"built\n"`) {
		t.Fatalf("build output = %q", output.String())
	}

	err := backend.BuildImage(context.Background(), buildContext(t, "../escape", tar.TypeReg), model.BuildOptions{}, io.Discard)
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("unsafe context error = %v, want bad request", err)
	}
	err = backend.BuildImage(context.Background(), buildContext(t, "link", tar.TypeSymlink), model.BuildOptions{}, io.Discard)
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("symlink context error = %v, want bad request", err)
	}
}

func TestVolumeAndNetworkStructuredOutput(t *testing.T) {
	runner := &fakeRunner{
		run: func(args []string) (CommandResult, error) {
			switch args[0] {
			case "volume":
				return CommandResult{Stdout: []byte(`[{"id":"data","configuration":{"creationDate":"2026-06-27T16:37:19Z","driver":"local","labels":{},"name":"data","options":{},"source":"/volumes/data.img"}}]`)}, nil
			case "network":
				return CommandResult{Stdout: []byte(`[{"id":"default","configuration":{"creationDate":"2026-06-12T16:28:35Z","labels":{},"mode":"nat","name":"default","options":{},"plugin":"container-network-vmnet"}}]`)}, nil
			default:
				t.Fatalf("unexpected command: %v", args)
				return CommandResult{}, nil
			}
		},
		start: func([]string, io.Writer, io.Writer) (func() error, error) {
			return nil, errors.New("unexpected start")
		},
	}
	backend := New(&Client{Runner: runner})
	volumes, err := backend.ListVolumes(context.Background())
	if err != nil || len(volumes) != 1 || volumes[0].Mountpoint != "/volumes/data.img" {
		t.Fatalf("volumes=%+v err=%v", volumes, err)
	}
	networks, err := backend.ListNetworks(context.Background())
	if err != nil || len(networks) != 1 || networks[0].Driver != "bridge" {
		t.Fatalf("networks=%+v err=%v", networks, err)
	}
}

func buildContext(t *testing.T, name string, entryType byte) io.Reader {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	content := []byte("FROM scratch\n")
	if err := writer.WriteHeader(&tar.Header{Name: name, Typeflag: entryType, Mode: 0o644, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if entryType == tar.TypeReg {
		if _, err := writer.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buffer.Bytes())
}

func containsArgs(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
