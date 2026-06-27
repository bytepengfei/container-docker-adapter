package api

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bytepengfei/container-docker-adapter/internal/backend"
)

type ServerOptions struct {
	SocketPath string
}

type Server struct {
	backend backend.Backend
	options ServerOptions
	server  *http.Server
}

func NewServer(backend backend.Backend, options ServerOptions) *Server {
	s := &Server{
		backend: backend,
		options: options,
	}
	s.server = &http.Server{
		Handler: NewRouter(backend),
	}
	return s
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	if s.options.SocketPath == "" {
		return errors.New("socket path is required")
	}

	if err := os.MkdirAll(filepath.Dir(s.options.SocketPath), 0755); err != nil {
		return err
	}
	if err := os.RemoveAll(s.options.SocketPath); err != nil {
		return err
	}

	listener, err := net.Listen("unix", s.options.SocketPath)
	if err != nil {
		return err
	}
	defer listener.Close()

	if err := os.Chmod(s.options.SocketPath, 0600); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("docker compatibility API listening on unix://%s", s.options.SocketPath)
		errCh <- s.server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
