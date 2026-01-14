package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lsy88/uptime-chopper/internal/model"
	"github.com/lsy88/uptime-chopper/internal/monitor"
)

func monitorsRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		st := deps.Store.GetState()
		writeJSON(w, http.StatusOK, st.Monitors)
	})
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var m model.Monitor
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if m.ID == "" {
			m.ID = monitor.NewID()
		}
		m = normalizeMonitor(m)
		out, err := deps.Store.UpsertMonitor(m)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var m model.Monitor
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		m.ID = id
		m = normalizeMonitor(m)
		out, err := deps.Store.UpsertMonitor(m)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := deps.Store.DeleteMonitor(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	return r
}

func normalizeMonitor(m model.Monitor) model.Monitor {
	if m.IntervalSeconds <= 0 {
		m.IntervalSeconds = 60
	}
	if m.TimeoutSeconds <= 0 {
		m.TimeoutSeconds = 10
	}
	if m.Logs.Tail <= 0 {
		m.Logs.Tail = 200
	}
	if m.Type == model.MonitorTypeHTTP && m.HTTP == nil {
		m.HTTP = &model.HTTPMonitor{}
	}
	if m.Type == model.MonitorTypeContainer && m.Container == nil {
		m.Container = &model.ContainerMonitor{}
	}
	return m
}
