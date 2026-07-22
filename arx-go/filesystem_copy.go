package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startFilesystemCopy(entries []fileEntry, destination string) (tea.Model, tea.Cmd) {
	conflicts, err := filesystemCopyConflicts(entries, destination)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	if len(conflicts) > 0 {
		m.modal = modalConfirm
		m.modalTitle = "Overwrite existing items"
		m.modalMessage = copyConflictMessage(conflicts, destination)
		m.confirm = confirmFilesystemCopy
		m.confirmEntries = append([]fileEntry(nil), entries...)
		m.confirmDestination = destination
		return m, nil
	}
	return m.runFilesystemCopy(entries, destination, false)
}

func (m model) runFilesystemCopy(entries []fileEntry, destination string, overwrite bool) (tea.Model, tea.Cmd) {
	items := append([]fileEntry(nil), entries...)
	return m.startOperation(fmt.Sprintf("Copying %d selected item(s)...", len(items)), func() Result {
		return copyFilesystem(items, destination, overwrite)
	})
}

func copyConflictMessage(conflicts []string, destination string) string {
	const visibleLimit = 6
	visible := conflicts
	if len(visible) > visibleLimit {
		visible = visible[:visibleLimit]
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%d item(s) already exist in %s:\n\n", len(conflicts), destination)
	for _, name := range visible {
		body.WriteString("  " + name + "\n")
	}
	if len(conflicts) > len(visible) {
		fmt.Fprintf(&body, "  ... and %d more\n", len(conflicts)-len(visible))
	}
	body.WriteString("\nOverwrite all listed conflicts?")
	return body.String()
}

func filesystemCopyConflicts(entries []fileEntry, destination string) ([]string, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("select or mark at least one file or directory")
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return nil, err
	}
	destinationInfo, err := os.Stat(destinationAbs)
	if err != nil {
		return nil, err
	}
	if !destinationInfo.IsDir() {
		return nil, fmt.Errorf("copy destination is not a directory: %s", destination)
	}
	resolvedDestination, err := filepath.EvalSymlinks(destinationAbs)
	if err != nil {
		return nil, err
	}

	conflicts := make([]string, 0)
	for _, entry := range entries {
		source, err := filepath.Abs(entry.Path)
		if err != nil {
			return nil, err
		}
		sourceInfo, err := os.Lstat(source)
		if err != nil {
			return nil, err
		}
		name := filepath.Base(source)
		target := filepath.Join(destinationAbs, name)
		if filepath.Clean(source) == filepath.Clean(target) {
			return nil, fmt.Errorf("source and destination are the same: %s", source)
		}
		if sourceInfo.IsDir() {
			resolvedSource, err := filepath.EvalSymlinks(source)
			if err != nil {
				return nil, err
			}
			resolvedTarget := filepath.Join(resolvedDestination, name)
			inside, err := pathWithin(resolvedTarget, resolvedSource)
			if err != nil {
				return nil, err
			}
			if inside {
				return nil, fmt.Errorf("cannot copy a directory into itself: %s", source)
			}
		}
		targetInfo, err := os.Lstat(target)
		if err == nil {
			if os.SameFile(sourceInfo, targetInfo) {
				return nil, fmt.Errorf("source and destination are the same: %s", source)
			}
			conflicts = append(conflicts, name)
			continue
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return conflicts, nil
}

func pathWithin(path, parent string) (bool, error) {
	relative, err := filepath.Rel(parent, path)
	if err != nil {
		return false, err
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))), nil
}

