package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

var ErrDockerUnavailable = errors.New("docker unavailable")

type Client struct {
	cli     *client.Client
	isMock  bool
	mockMux sync.Mutex
	mockDB  map[string]*ContainerSummary
}

type ContainerSummary struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	State         string            `json:"state"`
	Status        string            `json:"status"`
	Labels        map[string]string `json:"labels"`
	Names         []string          `json:"names"`
	RestartPolicy string            `json:"restart_policy"` // For mock
}

func NewClient() (*Client, error) {
	// Try connecting to real Docker
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	
	useMock := false
	if err == nil {
		// Verify connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := cli.Ping(ctx); err != nil {
			useMock = true
		}
	} else {
		useMock = true
	}

	if useMock {
		// Initialize mock data
		return &Client{
			isMock: true,
			mockDB: map[string]*ContainerSummary{
				"mock-1": {
					ID:     "mock-1",
					Name:   "mock-postgres",
					Names:  []string{"/mock-postgres"},
					Image:  "postgres:15",
					State:  "running",
					Status: "Up 2 hours",
					RestartPolicy: "always",
				},
				"mock-2": {
					ID:     "mock-2",
					Name:   "mock-nginx",
					Names:  []string{"/mock-nginx"},
					Image:  "nginx:latest",
					State:  "exited",
					Status: "Exited (0) 10 minutes ago",
					RestartPolicy: "no",
				},
				"mock-3": {
					ID:     "mock-3",
					Name:   "mock-redis",
					Names:  []string{"/mock-redis"},
					Image:  "redis:alpine",
					State:  "running",
					Status: "Up 5 days",
					RestartPolicy: "on-failure",
				},
			},
		}, nil
	}
	
	return &Client{cli: cli}, nil
}

func (c *Client) ListContainers(ctx context.Context) ([]ContainerSummary, error) {
	if c.isMock {
		c.mockMux.Lock()
		defer c.mockMux.Unlock()
		out := make([]ContainerSummary, 0, len(c.mockDB))
		for _, v := range c.mockDB {
			out = append(out, *v)
		}
		return out, nil
	}

	if c == nil || c.cli == nil {
		return nil, ErrDockerUnavailable
	}
	res, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	out := make([]ContainerSummary, 0, len(res))
	for _, r := range res {
		name := ""
		if len(r.Names) > 0 {
			name = r.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		out = append(out, ContainerSummary{
			ID:     r.ID,
			Name:   name,
			Names:  r.Names,
			Image:  r.Image,
			State:  r.State,
			Status: r.Status,
			Labels: r.Labels,
		})
	}
	return out, nil
}

func (c *Client) ContainerState(ctx context.Context, id string) (string, error) {
	if c.isMock {
		c.mockMux.Lock()
		defer c.mockMux.Unlock()
		if ct, ok := c.mockDB[id]; ok {
			return ct.State, nil
		}
		return "", errors.New("container not found")
	}

	if c == nil || c.cli == nil {
		return "", ErrDockerUnavailable
	}
	ins, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	if ins.State == nil {
		return "", nil
	}
	return ins.State.Status, nil
}

func (c *Client) Start(ctx context.Context, id string) error {
	if c.isMock {
		c.mockMux.Lock()
		defer c.mockMux.Unlock()
		if ct, ok := c.mockDB[id]; ok {
			ct.State = "running"
			ct.Status = "Up (Mock)"
			return nil
		}
		return errors.New("container not found")
	}

	if c == nil || c.cli == nil {
		return ErrDockerUnavailable
	}
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) Stop(ctx context.Context, id string, timeout time.Duration) error {
	if c.isMock {
		c.mockMux.Lock()
		defer c.mockMux.Unlock()
		if ct, ok := c.mockDB[id]; ok {
			ct.State = "exited"
			ct.Status = "Exited (Mock)"
			return nil
		}
		return errors.New("container not found")
	}

	if c == nil || c.cli == nil {
		return ErrDockerUnavailable
	}
	sec := int(timeout.Seconds())
	return c.cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &sec})
}

func (c *Client) Restart(ctx context.Context, id string, timeout time.Duration) error {
	if c.isMock {
		c.mockMux.Lock()
		defer c.mockMux.Unlock()
		if ct, ok := c.mockDB[id]; ok {
			ct.State = "running"
			ct.Status = "Up (Mock Restarted)"
			return nil
		}
		return errors.New("container not found")
	}

	if c == nil || c.cli == nil {
		return ErrDockerUnavailable
	}
	sec := int(timeout.Seconds())
	return c.cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &sec})
}

func (c *Client) UpdateRestartPolicy(ctx context.Context, id string, policy container.RestartPolicy) error {
	if c.isMock {
		c.mockMux.Lock()
		defer c.mockMux.Unlock()
		if ct, ok := c.mockDB[id]; ok {
			ct.RestartPolicy = string(policy.Name)
			return nil
		}
		return errors.New("container not found")
	}

	if c == nil || c.cli == nil {
		return ErrDockerUnavailable
	}
	_, err := c.cli.ContainerUpdate(ctx, id, container.UpdateConfig{RestartPolicy: policy})
	return err
}

func (c *Client) Logs(ctx context.Context, id string, tail string, since time.Time) (io.ReadCloser, error) {
	if c.isMock {
		// Return fake logs
		logs := fmt.Sprintf("[%s] Mock log entry for container %s\n[%s] Another mock log entry...\n[%s] System is running fine.\n[%s] Random value: %d\n",
			time.Now().Format(time.RFC3339), id,
			time.Now().Add(-1*time.Minute).Format(time.RFC3339),
			time.Now().Add(-5*time.Minute).Format(time.RFC3339),
			time.Now().Add(-10*time.Minute).Format(time.RFC3339),
			rand.Intn(100))
		return io.NopCloser(bytes.NewBufferString(logs)), nil
	}

	if c == nil || c.cli == nil {
		return nil, ErrDockerUnavailable
	}
	return c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       tail,
		Since:      since.UTC().Format(time.RFC3339),
		Details:    false,
		Follow:     false,
	})
}

func (c *Client) HasDocker(ctx context.Context) bool {
	if c.isMock {
		return true // Mock always works
	}
	if c == nil || c.cli == nil {
		return false
	}
	_, err := c.cli.Ping(ctx)
	return err == nil
}
