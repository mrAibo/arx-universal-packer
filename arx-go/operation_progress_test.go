package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOperationProgressSnapshot(t *testing.T) {
	state := beginOperation("Copying", 3)
	t.Cleanup(func() { finishOperation(state) })

	setOperationItem("alpha.txt")
	addOperationBytes(1536)
	advanceOperation()

	snapshot, ok := snapshotOperation()
	if !ok {
		t.Fatal("expected active operation")
	}
	if snapshot.completed != 1 || snapshot.total != 3 || snapshot.current != "alpha.txt" || snapshot.bytes != 1536 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	status := formatOperationStatus(snapshot)
	for _, part := range []string{"Copying 1/3", "alpha.txt", "1.5 KiB", "Esc cancels"} {
		if !strings.Contains(status, part) {
			t.Fatalf("status %q does not contain %q", status, part)
		}
	}
}

func TestCancelCurrentOperation(t *testing.T) {
	state := beginOperation("Moving", 1)
	t.Cleanup(func() { finishOperation(state) })

	if !cancelCurrentOperation() {
		t.Fatal("cancel did not find active operation")
	}
	if err := operationCancelled(); !errors.Is(err, context.Canceled) {
		t.Fatalf("operationCancelled=%v", err)
	}
	snapshot, ok := snapshotOperation()
	if !ok || !snapshot.cancelled {
		t.Fatalf("snapshot=%+v ok=%v", snapshot, ok)
	}
}

func TestCancellableReaderStopsAfterCancel(t *testing.T) {
	state := beginOperation("Copying", 1)
	finish := func() { finishOperation(state) }
	t.Cleanup(finish)
	cancelCurrentOperation()

	reader := cancellableReader{ctx: state.ctx, reader: strings.NewReader("data")}
	buffer := make([]byte, 4)
	if _, err := reader.Read(buffer); !errors.Is(err, context.Canceled) {
		t.Fatalf("Read error=%v", err)
	}
}

func TestOperationElapsedNeverNegative(t *testing.T) {
	status := formatOperationStatus(operationSnapshot{label: "Testing", started: time.Now().Add(time.Second)})
	if strings.Contains(status, "-") {
		t.Fatalf("negative elapsed status: %q", status)
	}
}
