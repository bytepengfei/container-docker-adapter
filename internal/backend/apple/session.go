package apple

import (
	"bytes"
	"io"
	"sync"
)

type processResult struct {
	exitCode int
	err      error
}

type processSession struct {
	mu          sync.Mutex
	started     bool
	completed   bool
	result      processResult
	buffer      bytes.Buffer
	subscribers map[*io.PipeWriter]struct{}
	done        chan struct{}
}

func newProcessSession() *processSession {
	return &processSession{
		subscribers: make(map[*io.PipeWriter]struct{}),
		done:        make(chan struct{}),
	}
}

func (s *processSession) subscribe() io.ReadCloser {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := append([]byte(nil), s.buffer.Bytes()...)
	if s.completed {
		return io.NopCloser(bytes.NewReader(snapshot))
	}
	reader, writer := io.Pipe()
	s.subscribers[writer] = struct{}{}
	return &sessionReader{Reader: io.MultiReader(bytes.NewReader(snapshot), reader), closer: reader}
}

type sessionReader struct {
	io.Reader
	closer io.Closer
}

func (r *sessionReader) Close() error {
	return r.closer.Close()
}

func (s *processSession) Write(payload []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, _ := s.buffer.Write(payload)
	for writer := range s.subscribers {
		if _, err := writer.Write(payload); err != nil {
			delete(s.subscribers, writer)
		}
	}
	return n, nil
}

func (s *processSession) finish(result processResult) {
	s.mu.Lock()
	if s.completed {
		s.mu.Unlock()
		return
	}
	s.completed = true
	s.result = result
	for writer := range s.subscribers {
		_ = writer.Close()
		delete(s.subscribers, writer)
	}
	s.mu.Unlock()
	close(s.done)
}

func (s *processSession) wait() processResult {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.result
}

func (s *processSession) state() (started, completed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started, s.completed
}

func (s *processSession) markStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return false
	}
	s.started = true
	return true
}
