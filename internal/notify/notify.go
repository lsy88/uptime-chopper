package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lsy88/uptime-chopper/internal/config"
)

type Payload struct {
	Type      string                `json:"type"`
	MonitorID string                `json:"monitorId"`
	At        time.Time             `json:"at"`
	Data      map[string]any        `json:"data"`
	Logs      *DockerLogsAttachment `json:"logs,omitempty"`
}

type DockerLogsAttachment struct {
	ContainerID string `json:"containerId"`
	Content     string `json:"content"`
	Truncated   bool   `json:"truncated"`
}

type Dispatcher struct {
	webhooks map[string]config.NotificationWebhook
	client   *http.Client
}

func NewDispatcher(webhooks []config.NotificationWebhook) *Dispatcher {
	m := make(map[string]config.NotificationWebhook, len(webhooks))
	for _, w := range webhooks {
		if w.Name == "" || w.URL == "" {
			continue
		}
		m[w.Name] = w
	}
	return &Dispatcher{
		webhooks: m,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *Dispatcher) Client() *http.Client {
	return d.client
}

func (d *Dispatcher) SendWebhook(ctx context.Context, webhookName string, payload Payload) error {
	w, ok := d.webhooks[webhookName]
	if !ok {
		return nil
	}
	return Send(ctx, d.client, w, payload)
}

func Send(ctx context.Context, client *http.Client, w config.NotificationWebhook, payload Payload) error {
	var body []byte
	var err error

	switch w.Type {
	case "dingtalk":
		body, err = buildDingTalkPayload(payload)
	case "wechat":
		body, err = buildWeChatPayload(payload)
	case "discord":
		body, err = buildDiscordPayload(payload)
	default:
		// Default to generic webhook
		body, err = json.Marshal(payload)
	}

	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook %s returned status %d: %s", w.Name, resp.StatusCode, string(respBody))
	}

	// For DingTalk, check errcode
	if w.Type == "dingtalk" {
		var dtResp struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(respBody, &dtResp); err == nil {
			if dtResp.ErrCode != 0 {
				return fmt.Errorf("dingtalk error %d: %s", dtResp.ErrCode, dtResp.ErrMsg)
			}
		}
	}

	return nil
}

func buildDingTalkPayload(p Payload) ([]byte, error) {
	title := fmt.Sprintf("Monitor Alert: %s", p.Type)
	text := formatMarkdown(title, p)

	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
	}
	return json.Marshal(payload)
}

func buildWeChatPayload(p Payload) ([]byte, error) {
	title := fmt.Sprintf("Monitor Alert: %s", p.Type)
	text := formatMarkdown(title, p)

	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": text,
		},
	}
	return json.Marshal(payload)
}

func buildDiscordPayload(p Payload) ([]byte, error) {
	title := fmt.Sprintf("Monitor Alert: %s", p.Type)
	description := formatMarkdown(title, p)

	color := 0x5cdd8b // Green
	if p.Type == "down" {
		color = 0xdc3545 // Red
	} else if p.Type == "pending" {
		color = 0xffc107 // Yellow
	}

	payload := map[string]any{
		"username": "Uptime Chopper",
		"embeds": []map[string]any{
			{
				"title":       title,
				"description": description,
				"color":       color,
				"timestamp":   p.At.Format(time.RFC3339),
			},
		},
	}
	return json.Marshal(payload)
}

func formatMarkdown(title string, p Payload) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("## %s\n", title))
	buf.WriteString(fmt.Sprintf("**Monitor ID**: %s\n", p.MonitorID))
	buf.WriteString(fmt.Sprintf("**Time**: %s\n", p.At.Format(time.RFC3339)))

	if msg, ok := p.Data["message"].(string); ok {
		buf.WriteString(fmt.Sprintf("**Message**: %s\n", msg))
	}

	// Add other data fields
	for k, v := range p.Data {
		if k == "message" {
			continue
		}
		buf.WriteString(fmt.Sprintf("- **%s**: %v\n", k, v))
	}

	if p.Logs != nil {
		buf.WriteString("\n**Container Logs**:\n")
		buf.WriteString("```\n")
		// Limit log length for markdown to avoid message too long errors
		content := p.Logs.Content
		if len(content) > 1000 {
			content = content[len(content)-1000:]
			buf.WriteString("...(truncated)...\n")
		}
		buf.WriteString(content)
		buf.WriteString("\n```\n")
	}

	return buf.String()
}
