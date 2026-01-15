package monitor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"go.uber.org/zap"

	"github.com/lsy88/uptime-chopper/internal/config"
	"github.com/lsy88/uptime-chopper/internal/docker"
	"github.com/lsy88/uptime-chopper/internal/model"
	"github.com/lsy88/uptime-chopper/internal/notify"
	"github.com/lsy88/uptime-chopper/internal/store"
)

type EngineDeps struct {
	Logger       *zap.Logger
	Store        store.Store
	Docker       *docker.Client
	Notifier     *notify.Dispatcher
	MaxLogBytes  int
	DefaultSince time.Duration
}

type Engine struct {
	deps EngineDeps

	mu          sync.RWMutex
	lastStatus  map[string]model.MonitorStatus
	lastCheck   map[string]time.Time
	history     map[string][]model.MonitorHistoryEntry
	remediateAt map[string]time.Time
	attempts    map[string]int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewEngine(deps EngineDeps) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		deps:        deps,
		lastStatus:  map[string]model.MonitorStatus{},
		lastCheck:   map[string]time.Time{},
		history:     map[string][]model.MonitorHistoryEntry{},
		remediateAt: map[string]time.Time{},
		attempts:    map[string]int{},
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (e *Engine) Start() {
	e.deps.Logger.Info("monitor engine started")
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.loop()
	}()
}

func (e *Engine) Stop() {
	e.deps.Logger.Info("monitor engine stopping")
	e.cancel()
	e.wg.Wait()
	e.deps.Logger.Info("monitor engine stopped")
}

func (e *Engine) StatusSnapshot() map[string]model.MonitorStatusInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make(map[string]model.MonitorStatusInfo, len(e.lastStatus))
	for k, v := range e.lastStatus {
		out[k] = model.MonitorStatusInfo{
			Status:    v,
			LastCheck: e.lastCheck[k],
		}
	}
	return out
}

func (e *Engine) loop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	nextRun := map[string]time.Time{}

	for {
		select {
		case <-e.ctx.Done():
			return
		case now := <-ticker.C:
			state := e.deps.Store.GetState()
			for _, m := range state.Monitors {
				if m.IsPaused {
					e.setLastStatus(m.ID, model.StatusPaused, now)
					continue
				}
				interval := time.Duration(maxInt(5, m.IntervalSeconds)) * time.Second
				nr, ok := nextRun[m.ID]
				if !ok || !now.Before(nr) {
					nextRun[m.ID] = now.Add(interval)
					e.checkOnce(now, m)
				}
			}
		}
	}
}

func (e *Engine) checkOnce(now time.Time, m model.Monitor) {
	ctx, cancel := context.WithTimeout(e.ctx, time.Duration(maxInt(1, m.TimeoutSeconds))*time.Second)
	defer cancel()

	var res model.CheckResult
	var logs *notify.DockerLogsAttachment
	switch m.Type {
	case model.MonitorTypeHTTP:
		res = checkHTTP(ctx, now, m)
	case model.MonitorTypeContainer:
		res, logs = e.checkContainer(ctx, now, m)
	default:
		res = model.CheckResult{MonitorID: m.ID, Status: model.StatusUnknown, CheckedAt: now, Message: "unknown monitor type"}
	}

	prev := e.getLastStatus(m.ID)
	e.setLastStatus(m.ID, res.Status, now)
	e.appendHistory(m.ID, model.MonitorHistoryEntry{
		Status:    res.Status,
		CheckedAt: res.CheckedAt,
		LatencyMs: res.LatencyMs,
		Message:   res.Message,
	})

	if res.Status == model.StatusUp && prev != model.StatusUp {
		e.resetAttempts(m.ID)
	}

	if prev != res.Status {
		e.deps.Logger.Info("monitor status changed",
			zap.String("monitor_id", m.ID),
			zap.String("monitor_name", m.Name),
			zap.String("previous", string(prev)),
			zap.String("current", string(res.Status)),
			zap.String("message", res.Message),
		)
		e.emitNotification(ctx, m, res, logs, prev)
	}
}

