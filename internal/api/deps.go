package api

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/lsy88/uptime-chopper/internal/config"
	"github.com/lsy88/uptime-chopper/internal/docker"
	"github.com/lsy88/uptime-chopper/internal/monitor"
	"github.com/lsy88/uptime-chopper/internal/store"
)

type Deps struct {
	Logger *zap.Logger
	Store  store.Store
	Docker *docker.Client
	Engine *monitor.Engine
	Config *config.Config
}

func (d Deps) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := d.Engine.StatusSnapshot()
	writeJSON(w, http.StatusOK, map[string]any{"status": status})
}
