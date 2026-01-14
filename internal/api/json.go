package api

import (
	"encoding/json"
	"io"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ioCopyAtMost(dst io.Writer, src io.Reader, maxBytes int) (int64, bool) {
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	lr := &io.LimitedReader{R: src, N: int64(maxBytes) + 1}
	n, _ := io.Copy(dst, lr)
	return n, n > int64(maxBytes)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type limitedWriter struct {
	max       int
	buf       []byte
	truncated bool
}

func newLimitedWriter(maxBytes int) *limitedWriter {
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	return &limitedWriter{max: maxBytes, buf: make([]byte, 0, minInt(maxBytes, 4096))}
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remain := w.max - len(w.buf)
	if remain <= 0 {
		w.truncated = true
		return len(p), nil
	}
	if len(p) <= remain {
		w.buf = append(w.buf, p...)
		return len(p), nil
	}
	w.buf = append(w.buf, p[:remain]...)
	w.truncated = true
	return len(p), nil
}

func (w *limitedWriter) Bytes() []byte {
	return w.buf
}

func (w *limitedWriter) Truncated() bool {
	return w.truncated
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
