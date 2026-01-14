package api

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors(deps.Config.AllowedCORSOrigin))

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		})
		r.Mount("/monitors", monitorsRouter(deps))
		r.Mount("/containers", containersRouter(deps))
		r.Get("/status", deps.handleStatus)
		r.Mount("/notifications", notificationsRouter(deps))
	})

	if deps.Config.ServeFrontendFromDist {
		distDir := deps.Config.FrontendDistDirectory
		if distDir == "" {
			distDir = "web/dist"
		}
		if fi, err := os.Stat(distDir); err == nil && fi.IsDir() {
			r.Handle("/*", spaFileServer(distDir))
		}
	}

	return r
}

func spaFileServer(distDir string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(distDir))
	index := filepath.Join(distDir, "index.html")
	return func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(distDir, filepath.Clean(r.URL.Path))
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	}
}