func checkHTTP(ctx context.Context, now time.Time, m model.Monitor) model.CheckResult {
	if m.HTTP == nil || m.HTTP.URL == "" {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, Message: "missing url"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.HTTP.URL, nil)
	if err != nil {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, Message: err.Error()}
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	lat := time.Since(start)
	if err != nil {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, LatencyMs: int(lat.Milliseconds()), Message: err.Error()}
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusUp, CheckedAt: now, LatencyMs: int(lat.Milliseconds()), Message: resp.Status}
	}
	return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, LatencyMs: int(lat.Milliseconds()), Message: resp.Status}
}

func (e *Engine) checkContainer(ctx context.Context, now time.Time, m model.Monitor) (model.CheckResult, *notify.DockerLogsAttachment) {
	if m.Container == nil || m.Container.ContainerID == "" {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, Message: "missing container id"}, nil
	}
	state, err := e.deps.Docker.ContainerState(ctx, m.Container.ContainerID)
	if err != nil {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, Message: err.Error()}, e.tryAttachLogs(ctx, m, now)
	}
	if state == "running" {
		return model.CheckResult{MonitorID: m.ID, Status: model.StatusUp, CheckedAt: now, Message: state}, nil
	}

	e.applyRestartPolicy(ctx, m)
	e.tryRemediate(ctx, now, m)

	return model.CheckResult{MonitorID: m.ID, Status: model.StatusDown, CheckedAt: now, Message: state}, e.tryAttachLogs(ctx, m, now)
}

func (e *Engine) applyRestartPolicy(ctx context.Context, m model.Monitor) {
	if m.Container == nil || m.Container.RestartPolicy == nil {
		return
	}
	p := m.Container.RestartPolicy
	if p.Name == "" {
		return
	}
	_ = e.deps.Docker.UpdateRestartPolicy(ctx, m.Container.ContainerID, container.RestartPolicy{
		Name:              container.RestartPolicyMode(p.Name),
		MaximumRetryCount: p.MaximumRetryCount,
	})
}

func (e *Engine) tryRemediate(ctx context.Context, now time.Time, m model.Monitor) {
	if m.Container == nil {
		return
	}
	p := m.Container.Remediation
	if p.Action == "" || p.Action == model.RemediationNone {
		return
	}
	if p.MaxAttempts <= 0 {
		return
	}

	e.mu.Lock()
	next := e.remediateAt[m.ID]
	if !next.IsZero() && now.Before(next) {
		e.mu.Unlock()
		return
	}
	if e.attempts[m.ID] >= p.MaxAttempts {
		e.mu.Unlock()
		return
	}
	e.attempts[m.ID]++
	e.remediateAt[m.ID] = now.Add(time.Duration(maxInt(5, p.CooldownSeconds)) * time.Second)
	e.mu.Unlock()

	timeout := 10 * time.Second
	var err error
	switch p.Action {
	case model.RemediationStart:
		err = e.deps.Docker.Start(ctx, m.Container.ContainerID)
	case model.RemediationRestart:
		err = e.deps.Docker.Restart(ctx, m.Container.ContainerID, timeout)
	default:
		return
	}
	if err == nil {
		e.deps.Logger.Info("remediation action success",
			zap.String("monitor_id", m.ID),
			zap.String("action", string(p.Action)),
		)
		e.emitWebhookBestEffort(ctx, m, notify.Payload{
			Type:      string(model.EventRemediated),
			MonitorID: m.ID,
			At:        now,
			Data: map[string]any{
				"action":  string(p.Action),
				"attempt": e.getAttempts(m.ID),
			},
		})
	} else {
		e.deps.Logger.Error("remediation action failed",
			zap.String("monitor_id", m.ID),
			zap.String("action", string(p.Action)),
			zap.Error(err),
		)
	}
}

