// Package activity tracks ephemeral background work; no paths or credentials are retained.
package activity

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

type Task struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	StartedAt      time.Time `json:"startedAt"`
	RuntimeSeconds float64   `json:"runtimeSeconds"`
	Completed      int       `json:"completed"`
	Total          int       `json:"total"`
	Percent        *float64  `json:"percent"`
	Unit           string    `json:"unit"`
	CurrentFile    string    `json:"currentFile,omitempty"`
}
type Tracker struct {
	mu    sync.Mutex
	next  uint64
	tasks map[string]Task
}

var Default = &Tracker{tasks: map[string]Task{}}

func (t *Tracker) Start(name, unit string, total int) (string, func()) {
	t.mu.Lock()
	t.next++
	id := strconv.FormatUint(t.next, 10)
	if t.tasks == nil {
		t.tasks = map[string]Task{}
	}
	t.tasks[id] = Task{ID: id, Name: name, Unit: unit, Total: total, StartedAt: time.Now()}
	t.mu.Unlock()
	return id, func() { t.mu.Lock(); delete(t.tasks, id); t.mu.Unlock() }
}
func (t *Tracker) SetCurrentFile(id, name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if task, ok := t.tasks[id]; ok {
		task.CurrentFile = name
		t.tasks[id] = task
	}
}

func (t *Tracker) Progress(id string, completed, total int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if v, ok := t.tasks[id]; ok {
		v.Completed = max(0, completed)
		v.Total = max(0, total)
		if v.Total > 0 {
			v.Completed = min(v.Completed, v.Total)
		}
		t.tasks[id] = v
	}
}
func (t *Tracker) Snapshot(now time.Time) []Task {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]Task, 0, len(t.tasks))
	for _, v := range t.tasks {
		v.RuntimeSeconds = max(0, now.Sub(v.StartedAt).Seconds())
		if v.Total > 0 {
			p := min(100, max(0, 100*float64(v.Completed)/float64(v.Total)))
			v.Percent = &p
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out
}
