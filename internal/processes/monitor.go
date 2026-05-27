package processes

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nathanmauro/agent-observatory/internal/events"
	"github.com/nathanmauro/agent-observatory/internal/models"
)

var agentNames = map[string]models.AgentType{
	"claude":  models.AgentClaude,
	"codex":   models.AgentCodex,
	"auggie":  models.AgentAuggie,
	"cursor":  models.AgentCursor,
	"Cursor":  models.AgentCursor,
	"copilot": "copilot",
}

type Monitor struct {
	bus      *events.Bus
	mu       sync.RWMutex
	current  []models.Process
	stopOnce sync.Once
	done     chan struct{}
}

func NewMonitor(bus *events.Bus) *Monitor {
	return &Monitor{
		bus:  bus,
		done: make(chan struct{}),
	}
}

func (m *Monitor) Snapshot() []models.Process {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]models.Process, len(m.current))
	copy(out, m.current)
	return out
}

func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	first := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.done:
			return
		case <-ticker.C:
			procs, err := poll()
			if err != nil {
				log.Printf("process poll: %v", err)
				continue
			}

			m.mu.Lock()
			prev := m.current
			m.current = procs
			m.mu.Unlock()

			if first {
				m.bus.Publish(events.Event{
					Type:  "process.snapshot",
					Topic: "processes",
					Data:  procs,
				})
				first = false
			} else if !processesEqual(prev, procs) {
				m.bus.Publish(events.Event{
					Type:  "process.diff",
					Topic: "processes",
					Data:  procs,
				})
			}
		}
	}
}

func (m *Monitor) Stop() {
	m.stopOnce.Do(func() { close(m.done) })
}

func poll() ([]models.Process, error) {
	cmd := exec.Command("ps", "-eo", "pid,ppid,rss,pcpu,comm")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var procs []models.Process
	scanner := bufio.NewScanner(&buf)
	scanner.Scan() // skip header
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		comm := fields[4]
		base := comm
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}

		agentType, ok := agentNames[base]
		if !ok {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		ppid, _ := strconv.Atoi(fields[1])
		rss, _ := strconv.ParseInt(fields[2], 10, 64)
		cpu, _ := strconv.ParseFloat(fields[3], 64)

		procs = append(procs, models.Process{
			PID:        pid,
			PPID:       ppid,
			Name:       base,
			AgentType:  agentType,
			Command:    comm,
			CPUPercent: cpu,
			RSSBytes:   rss * 1024, // ps reports RSS in KB
			Status:     "running",
		})
	}
	return procs, scanner.Err()
}

func processesEqual(a, b []models.Process) bool {
	if len(a) != len(b) {
		return false
	}
	pids := make(map[int]models.Process, len(a))
	for _, p := range a {
		pids[p.PID] = p
	}
	for _, p := range b {
		prev, ok := pids[p.PID]
		if !ok {
			return false
		}
		if prev.Name != p.Name || prev.Status != p.Status {
			return false
		}
	}
	return true
}
