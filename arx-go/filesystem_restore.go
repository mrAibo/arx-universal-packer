package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type trashRecord struct {
	originalPath string
	dataPath     string
	infoPath     string
}

var lastTrash struct {
	sync.Mutex
	records []trashRecord
}

func setLastTrashRecords(records []trashRecord) {
	lastTrash.Lock()
	defer lastTrash.Unlock()
	lastTrash.records = append([]trashRecord(nil), records...)
}

func lastTrashRecords() []trashRecord {
	lastTrash.Lock()
	defer lastTrash.Unlock()
	return append([]trashRecord(nil), lastTrash.records...)
}

func (m model) startRestoreLastTrash() (tea.Model, tea.Cmd) {
	records := lastTrashRecords()
	if len(records) == 0 {
		m.showError(fmt.Errorf("no ARX trash operation is available to undo"))
		return m, nil
	}
	m.modal = modalConfirm
	m.modalTitle = "Restore from trash"
	m.modalMessage = fmt.Sprintf("Restore %d item(s) to their original locations?\n\nExisting paths will not be overwritten.", len(records))
	m.confirm = confirmFilesystemRestore
	return m, nil
}

func (m model) runFilesystemRestore() (tea.Model, tea.Cmd) {
	records := lastTrashRecords()
	return m.startOperation(fmt.Sprintf("Restoring %d item(s) from trash...", len(records)), func() Result {
		return restoreTrashRecords(records)
	})
}

func restoreTrashRecords(records []trashRecord) Result {
	restored := 0
	for index, record := range records {
		if err := restoreTrashRecord(record); err != nil {
			setLastTrashRecords(records[index:])
			return Result{Err: fmt.Errorf("restore %s: %w (%d of %d restored)", record.originalPath, err, restored, len(records))}
		}
		restored++
	}
	setLastTrashRecords(nil)
	return Result{Output: fmt.Sprintf("Restored %d item(s) from trash", restored)}
}

func restoreTrashRecord(record trashRecord) error {
	if record.originalPath == "" || record.dataPath == "" || record.infoPath == "" {
		return fmt.Errorf("invalid trash record")
	}
	if _, err := os.Lstat(record.originalPath); err == nil {
		return fmt.Errorf("original path already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Lstat(record.dataPath); err != nil {
		return fmt.Errorf("trashed item is missing: %w", err)
	}
	if _, err := os.Lstat(record.infoPath); err != nil {
		return fmt.Errorf("trash metadata is missing: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(record.originalPath), 0o755); err != nil {
		return err
	}
	if err := moveFilesystemPath(record.dataPath, record.originalPath, false); err != nil {
		return err
	}
	if err := os.Remove(record.infoPath); err != nil {
		rollbackErr := moveFilesystemPath(record.originalPath, record.dataPath, false)
		if rollbackErr != nil {
			return fmt.Errorf("remove trash metadata: %w; rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("remove trash metadata: %w", err)
	}
	return nil
}
