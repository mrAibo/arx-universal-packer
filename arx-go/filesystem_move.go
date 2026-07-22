package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

type filesystemMovePlan struct {
	source string
	target string
}

func (m model) startMove() (tea.Model, tea.Cmd) {
	active := m.panes[m.active]
	passive := m.panes[1-m.active]
	if active.mode != paneFilesystem || passive.mode != paneFilesystem {
		m.showError(fmt.Errorf("move/rename requires filesystem panels"))
		return m, nil
	}
	entries, err := active.operationEntries()
	if err != nil {
		m.showError(err)
		return m, nil
	}
	m.pendingSources = make([]string, 0, len(entries))
	for _, entry := range entries {
		m.pendingSources = append(m.pendingSources, entry.Path)
	}
	m.pendingBaseDir = passive.path
	target := passive.path
	if len(entries) == 1 && filepath.Clean(active.path) == filepath.Clean(passive.path) {
		target = entries[0].Name
	}
	return m.openNavigationInput(navigationInputMove, "Move/Rename", target), nil
}

func (m model) startFilesystemMove(entries []fileEntry, target, baseDir string) (tea.Model, tea.Cmd) {
	return m.startFilesystemMoveWithConflicts(entries, target, baseDir)
}

func (m model) runFilesystemMove(entries []fileEntry, target string, overwrite bool) (tea.Model, tea.Cmd) {
	items := append([]fileEntry(nil), entries...)
	return m.startOperation(fmt.Sprintf("Moving %d selected item(s)...", len(items)), func() Result {
		return moveFilesystem(items, target, overwrite)
	})
}

func normalizeMoveTarget(target, baseDir string) (string, error) {
	target = expandLocation(target)
	if strings.TrimSpace(target) == "" {
		return "", fmt.Errorf("enter a move destination or new name")
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(baseDir, target)
	}
	return filepath.Abs(target)
}

func moveConflictMessage(conflicts []string, target string) string {
	const visibleLimit = 6
	visible := conflicts
	if len(visible) > visibleLimit {
		visible = visible[:visibleLimit]
	}
	var body strings.Builder
	fmt.Fprintf(&body, "%d destination item(s) already exist for %s:\n\n", len(conflicts), target)
	for _, conflict := range visible {
		body.WriteString("  " + conflict + "\n")
	}
	if len(conflicts) > len(visible) {
		fmt.Fprintf(&body, "  ... and %d more\n", len(conflicts)-len(visible))
	}
	body.WriteString("\nReplace all listed conflicts? Existing directories are replaced, not merged.")
	return body.String()
}

func filesystemMovePlans(entries []fileEntry, target string) ([]filesystemMovePlan, []string, error) {
	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("select or mark at least one file or directory")
	}
	targetInfo, targetErr := os.Lstat(target)
	targetIsDirectory := targetErr == nil && targetInfo.IsDir()
	if targetErr != nil && !os.IsNotExist(targetErr) {
		return nil, nil, targetErr
	}
	if len(entries) > 1 && !targetIsDirectory {
		return nil, nil, fmt.Errorf("moving multiple items requires an existing destination directory")
	}
	if !targetIsDirectory {
		parentInfo, err := os.Stat(filepath.Dir(target))
		if err != nil {
			return nil, nil, err
		}
		if !parentInfo.IsDir() {
			return nil, nil, fmt.Errorf("move destination parent is not a directory: %s", filepath.Dir(target))
		}
	}

	plans := make([]filesystemMovePlan, 0, len(entries))
	conflicts := make([]string, 0)
	for _, entry := range entries {
		source, err := filepath.Abs(entry.Path)
		if err != nil {
			return nil, nil, err
		}
		sourceInfo, err := os.Lstat(source)
		if err != nil {
			return nil, nil, err
		}
		if !supportedFilesystemMoveType(sourceInfo.Mode()) {
			return nil, nil, fmt.Errorf("unsupported file type: %s", source)
		}
		destination := target
		if targetIsDirectory {
			destination = filepath.Join(target, filepath.Base(source))
		}
		if filepath.Clean(source) == filepath.Clean(destination) {
			return nil, nil, fmt.Errorf("source and destination are the same: %s", source)
		}
		if sourceInfo.IsDir() {
			resolvedSource, err := filepath.EvalSymlinks(source)
			if err != nil {
				return nil, nil, err
			}
			resolvedDestination, err := resolveProspectivePath(destination)
			if err != nil {
				return nil, nil, err
			}
			inside, err := pathWithin(resolvedDestination, resolvedSource)
			if err != nil {
				return nil, nil, err
			}
			if inside {
				return nil, nil, fmt.Errorf("cannot move a directory into itself: %s", source)
			}
		}
		destinationInfo, err := os.Lstat(destination)
		if err == nil {
			if os.SameFile(sourceInfo, destinationInfo) {
				return nil, nil, fmt.Errorf("source and destination are the same: %s", source)
			}
			conflicts = append(conflicts, filepath.Base(destination))
		} else if !os.IsNotExist(err) {
			return nil, nil, err
		}
		plans = append(plans, filesystemMovePlan{source: source, target: destination})
	}
	return plans, conflicts, nil
}

func supportedFilesystemMoveType(mode os.FileMode) bool {
	return mode.IsRegular() || mode.IsDir() || mode&os.ModeSymlink != 0
}

func resolveProspectivePath(path string) (string, error) {
	if _, err := os.Lstat(path); err == nil {
		return filepath.EvalSymlinks(path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func moveFilesystem(entries []fileEntry, target string, overwrite bool) Result {
	plans, conflicts, err := filesystemMovePlans(entries, target)
	if err != nil {
		return Result{Err: err}
	}
	if len(conflicts) > 0 && !overwrite {
		return Result{Err: fmt.Errorf("destination already contains: %s", strings.Join(conflicts, ", "))}
	}
	for index, plan := range plans {
		if err := moveFilesystemPath(plan.source, plan.target, overwrite); err != nil {
			return Result{Err: fmt.Errorf("move %s: %w (%d of %d completed)", plan.source, err, index, len(plans))}
		}
	}
	return Result{Output: fmt.Sprintf("Moved %d item(s) to %s", len(plans), target)}
}

func moveFilesystemPath(source, target string, overwrite bool) error {
	return moveFilesystemPathWithReplace(source, target, overwrite, replaceFilesystemPath)
}

func moveFilesystemPathWithReplace(source, target string, overwrite bool, replace func(string, string, bool) error) error {
	err := replace(source, target, overwrite)
	if err == nil {
		return nil
	}
	if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	stagingFile, err := os.CreateTemp(filepath.Dir(target), ".arx-move-*")
	if err != nil {
		return err
	}
	staging := stagingFile.Name()
	if err := stagingFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(staging); err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err := copyFilesystemPath(source, staging, false); err != nil {
		return err
	}
	if err := replaceFilesystemPath(staging, target, overwrite); err != nil {
		return err
	}
	if err := os.RemoveAll(source); err != nil {
		return fmt.Errorf("destination committed but source cleanup failed: %w", err)
	}
	return nil
}