func (e *Engine) tryAttachLogs(ctx context.Context, m model.Monitor, now time.Time) *notify.DockerLogsAttachment {
	if m.Type != model.MonitorTypeContainer || m.Container == nil {
		return nil
	}
	if !m.Logs.Include {
		return nil
	}
	tail := m.Logs.Tail
	if tail <= 0 {
		tail = 200
	}
	since := now.Add(-e.deps.DefaultSince)

	rc, err := e.deps.Docker.Logs(ctx, m.Container.ContainerID, intToTail(tail), since)
	if err != nil {
		return nil
	}
	defer rc.Close()

	lw := newLimitedWriter(e.deps.MaxLogBytes)
	_, _ = stdcopy.StdCopy(lw, lw, rc)
	content := string(lw.Bytes())
	if len(bytes.TrimSpace(lw.Bytes())) == 0 {
		return nil
	}
	return &notify.DockerLogsAttachment{
		ContainerID: m.Container.ContainerID,
		Content:     content,
		Truncated:   lw.Truncated(),
	}
}

func (e *Engine) emitNotification(ctx context.Context, m model.Monitor, res model.CheckResult, logs *notify.DockerLogsAttachment, prev model.MonitorStatus) {
	target := ""
	if m.Type == model.MonitorTypeHTTP && m.HTTP != nil {
		target = m.HTTP.URL
	} else if m.Type == model.MonitorTypeContainer && m.Container != nil {
		target = m.Container.ContainerID
	}

	payload := notify.Payload{
		Type:      string(model.EventStatusChanged),
		MonitorID: m.ID,
		At:        res.CheckedAt,
		Data: map[string]any{
			"monitorName": m.Name,
			"target":      target,
			"previous":    string(prev),
			"current":     string(res.Status),
			"message":     res.Message,
			"latencyMs":   res.LatencyMs,
		},
		Logs: logs,
	}
	e.emitWebhookBestEffort(ctx, m, payload)
}

func (e *Engine) emitWebhookBestEffort(ctx context.Context, m model.Monitor, payload notify.Payload) {
	// 1. Try to find in Store (user configured notifications)
	allNotifs := e.deps.Store.GetNotifications()
	for _, id := range m.NotifyWebhookIDs {
		var found *model.Notification
		// Try match by ID
		for _, n := range allNotifs {
			if n.ID == id {
				v := n // copy
				found = &v
				break
			}
		}
		// Try match by Name (legacy compatibility or user convenience)
		if found == nil {
			for _, n := range allNotifs {
				if n.Name == id {
					v := n
					found = &v
					break
				}
			}
		}

		if found != nil {
			w := config.NotificationWebhook{
				Name: found.Name,
				URL:  found.URL,
				Type: found.Type,
			}
			_ = notify.Send(ctx, e.deps.Notifier.Client(), w, payload)
			continue
		}

		// 2. Fallback to legacy Config-based notifications
		_ = e.deps.Notifier.SendWebhook(ctx, id, payload)
	}
}

func (e *Engine) getLastStatus(id string) model.MonitorStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if v, ok := e.lastStatus[id]; ok {
		return v
	}
	return model.StatusUnknown
}

func (e *Engine) setLastStatus(id string, s model.MonitorStatus, t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastStatus[id] = s
	e.lastCheck[id] = t
}

func (e *Engine) resetAttempts(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.attempts, id)
}

func (e *Engine) getAttempts(id string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.attempts[id]
}

func (e *Engine) appendHistory(id string, entry model.MonitorHistoryEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	hist := e.history[id]
	// Prepend
	hist = append([]model.MonitorHistoryEntry{entry}, hist...)
	// Keep last 50
	if len(hist) > 50 {
		hist = hist[:50]
	}
	e.history[id] = hist
}

func (e *Engine) GetHistory(id string) []model.MonitorHistoryEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	hist := e.history[id]
	if hist == nil {
		return []model.MonitorHistoryEntry{}
	}
	// Return copy
	out := make([]model.MonitorHistoryEntry, len(hist))
	copy(out, hist)
	return out
}

func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intToTail(n int) string {
	if n <= 0 {
		return "all"
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [32]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + (n % 10))
		n /= 10
	}
	return string(buf[i:])
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
