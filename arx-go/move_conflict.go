package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const modalMoveConflict modalKind = 301

type filesystemMoveDecisionPlan struct {
	source    string
	target    string
	overwrite bool
	skip      bool
	conflict  bool
}

func (m model) startFilesystemMoveWithConflicts(entries []fileEntry, target, baseDir string) (tea.Model, tea.Cmd) {
	target, err := normalizeMoveTarget(target, baseDir)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	plans, err := filesystemMoveDecisionPlans(entries, target)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	first := nextMoveConflict(plans, 0)
	if first < 0 {
		return m.runFilesystemMoveDecisionPlans(plans, target)
	}
	m.modal = modalMoveConflict
	m.modalTitle = "Move conflict"
	m.movePlans = plans
	m.moveConflictIndex = first
	m.moveConflictAction = copyConflictReplace
	m.moveConflictApplyAll = false
	m.moveConflictRename = suggestedCopyName(plans[first].target)
	m.moveConflictTarget = target
	return m, nil
}

func filesystemMoveDecisionPlans(entries []fileEntry, target string) ([]filesystemMoveDecisionPlan, error) {
	plans, _, err := filesystemMovePlans(entries, target)
	if err != nil {
		return nil, err
	}
	result := make([]filesystemMoveDecisionPlan, 0, len(plans))
	for _, plan := range plans {
		_, statErr := os.Lstat(plan.target)
		result = append(result, filesystemMoveDecisionPlan{
			source:   plan.source,
			target:   plan.target,
			conflict: statErr == nil,
		})
		if statErr != nil && !os.IsNotExist(statErr) {
			return nil, statErr
		}
	}
	return result, nil
}

func (m model) runFilesystemMoveDecisionPlans(plans []filesystemMoveDecisionPlan, target string) (tea.Model, tea.Cmd) {
	items := append([]filesystemMoveDecisionPlan(nil), plans...)
	return m.startOperation(fmt.Sprintf("Moving %d selected item(s)...", len(items)), func() Result {
		return moveFilesystemDecisionPlans(items, target)
	})
}

func moveFilesystemDecisionPlans(plans []filesystemMoveDecisionPlan, target string) Result {
	moved := 0
	skipped := 0
	for _, plan := range plans {
		if plan.skip {
			skipped++
			continue
		}
		if err := moveFilesystemPath(plan.source, plan.target, plan.overwrite); err != nil {
			return Result{Err: fmt.Errorf("move %s: %w (%d moved, %d skipped)", plan.source, err, moved, skipped)}
		}
		moved++
	}
	return Result{Output: fmt.Sprintf("Moved %d item(s) to %s, skipped %d", moved, target, skipped)}
}

func nextMoveConflict(plans []filesystemMoveDecisionPlan, start int) int {
	for index := start; index < len(plans); index++ {
		if plans[index].conflict && !plans[index].skip && !plans[index].overwrite {
			return index
		}
	}
	return -1
}

func (m model) updateMoveConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.moveConflictIndex < 0 || m.moveConflictIndex >= len(m.movePlans) {
		m.closeModal()
		return m, nil
	}
	key := msg.String()
	if m.moveConflictAction == copyConflictRename {
		switch key {
		case "backspace":
			runes := []rune(m.moveConflictRename)
			if len(runes) > 0 {
				m.moveConflictRename = string(runes[:len(runes)-1])
			}
			return m, nil
		case "ctrl+u":
			m.moveConflictRename = ""
			return m, nil
		}
		if msg.Type == tea.KeyRunes {
			m.moveConflictRename += string(msg.Runes)
			return m, nil
		}
	}

	switch key {
	case "esc", "q":
		m.closeModal()
		return m, nil
	case "left", "up", "shift+tab":
		m.moveConflictAction = copyConflictAction(wrapIndex(int(m.moveConflictAction)-1, len(copyConflictActions())))
	case "right", "down", "tab":
		m.moveConflictAction = copyConflictAction(wrapIndex(int(m.moveConflictAction)+1, len(copyConflictActions())))
	case " ", "space":
		m.moveConflictApplyAll = !m.moveConflictApplyAll
	case "enter":
		return m.applyMoveConflictDecision()
	}
	if m.moveConflictAction == copyConflictRename && strings.TrimSpace(m.moveConflictRename) == "" {
		m.moveConflictRename = suggestedCopyName(m.movePlans[m.moveConflictIndex].target)
	}
	return m, nil
}

