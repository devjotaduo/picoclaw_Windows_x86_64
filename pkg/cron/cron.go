// Package cron provides a tiny in-memory scheduler for one-shot and recurring
// jobs. Schedules are expressed as "in <dur>", "every <dur>", or an RFC3339
// timestamp. It is intentionally minimal; cron-expression support can layer on
// later without changing the Job model.
package cron

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind distinguishes a one-shot job from a recurring one.
type Kind string

const (
	Once     Kind = "once"
	Interval Kind = "interval"
)

// Job is a scheduled unit of work.
type Job struct {
	ID       string        `json:"id"`
	Prompt   string        `json:"prompt"`
	Kind     Kind          `json:"kind"`
	Spec     string        `json:"spec"`
	Interval time.Duration `json:"-"`
	NextRun  time.Time     `json:"next_run"`
	Enabled  bool          `json:"enabled"`
}

// Handler runs when a job fires.
type Handler func(ctx context.Context, job Job)

// Scheduler holds jobs and fires them on time.
type Scheduler struct {
	mu      sync.Mutex
	jobs    map[string]*Job
	seq     int
	handler Handler
}

// New builds an empty scheduler.
func New() *Scheduler { return &Scheduler{jobs: map[string]*Job{}} }

// SetHandler sets the function invoked when a job fires.
func (s *Scheduler) SetHandler(h Handler) { s.handler = h }

// Add parses spec and registers a job, returning it.
func (s *Scheduler) Add(spec, prompt string) (Job, error) {
	kind, interval, next, err := parseSpec(spec)
	if err != nil {
		return Job{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := fmt.Sprintf("job-%d", s.seq)
	j := &Job{
		ID:       id,
		Prompt:   prompt,
		Kind:     kind,
		Spec:     spec,
		Interval: interval,
		NextRun:  next,
		Enabled:  true,
	}
	s.jobs[id] = j
	return *j, nil
}

// List returns all jobs sorted by next run time.
func (s *Scheduler) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NextRun.Before(out[j].NextRun) })
	return out
}

// Remove deletes a job by id.
func (s *Scheduler) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return false
	}
	delete(s.jobs, id)
	return true
}

// SetEnabled toggles a job.
func (s *Scheduler) SetEnabled(id string, enabled bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return false
	}
	j.Enabled = enabled
	return true
}

// Run ticks every second, firing due jobs until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			for _, j := range s.due(now) {
				if s.handler != nil {
					go s.handler(ctx, j)
				}
			}
		}
	}
}

// due returns jobs whose time has come, advancing or removing them.
func (s *Scheduler) due(now time.Time) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	var fired []Job
	for id, j := range s.jobs {
		if !j.Enabled || now.Before(j.NextRun) {
			continue
		}
		fired = append(fired, *j)
		if j.Kind == Interval {
			j.NextRun = now.Add(j.Interval)
		} else {
			delete(s.jobs, id)
		}
	}
	return fired
}

func parseSpec(spec string) (Kind, time.Duration, time.Time, error) {
	spec = strings.TrimSpace(spec)
	now := time.Now()
	switch {
	case strings.HasPrefix(spec, "in "):
		d, err := time.ParseDuration(strings.TrimSpace(spec[3:]))
		if err != nil {
			return "", 0, time.Time{}, fmt.Errorf("bad duration: %w", err)
		}
		return Once, 0, now.Add(d), nil
	case strings.HasPrefix(spec, "every "):
		d, err := time.ParseDuration(strings.TrimSpace(spec[6:]))
		if err != nil {
			return "", 0, time.Time{}, fmt.Errorf("bad interval: %w", err)
		}
		if d < time.Second {
			return "", 0, time.Time{}, fmt.Errorf("interval too small")
		}
		return Interval, d, now.Add(d), nil
	default:
		t, err := time.Parse(time.RFC3339, spec)
		if err != nil {
			return "", 0, time.Time{}, fmt.Errorf(`spec must be "in <dur>", "every <dur>", or RFC3339`)
		}
		return Once, 0, t, nil
	}
}
