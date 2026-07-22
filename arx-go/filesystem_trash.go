package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startFilesystemTrash() (tea.Model, tea.Cmd) {
	pane := m.panes[m.active]
	if pane.mode != paneFilesystem {
		return m.startArchiveDelete()
	}
	entries, err := pane.operationEntries()
	if err != nil {
		m.showError(err)
		return m, nil
	}
	files, directories, bytes, err := filesystemTrashSummary(entries)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	m.modal = modalConfirm
	m.modalTitle = "Move to trash"
	m.modalMessage = fmt.Sprintf("Move %d selected item(s) to trash?\n\n%d file(s), %d directorie(s), %s\n\nItems can be restored from the desktop trash.", len(entries), files, directories, formatSize(bytes))
	m.confirm = confirmFilesystemTrash
	m.confirmEntries = append([]fileEntry(nil), entries...)
	return m, nil
}

func (m model) runFilesystemTrash(entries []fileEntry) (tea.Model, tea.Cmd) {
	items := append([]fileEntry(nil), entries...)
	return m.startOperation(fmt.Sprintf("Moving %d selected item(s) to trash...", len(items)), func() Result {
		return trashFilesystem(items)
	})
}

func filesystemTrashSummary(entries []fileEntry) (files, directories int, bytes int64, err error) {
	for _, entry := range entries {
		entryFiles, entryDirectories, entryBytes, walkErr := filesystemPathSummary(entry.Path)
		if walkErr != nil {
			return 0, 0, 0, walkErr
		}
		files += entryFiles
		directories += entryDirectories
		bytes += entryBytes
	}
	return files, directories, bytes, nil
}

func filesystemPathSummary(path string) (files, directories int, bytes int64, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, 0, 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 1, 0, info.Size(), nil
	}
	if !info.IsDir() {
		return 1, 0, info.Size(), nil
	}
	err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if entry.IsDir() {
			directories++
			return nil
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, directories, bytes, err
}

func trashFilesystem(entries []fileEntry) Result {
	filesDir, infoDir, err := trashDirectories()
	if err != nil {
		return Result{Err: err}
	}
	completed := 0
	records := make([]trashRecord, 0, len(entries))
	for _, entry := range entries {
		record, err := trashFilesystemPathRecord(entry.Path, filesDir, infoDir, time.Now())
		if err != nil {
			setLastTrashRecords(records)
			return Result{Err: fmt.Errorf("trash %s: %w (%d of %d completed)", entry.Path, err, completed, len(entries))}
		}
		records = append(records, record)
		completed++
	}
	setLastTrashRecords(records)
	return Result{Output: fmt.Sprintf("Moved %d item(s) to trash", completed)}
}

func trashDirectories() (string, string, error) {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	root := filepath.Join(dataHome, "Trash")
	filesDir := filepath.Join(root, "files")
	infoDir := filepath.Join(root, "info")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return "", "", err
	}
	return filesDir, infoDir, nil
}

func trashFilesystemPath(path, filesDir, infoDir string, deletedAt time.Time) error {
	_, err := trashFilesystemPathRecord(path, filesDir, infoDir, deletedAt)
	return err
}

func trashFilesystemPathRecord(path, filesDir, infoDir string, deletedAt time.Time) (trashRecord, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return trashRecord{}, err
	}
	if _, err := os.Lstat(absolute); err != nil {
		return trashRecord{}, err
	}
	_, dataPath, infoPath, err := availableTrashName(filepath.Base(absolute), filesDir, infoDir)
	if err != nil {
		return trashRecord{}, err
	}
	infoContent := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", url.PathEscape(absolute), deletedAt.Format("2006-01-02T15:04:05"))
	temporary, err := os.CreateTemp(infoDir, ".arx-trashinfo-*")
	if err != nil {
		return trashRecord{}, err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return trashRecord{}, err
	}
	if _, err := temporary.WriteString(infoContent); err != nil {
		temporary.Close()
		return trashRecord{}, err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return trashRecord{}, err
	}
	if err := temporary.Close(); err != nil {
		return trashRecord{}, err
	}
	if err := moveFilesystemPath(absolute, dataPath, false); err != nil {
		return trashRecord{}, err
	}
	if err := os.Rename(temporaryName, infoPath); err != nil {
		rollbackErr := moveFilesystemPath(dataPath, absolute, false)
		if rollbackErr != nil {
			return trashRecord{}, fmt.Errorf("write trash metadata: %w; rollback failed: %v", err, rollbackErr)
		}
		return trashRecord{}, err
	}
	return trashRecord{originalPath: absolute, dataPath: dataPath, infoPath: infoPath}, nil
}

func availableTrashName(base, filesDir, infoDir string) (string, string, string, error) {
	if base == "" || base == "." || base == ".." {
		base = "item"
	}
	for index := 0; ; index++ {
		name := base
		if index > 0 {
			name = fmt.Sprintf("%s.%d", base, index)
		}
		dataPath := filepath.Join(filesDir, name)
		infoPath := filepath.Join(infoDir, name+".trashinfo")
		_, dataErr := os.Lstat(dataPath)
		_, infoErr := os.Lstat(infoPath)
		if os.IsNotExist(dataErr) && os.IsNotExist(infoErr) {
			return name, dataPath, infoPath, nil
		}
		if dataErr != nil && !os.IsNotExist(dataErr) {
			return "", "", "", dataErr
		}
		if infoErr != nil && !os.IsNotExist(infoErr) {
			return "", "", "", infoErr
		}
	}
}
