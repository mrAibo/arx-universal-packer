package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const operationTickInterval = 150 * time.Millisecond

type operationTickMsg struct{}

type operationSnapshot struct {
	label     string
	current   string
	started   time.Time
	completed int
	total     int
	bytes     int64
	cancelled bool
}

type operationState struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu        sync.RWMutex
	label     string
	current   string
	started   time.Time
	completed int
	total     int
	bytes     int64
	cancelled bool
}

var operationRegistry struct {
	sync.RWMutex
	active *operationState
}

func beginOperation(label string, total int) *operationState {
	ctx, cancel := context.WithCancel(context.Background())
	state := &operationState{
		ctx:     ctx,
		cancel:  cancel,
		label:   label,
		started: time.Now(),
		total:   total,
	}
	operationRegistry.Lock()
	operationRegistry.active = state
	operationRegistry.Unlock()
	return state
}

func finishOperation(state *operationState) {
	if state == nil {
		return
	}
	state.cancel()
	operationRegistry.Lock()
	if operationRegistry.active == state {
		operationRegistry.active = nil
	}
	operationRegistry.Unlock()
}

func currentOperation() *operationState {
	operationRegistry.RLock()
	defer operationRegistry.RUnlock()
	return operationRegistry.active
}

func operationContext() context.Context {
	if state := currentOperation(); state != nil {
		return state.ctx
	}
	return context.Background()
}

func cancelCurrentOperation() bool {
	state := currentOperation()
	if state == nil {
		return false
	}
	state.mu.Lock()
	state.cancelled = true
	state.mu.Unlock()
	state.cancel()
	return true
}

func isOperationCancelled(err error) bool {
	return errors.Is(err, context.Canceled)
}

func operationCancelled() error {
	select {
	case <-operationContext().Done():
		return context.Canceled
	default:
		return nil
	}
}

func setOperationItem(name string) {
	if state := currentOperation(); state != nil {
		state.mu.Lock()
		state.current = name
		state.mu.Unlock()
	}
}

func advanceOperation() {
	if state := currentOperation(); state != nil {
		state.mu.Lock()
		state.completed++
		state.mu.Unlock()
	}
}

func addOperationBytes(count int64) {
	if state := currentOperation(); state != nil {
		state.mu.Lock()
		state.bytes += count
		state.mu.Unlock()
	}
}

func snapshotOperation() (operationSnapshot, bool) {
	state := currentOperation()
	if state == nil {
		return operationSnapshot{}, false
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	return operationSnapshot{
		label:     state.label,
		current:   state.current,
		started:   state.started,
		completed: state.completed,
		total:     state.total,
		bytes:     state.bytes,
		cancelled: state.cancelled,
	}, true
}

func formatOperationStatus(snapshot operationSnapshot) string {
	elapsed := time.Since(snapshot.started).Round(time.Second)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed < time.Second {
		elapsed = 0
	}
	status := snapshot.label
	if snapshot.total > 0 {
		status = fmt.Sprintf("%s %d/%d", status, snapshot.completed, snapshot.total)
	}
	if snapshot.current != "" {
		status += " · " + snapshot.current
	}
	if snapshot.bytes > 0 {
		status += " · " + formatOperationBytes(snapshot.bytes)
	}
	status += fmt.Sprintf(" · %s · Esc cancels", elapsed)
	if snapshot.cancelled {
		status += " (cancelling…)"
	}
	return status
}

func formatOperationBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KiB", "MiB", "GiB", "TiB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f PiB", value/unit)
}

func operationTick() tea.Cmd {
	return tea.Tick(operationTickInterval, func(time.Time) tea.Msg {
		return operationTickMsg{}
	})
}

func (m model) startOperation(status string, fn func() Result) (tea.Model, tea.Cmd) {
	return m.startOperationProgress(status, 0, fn)
}

func (m model) startOperationProgress(status string, total int, fn func() Result) (tea.Model, tea.Cmd) {
	state := beginOperation(status, total)
	m.busy = true
	m.status = status + " · Esc cancels"
	work := func() tea.Msg {
		result := fn()
		if errors.Is(result.Err, context.Canceled) {
			result.Err = context.Canceled
		}
		return operationMsg{result: result, state: state}
	}
	return m, tea.Batch(work, operationTick())
}

type cancellableReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader cancellableReader) Read(buffer []byte) (int, error) {
	select {
	case <-reader.ctx.Done():
		return 0, context.Canceled
	default:
	}
	count, err := reader.reader.Read(buffer)
	if count > 0 {
		addOperationBytes(int64(count))
	}
	return count, err
}

func copyWithOperationProgress(writer io.Writer, reader io.Reader) (int64, error) {
	return io.Copy(writer, cancellableReader{ctx: operationContext(), reader: reader})
}