func (m model) applyMoveConflictDecision() (tea.Model, tea.Cmd) {
	index := m.moveConflictIndex
	action := m.moveConflictAction
	if err := applyMoveDecision(m.movePlans, index, action, m.moveConflictRename); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}
	if m.moveConflictApplyAll {
		for next := index + 1; next < len(m.movePlans); next++ {
			if !m.movePlans[next].conflict {
				continue
			}
			rename := ""
			if action == copyConflictRename {
				rename = suggestedCopyName(m.movePlans[next].target)
			}
			if err := applyMoveDecision(m.movePlans, next, action, rename); err != nil {
				m.dialogError = err.Error()
				return m, nil
			}
		}
		plans := append([]filesystemMoveDecisionPlan(nil), m.movePlans...)
		target := m.moveConflictTarget
		m.closeModal()
		return m.runFilesystemMoveDecisionPlans(plans, target)
	}
	next := nextMoveConflict(m.movePlans, index+1)
	if next < 0 {
		plans := append([]filesystemMoveDecisionPlan(nil), m.movePlans...)
		target := m.moveConflictTarget
		m.closeModal()
		return m.runFilesystemMoveDecisionPlans(plans, target)
	}
	m.moveConflictIndex = next
	m.moveConflictAction = copyConflictReplace
	m.moveConflictRename = suggestedCopyName(m.movePlans[next].target)
	m.dialogError = ""
	return m, nil
}

func applyMoveDecision(plans []filesystemMoveDecisionPlan, index int, action copyConflictAction, rename string) error {
	if index < 0 || index >= len(plans) {
		return fmt.Errorf("invalid move conflict index")
	}
	plan := &plans[index]
	switch action {
	case copyConflictReplace:
		plan.overwrite = true
	case copyConflictSkip:
		plan.skip = true
	case copyConflictRename:
		name, err := validateCopyRename(rename)
		if err != nil {
			return err
		}
		target := filepath.Join(filepath.Dir(plan.target), name)
		if filepath.Clean(target) == filepath.Clean(plan.source) {
			return fmt.Errorf("source and destination are the same: %s", plan.source)
		}
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("rename target already exists: %s", name)
		} else if !os.IsNotExist(err) {
			return err
		}
		plan.target = target
		plan.conflict = false
	default:
		return fmt.Errorf("unknown move conflict action")
	}
	return nil
}

func (m model) renderMoveConflict() string {
	plan := m.movePlans[m.moveConflictIndex]
	var body strings.Builder
	fmt.Fprintf(&body, "%s already exists.\n\n", filepath.Base(plan.target))
	fmt.Fprintf(&body, "Source:      %s\n", plan.source)
	fmt.Fprintf(&body, "Destination: %s\n\n", plan.target)
	for _, action := range copyConflictActions() {
		marker := "  "
		if action == m.moveConflictAction {
			marker = "> "
		}
		body.WriteString(marker + copyConflictActionLabel(action) + "  ")
	}
	body.WriteString("\n\n")
	if m.moveConflictAction == copyConflictRename {
		body.WriteString("New name:\n")
		body.WriteString(activeFieldStyle.Width(48).Render(m.moveConflictRename))
		body.WriteString("\n\n")
	}
	check := "[ ]"
	if m.moveConflictApplyAll {
		check = "[x]"
	}
	body.WriteString(check + " Apply to all remaining conflicts\n")
	if m.dialogError != "" {
		body.WriteString("\n" + errorStyle.Render(m.dialogError) + "\n")
	}
	body.WriteString("\n" + mutedStyle.Render("Arrows/Tab choose · Space toggles all · Enter confirms · Esc cancels"))
	return body.String()
}
