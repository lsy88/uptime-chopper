package model

import "time"

type MonitorType string

const (
	MonitorTypeHTTP      MonitorType = "http"
	MonitorTypeContainer MonitorType = "container"
)

type RemediationAction string

const (
	RemediationNone    RemediationAction = "none"
	RemediationStart   RemediationAction = "start"
	RemediationRestart RemediationAction = "restart"
)

type RestartPolicyName string

const (
	RestartPolicyNo            RestartPolicyName = "no"
	RestartPolicyAlways        RestartPolicyName = "always"
	RestartPolicyOnFailure     RestartPolicyName = "on-failure"
	RestartPolicyUnlessStopped RestartPolicyName = "unless-stopped"
)

type RemediationPolicy struct {
	Action          RemediationAction `json:"action"`
	MaxAttempts     int               `json:"maxAttempts"`
	CooldownSeconds int               `json:"cooldownSeconds"`
}

type DockerLogOptions struct {
	Include bool `json:"include"`
	Tail    int  `json:"tail"`
}

type Monitor struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Type             MonitorType       `json:"type"`
	IsPaused         bool              `json:"isPaused"`
	IntervalSeconds  int               `json:"intervalSeconds"`
	TimeoutSeconds   int               `json:"timeoutSeconds"`
	NotifyWebhookIDs []string          `json:"notifyWebhookIds"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	HTTP             *HTTPMonitor      `json:"http,omitempty"`
	Container        *ContainerMonitor `json:"container,omitempty"`
	Logs             DockerLogOptions  `json:"logs"`
}

type HTTPMonitor struct {
	URL string `json:"url"`
}

type ContainerMonitor struct {
	ContainerID   string            `json:"containerId"`
	RestartPolicy *RestartPolicy    `json:"restartPolicy,omitempty"`
	Remediation   RemediationPolicy `json:"remediation"`
}

type RestartPolicy struct {
	Name              RestartPolicyName `json:"name"`
	MaximumRetryCount int               `json:"maximumRetryCount"`
}

type MonitorStatus string

const (
	StatusUnknown MonitorStatus = "unknown"
	StatusUp      MonitorStatus = "up"
	StatusDown    MonitorStatus = "down"
	StatusPaused  MonitorStatus = "paused"
)

type CheckResult struct {
	MonitorID string        `json:"monitorId"`
	Status    MonitorStatus `json:"status"`
	CheckedAt time.Time     `json:"checkedAt"`
	LatencyMs int           `json:"latencyMs"`
	Message   string        `json:"message"`
}

type MonitorHistoryEntry struct {
	Status    MonitorStatus `json:"status"`
	CheckedAt time.Time     `json:"checkedAt"`
	LatencyMs int           `json:"latencyMs"`
	Message   string        `json:"message"`
}

type EventType string

const (
	EventStatusChanged EventType = "status_changed"
	EventRemediated    EventType = "remediated"
	EventError         EventType = "error"
)

type MonitorStatusInfo struct {
	Status    MonitorStatus `json:"status"`
	LastCheck time.Time     `json:"lastCheck"`
}

type Event struct {
	ID        string         `json:"id"`
	Type      EventType      `json:"type"`
	MonitorID string         `json:"monitorId"`
	At        time.Time      `json:"at"`
	Data      map[string]any `json:"data"`
}

type Notification struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // webhook, dingtalk, wechat, discord
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
