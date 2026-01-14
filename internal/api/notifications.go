package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lsy88/uptime-chopper/internal/model"
	"github.com/lsy88/uptime-chopper/internal/monitor"
)

func notificationsRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		notifs := deps.Store.GetNotifications()

		type NotificationResponse struct {
			model.Notification
			Editable bool `json:"editable"`
		}

		var resp []NotificationResponse
		existingNames := make(map[string]bool)

		for _, n := range notifs {
			existingNames[n.Name] = true
			resp = append(resp, NotificationResponse{
				Notification: n,
				Editable:     true,
			})
		}

		writeJSON(w, http.StatusOK, resp)
	})

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var n model.Notification
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if n.ID == "" {
			n.ID = monitor.NewID()
		}

		out, err := deps.Store.UpsertNotification(n)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var n model.Notification
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		n.ID = id
		out, err := deps.Store.UpsertNotification(n)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := deps.Store.DeleteNotification(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	return r
}