func copyFilesystem(entries []fileEntry, destination string, overwrite bool) Result {
	conflicts, err := filesystemCopyConflicts(entries, destination)
	if err != nil {
		return Result{Err: err}
	}
	if len(conflicts) > 0 && !overwrite {
		return Result{Err: fmt.Errorf("destination already contains: %s", strings.Join(conflicts, ", "))}
	}
	for _, entry := range entries {
		source, err := filepath.Abs(entry.Path)
		if err != nil {
			return Result{Err: err}
		}
		target := filepath.Join(destination, filepath.Base(source))
		if err := copyFilesystemPath(source, target, overwrite); err != nil {
			return Result{Err: fmt.Errorf("copy %s: %w", source, err)}
		}
	}
	return Result{Output: fmt.Sprintf("Copied %d item(s) to %s", len(entries), destination)}
}

func copyFilesystemPath(source, destination string, overwrite bool) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return copyFilesystemSymlink(source, destination, overwrite)
	case info.IsDir():
		return copyFilesystemDirectory(source, destination, info, overwrite)
	case info.Mode().IsRegular():
		return copyFilesystemFile(source, destination, info, overwrite)
	default:
		return fmt.Errorf("unsupported file type")
	}
}

func copyFilesystemDirectory(source, destination string, info os.FileInfo, overwrite bool) error {
	destinationInfo, err := os.Lstat(destination)
	switch {
	case err == nil && !overwrite:
		return os.ErrExist
	case err == nil && !destinationInfo.IsDir():
		stagingRoot, err := os.MkdirTemp(filepath.Dir(destination), ".arx-dir-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(stagingRoot)
		staging := filepath.Join(stagingRoot, "replacement")
		if err := copyFilesystemDirectory(source, staging, info, false); err != nil {
			return err
		}
		return replaceFilesystemPath(staging, destination, true)
	case err == nil:
		// Merge into the existing directory after explicit overwrite confirmation.
	case os.IsNotExist(err):
		if err := os.Mkdir(destination, info.Mode().Perm()); err != nil {
			return err
		}
	default:
		return err
	}

	items, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := copyFilesystemPath(filepath.Join(source, item.Name()), filepath.Join(destination, item.Name()), overwrite); err != nil {
			return err
		}
	}
	if err := os.Chmod(destination, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(destination, info.ModTime(), info.ModTime())
}

func copyFilesystemFile(source, destination string, info os.FileInfo, overwrite bool) error {
	if _, err := os.Lstat(destination); err == nil && !overwrite {
		return os.ErrExist
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".arx-copy-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		temporary.Close()
		return err
	}
	if _, err := io.Copy(temporary, input); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(temporaryName, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	return replaceFilesystemPath(temporaryName, destination, overwrite)
}

func copyFilesystemSymlink(source, destination string, overwrite bool) error {
	linkTarget, err := os.Readlink(source)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil && !overwrite {
		return os.ErrExist
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".arx-link-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Remove(temporaryName); err != nil {
		return err
	}
	defer os.Remove(temporaryName)
	if err := os.Symlink(linkTarget, temporaryName); err != nil {
		return err
	}
	return replaceFilesystemPath(temporaryName, destination, overwrite)
}

func replaceFilesystemPath(source, destination string, overwrite bool) error {
	sourceInfo, err := os.Lstat(source)
	if err != nil {
		return err
	}
	destinationInfo, err := os.Lstat(destination)
	if os.IsNotExist(err) {
		return os.Rename(source, destination)
	}
	if err != nil {
		return err
	}
	if !overwrite {
		return os.ErrExist
	}
	if sourceInfo.IsDir() == destinationInfo.IsDir() && !destinationInfo.IsDir() {
		return os.Rename(source, destination)
	}

	backupFile, err := os.CreateTemp(filepath.Dir(destination), ".arx-backup-*")
	if err != nil {
		return err
	}
	backup := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(backup); err != nil {
		return err
	}
	if err := os.Rename(destination, backup); err != nil {
		return err
	}
	if err := os.Rename(source, destination); err != nil {
		rollbackErr := os.Rename(backup, destination)
		if rollbackErr != nil {
			return fmt.Errorf("replace failed: %w; rollback failed: %v", err, rollbackErr)
		}
		return err
	}
	return os.RemoveAll(backup)
}
