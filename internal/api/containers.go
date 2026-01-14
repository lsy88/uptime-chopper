package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/go-chi/chi/v5"

	"github.com/lsy88/uptime-chopper/internal/model"
)

func containersRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cs, err := deps.Docker.ListContainers(ctx)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, cs)
	})

	r.Get("/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		tail := r.URL.Query().Get("tail")
		if tail == "" {
			tail = "200"
		}
		sinceSec := 3600
		if v := r.URL.Query().Get("sinceSeconds"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				sinceSec = n
			}
		}
		since := time.Now().Add(-time.Duration(sinceSec) * time.Second)

		rc, err := deps.Docker.Logs(r.Context(), id, tail, since)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		defer rc.Close()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = writeDockerLogsAtMost(w, rc, deps.Config.MaxDockerLogBytes, stdcopy.StdCopy)
	})

	r.Post("/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := deps.Docker.Start(r.Context(), id); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	r.Post("/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body struct {
			TimeoutSeconds int `json:"timeoutSeconds"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		to := time.Duration(maxInt(1, body.TimeoutSeconds)) * time.Second
		if err := deps.Docker.Stop(r.Context(), id, to); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	r.Post("/{id}/restart", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body struct {
			TimeoutSeconds int `json:"timeoutSeconds"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		to := time.Duration(maxInt(1, body.TimeoutSeconds)) * time.Second
		if err := deps.Docker.Restart(r.Context(), id, to); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	r.Put("/{id}/restart-policy", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var body model.RestartPolicy
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if err := deps.Docker.UpdateRestartPolicy(r.Context(), id, container.RestartPolicy{
			Name:              container.RestartPolicyMode(body.Name),
			MaximumRetryCount: body.MaximumRetryCount,
		}); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	return r
}

type stdCopyFn func(dstout io.Writer, dsterr io.Writer, src io.Reader) (written int64, err error)

func writeDockerLogsAtMost(w io.Writer, src io.Reader, maxBytes int, stdCopy stdCopyFn) (int64, bool) {
	lw := newLimitedWriter(maxBytes)
	_, _ = stdCopy(lw, lw, src)
	_, _ = w.Write(lw.Bytes())
	return int64(len(lw.Bytes())), lw.Truncated()
}
