package api

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const dockerRawStreamContentType = "application/vnd.docker.raw-stream"

func writeDockerStream(w http.ResponseWriter, r *http.Request, stream io.Reader, tty bool, allowUpgrade bool) error {
	if allowUpgrade && requestsUpgrade(r) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return fmt.Errorf("response writer does not support connection hijacking")
		}
		conn, rw, err := hijacker.Hijack()
		if err != nil {
			return err
		}
		defer conn.Close()

		if _, err := rw.WriteString("HTTP/1.1 101 UPGRADED\r\nContent-Type: " + dockerRawStreamContentType + "\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n"); err != nil {
			return err
		}
		if err := rw.Flush(); err != nil {
			return err
		}
		return copyDockerStream(rw.Writer, stream, tty)
	}

	w.Header().Set("Content-Type", dockerRawStreamContentType)
	w.WriteHeader(http.StatusOK)
	return copyDockerStream(bufio.NewWriter(w), stream, tty)
}

func requestsUpgrade(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "tcp") &&
		headerHasToken(r.Header.Get("Connection"), "upgrade")
}

func headerHasToken(value, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

func copyDockerStream(dst *bufio.Writer, src io.Reader, tty bool) error {
	if tty {
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		return dst.Flush()
	}

	buffer := make([]byte, 32*1024)
	for {
		n, err := src.Read(buffer)
		if n > 0 {
			var header [8]byte
			header[0] = 1 // stdout
			binary.BigEndian.PutUint32(header[4:], uint32(n))
			if _, writeErr := dst.Write(header[:]); writeErr != nil {
				return writeErr
			}
			if _, writeErr := dst.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if flushErr := dst.Flush(); flushErr != nil {
				return flushErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
