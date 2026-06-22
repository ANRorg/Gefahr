package backend

import (
	"context"
	"io"
	"net/http"
	"time"
)

// HealthPolicy defines active probe timing and state transition thresholds.
type HealthPolicy struct {
	Path               string
	Interval           time.Duration
	Timeout            time.Duration
	HealthyThreshold   int
	UnhealthyThreshold int
}

// Checker performs bounded HTTP health probes for a backend set.
type Checker struct {
	Backends []*Backend
	Policy   HealthPolicy
	Client   *http.Client
	OnChange func(*Backend, bool)
}

// Run probes immediately and then on every interval until ctx is canceled.
func (c *Checker) Run(ctx context.Context) {
	c.CheckOnce(ctx)
	ticker := time.NewTicker(c.Policy.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.CheckOnce(ctx)
		}
	}
}

// CheckOnce probes all backends concurrently and waits for the bounded probes.
func (c *Checker) CheckOnce(ctx context.Context) {
	done := make(chan struct{}, len(c.Backends))
	for _, b := range c.Backends {
		go func(b *Backend) {
			c.check(ctx, b)
			done <- struct{}{}
		}(b)
	}
	for range c.Backends {
		<-done
	}
}

func (c *Checker) check(ctx context.Context, b *Backend) {
	u := b.URL()
	u.Path, u.RawPath, u.RawQuery = c.Policy.Path, "", ""
	probeCtx, cancel := context.WithTimeout(ctx, c.Policy.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, u.String(), nil)
	success := false
	if err == nil {
		client := c.Client
		if client == nil {
			client = http.DefaultClient
		}
		resp, requestErr := client.Do(req)
		if requestErr == nil {
			success = resp.StatusCode >= 200 && resp.StatusCode < 300
			if _, readErr := io.CopyN(io.Discard, resp.Body, (4<<10)+1); readErr != nil && readErr != io.EOF {
				success = false
			}
			resp.Body.Close()
		}
	}
	if b.RecordProbe(success, c.Policy.HealthyThreshold, c.Policy.UnhealthyThreshold) && c.OnChange != nil {
		c.OnChange(b, b.Alive())
	}
}
