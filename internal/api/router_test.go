package api

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytepengfei/container-docker-adapter/internal/backend/memory"
)

func TestSystemEndpoints(t *testing.T) {
	handler := NewRouter(memory.New())

	ping := httptest.NewRecorder()
	handler.ServeHTTP(ping, httptest.NewRequest(http.MethodGet, "/_ping", nil))
	if ping.Code != http.StatusOK {
		t.Fatalf("ping status = %d, want %d", ping.Code, http.StatusOK)
	}
	if ping.Body.String() != "OK" {
		t.Fatalf("ping body = %q, want OK", ping.Body.String())
	}

	version := httptest.NewRecorder()
	handler.ServeHTTP(version, httptest.NewRequest(http.MethodGet, "/v1.47/version", nil))
	if version.Code != http.StatusOK {
		t.Fatalf("version status = %d, want %d", version.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(version.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["ApiVersion"] != "1.47" {
		t.Fatalf("ApiVersion = %v, want 1.47", body["ApiVersion"])
	}
}

func TestContainerLifecycleRoutes(t *testing.T) {
	handler := NewRouter(memory.New())

	createBody := bytes.NewBufferString(`{"Image":"hello-world:latest","Cmd":["echo","hello"]}`)
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/v1.47/containers/create?name=demo", createBody))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", create.Code, http.StatusCreated, create.Body.String())
	}

	var created struct {
		ID       string   `json:"Id"`
		Warnings []string `json:"Warnings"`
	}
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("created ID is empty")
	}

	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/containers/"+created.ID+"/start", nil))
	if start.Code != http.StatusNoContent {
		t.Fatalf("start status = %d, want %d", start.Code, http.StatusNoContent)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/containers/json", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", list.Code, http.StatusOK)
	}
	var containers []map[string]any
	if err := json.NewDecoder(list.Body).Decode(&containers); err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 {
		t.Fatalf("container count = %d, want 1", len(containers))
	}
	if containers[0]["State"] != "running" {
		t.Fatalf("container state = %v, want running", containers[0]["State"])
	}

	inspect := httptest.NewRecorder()
	handler.ServeHTTP(inspect, httptest.NewRequest(http.MethodGet, "/containers/demo/json", nil))
	if inspect.Code != http.StatusOK {
		t.Fatalf("inspect status = %d, want %d; body=%s", inspect.Code, http.StatusOK, inspect.Body.String())
	}

	removeConflict := httptest.NewRecorder()
	handler.ServeHTTP(removeConflict, httptest.NewRequest(http.MethodDelete, "/containers/demo", nil))
	if removeConflict.Code != http.StatusConflict {
		t.Fatalf("remove running status = %d, want %d", removeConflict.Code, http.StatusConflict)
	}

	remove := httptest.NewRecorder()
	handler.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/containers/demo?force=true", nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("force remove status = %d, want %d", remove.Code, http.StatusNoContent)
	}
}

func TestImagesRoutes(t *testing.T) {
	handler := NewRouter(memory.New())

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/images/json", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", list.Code, http.StatusOK)
	}

	pull := httptest.NewRecorder()
	handler.ServeHTTP(pull, httptest.NewRequest(http.MethodPost, "/images/create?fromImage=alpine&tag=latest", nil))
	if pull.Code != http.StatusOK {
		t.Fatalf("pull status = %d, want %d; body=%s", pull.Code, http.StatusOK, pull.Body.String())
	}
	if pull.Body.Len() == 0 {
		t.Fatal("pull response is empty")
	}
}

func TestPlannedContainerRoutes(t *testing.T) {
	handler := NewRouter(memory.New())
	id := createStartedContainer(t, handler, "planned")

	for _, tc := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodPost, "/containers/planned/restart", http.StatusNoContent},
		{http.MethodPost, "/containers/planned/pause", http.StatusNoContent},
		{http.MethodPost, "/containers/planned/unpause", http.StatusNoContent},
		{http.MethodPost, "/containers/planned/kill?signal=SIGTERM", http.StatusNoContent},
		{http.MethodGet, "/containers/planned/stats", http.StatusOK},
		{http.MethodGet, "/containers/planned/top", http.StatusOK},
		{http.MethodGet, "/containers/planned/changes", http.StatusOK},
		{http.MethodGet, "/containers/planned/archive?path=/tmp/file", http.StatusOK},
		{http.MethodPut, "/containers/planned/archive?path=/tmp/file", http.StatusOK},
		{http.MethodPost, "/containers/planned/attach?stdout=true", http.StatusOK},
		{http.MethodPost, "/containers/planned/resize?h=40&w=120", http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString("archive")))
		if rec.Code != tc.status {
			t.Fatalf("%s %s status = %d, want %d; body=%s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
		}
	}

	prune := httptest.NewRecorder()
	handler.ServeHTTP(prune, httptest.NewRequest(http.MethodPost, "/containers/prune", nil))
	if prune.Code != http.StatusOK {
		t.Fatalf("prune status = %d, want %d", prune.Code, http.StatusOK)
	}

	_ = id
}

func TestExecRoutes(t *testing.T) {
	handler := NewRouter(memory.New())
	createStartedContainer(t, handler, "exec-demo")

	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/containers/exec-demo/exec", bytes.NewBufferString(`{"Cmd":["echo","hi"],"AttachStdout":true}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("exec create status = %d, want %d; body=%s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/exec/"+created.ID+"/start", bytes.NewBufferString(`{"Detach":false,"Tty":false}`)))
	if start.Code != http.StatusOK {
		t.Fatalf("exec start status = %d, want %d", start.Code, http.StatusOK)
	}

	inspect := httptest.NewRecorder()
	handler.ServeHTTP(inspect, httptest.NewRequest(http.MethodGet, "/exec/"+created.ID+"/json", nil))
	if inspect.Code != http.StatusOK {
		t.Fatalf("exec inspect status = %d, want %d", inspect.Code, http.StatusOK)
	}
}

func TestAttachAndExecUpgradeToDockerRawStream(t *testing.T) {
	handler := NewRouter(memory.New())
	createStartedContainer(t, handler, "stream-demo")

	execCreate := httptest.NewRecorder()
	handler.ServeHTTP(execCreate, httptest.NewRequest(http.MethodPost, "/containers/stream-demo/exec", bytes.NewBufferString(`{"Cmd":["echo","hi"],"AttachStdout":true}`)))
	if execCreate.Code != http.StatusCreated {
		t.Fatalf("exec create status = %d, want %d", execCreate.Code, http.StatusCreated)
	}
	var execSession struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(execCreate.Body).Decode(&execSession); err != nil {
		t.Fatal(err)
	}

	assertDockerUpgrade(t, handler, "/containers/stream-demo/attach?stream=1&stdout=1", "")
	assertDockerUpgrade(t, handler, "/exec/"+execSession.ID+"/start", `{"Detach":false,"Tty":false}`)
}

func assertDockerUpgrade(t *testing.T, handler http.Handler, path, body string) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	writer := &hijackResponseWriter{header: make(http.Header), conn: serverConn}
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "tcp")
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.ServeHTTP(writer, request)
	}()

	reader := bufio.NewReader(clientConn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if status != "HTTP/1.1 101 UPGRADED\r\n" {
		t.Fatalf("upgrade status = %q, want 101 UPGRADED", status)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
	}

	var header [8]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		t.Fatal(err)
	}
	if header[0] != 1 {
		t.Fatalf("stream type = %d, want stdout (1)", header[0])
	}
	size := binary.BigEndian.Uint32(header[4:])
	if size == 0 {
		t.Fatal("stream frame payload is empty")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatal(err)
	}
	<-done
}

type hijackResponseWriter struct {
	header http.Header
	conn   net.Conn
}

func (w *hijackResponseWriter) Header() http.Header {
	return w.header
}

func (w *hijackResponseWriter) Write(payload []byte) (int, error) {
	return w.conn.Write(payload)
}

func (w *hijackResponseWriter) WriteHeader(int) {}

func (w *hijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

func TestVolumeNetworkAuthAndEventRoutes(t *testing.T) {
	handler := NewRouter(memory.New())
	createStartedContainer(t, handler, "net-demo")

	volumeCreate := httptest.NewRecorder()
	handler.ServeHTTP(volumeCreate, httptest.NewRequest(http.MethodPost, "/volumes/create", bytes.NewBufferString(`{"Name":"data"}`)))
	if volumeCreate.Code != http.StatusCreated {
		t.Fatalf("volume create status = %d, want %d", volumeCreate.Code, http.StatusCreated)
	}

	for _, path := range []string{"/volumes", "/volumes/data"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}

	networkCreate := httptest.NewRecorder()
	handler.ServeHTTP(networkCreate, httptest.NewRequest(http.MethodPost, "/networks/create", bytes.NewBufferString(`{"Name":"devnet"}`)))
	if networkCreate.Code != http.StatusCreated {
		t.Fatalf("network create status = %d, want %d; body=%s", networkCreate.Code, http.StatusCreated, networkCreate.Body.String())
	}
	var network struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(networkCreate.Body).Decode(&network); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/networks", ""},
		{http.MethodGet, "/networks/" + network.ID, ""},
		{http.MethodPost, "/networks/" + network.ID + "/connect", `{"Container":"net-demo"}`},
		{http.MethodPost, "/networks/" + network.ID + "/disconnect", `{"Container":"net-demo","Force":true}`},
		{http.MethodPost, "/networks/prune", ""},
		{http.MethodPost, "/auth", `{"Username":"u","Password":"p"}`},
		{http.MethodGet, "/events", ""},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body)))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want %d; body=%s", tc.method, tc.path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

func TestExpandedImageRoutes(t *testing.T) {
	handler := NewRouter(memory.New())

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/images/hello-world:latest/json", ""},
		{http.MethodGet, "/images/hello-world:latest/history", ""},
		{http.MethodPost, "/images/hello-world:latest/push", ""},
		{http.MethodGet, "/images/hello-world:latest/get", ""},
		{http.MethodPost, "/images/load", "tar-data"},
		{http.MethodPost, "/images/prune", ""},
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body)))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want %d; body=%s", tc.method, tc.path, rec.Code, http.StatusOK, rec.Body.String())
		}
	}
}

func TestNotPlannedRoutesReturn501(t *testing.T) {
	handler := NewRouter(memory.New())
	for _, path := range []string{"/swarm/init", "/plugins", "/containers/demo/checkpoints"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("%s status = %d, want %d", path, rec.Code, http.StatusNotImplemented)
		}
	}
}

func createStartedContainer(t *testing.T, handler http.Handler, name string) string {
	t.Helper()
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/containers/create?name="+name, bytes.NewBufferString(`{"Image":"hello-world:latest","Cmd":["sh"]}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(create.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRecorder()
	handler.ServeHTTP(start, httptest.NewRequest(http.MethodPost, "/containers/"+created.ID+"/start", nil))
	if start.Code != http.StatusNoContent {
		t.Fatalf("start status = %d, want %d", start.Code, http.StatusNoContent)
	}
	return created.ID
}
