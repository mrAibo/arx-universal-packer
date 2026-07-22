package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const modalCopyConflict modalKind = 300

type copyConflictAction int

const (
	copyConflictReplace copyConflictAction = iota
	copyConflictSkip
	copyConflictRename
)

type filesystemCopyPlan struct {
	source    string
	target    string
	overwrite bool
	skip      bool
	conflict  bool
}

func (m model) startFilesystemCopy(entries []fileEntry, destination string) (tea.Model, tea.Cmd) {
	plans, err := filesystemCopyPlans(entries, destination)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	first := nextCopyConflict(plans, 0)
	if first < 0 {
		return m.runFilesystemCopyPlans(plans)
	}
	m.modal = modalCopyConflict
	m.modalTitle = "Copy conflict"
	m.copyPlans = plans
	m.copyConflictIndex = first
	m.copyConflictAction = copyConflictReplace
	m.copyConflictApplyAll = false
	m.copyConflictRename = suggestedCopyName(plans[first].target)
	return m, nil
}

func (m model) runFilesystemCopyPlans(plans []filesystemCopyPlan) (tea.Model, tea.Cmd) {
	items := append([]filesystemCopyPlan(nil), plans...)
	return m.startOperation(fmt.Sprintf("Copying %d selected item(s)...", len(items)), func() Result {
		return copyFilesystemPlans(items)
	})
}

func filesystemCopyPlans(entries []fileEntry, destination string) ([]filesystemCopyPlan, error) {
	if _, err := filesystemCopyConflicts(entries, destination); err != nil {
		return nil, err
	}
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return nil, err
	}
	plans := make([]filesystemCopyPlan, 0, len(entries))
	for _, entry := range entries {
		source, err := filepath.Abs(entry.Path)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(destinationAbs, filepath.Base(source))
		_, err = os.Lstat(target)
		plans = append(plans, filesystemCopyPlan{
			source:   source,
			target:   target,
			conflict: err == nil,
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return plans, nil
}

func copyFilesystemPlans(plans []filesystemCopyPlan) Result {
	copied := 0
	skipped := 0
	for _, plan := range plans {
		if plan.skip {
			skipped++
			continue
		}
		if err := copyFilesystemPath(plan.source, plan.target, plan.overwrite); err != nil {
			return Result{Err: fmt.Errorf("copy %s: %w (%d copied, %d skipped)", plan.source, err, copied, skipped)}
		}
		copied++
	}
	return Result{Output: fmt.Sprintf("Copied %d item(s), skipped %d", copied, skipped)}
}

func nextCopyConflict(plans []filesystemCopyPlan, start int) int {
	for index := start; index < len(plans); index++ {
		if plans[index].conflict && !plans[index].skip && !plans[index].overwrite {
			return index
		}
	}
	return -1
}

func copyConflictActions() []copyConflictAction {
	return []copyConflictAction{copyConflictReplace, copyConflictSkip, copyConflictRename}
}

func copyConflictActionLabel(action copyConflictAction) string {
	switch action {
	case copyConflictReplace:
		return "Replace"
	case copyConflictSkip:
		return "Skip"
	case copyConflictRename:
		return "Rename"
	default:
		return "Unknown"
	}
}

func (m model) updateCopyConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.copyConflictIndex < 0 || m.copyConflictIndex >= len(m.copyPlans) {
		m.closeModal()
		return m, nil
	}
	key := msg.String()
	if m.copyConflictAction == copyConflictRename {
		switch key {
		case "backspace":
			runes := []rune(m.copyConflictRename)
			if len(runes) > 0 {
				m.copyConflictRename = string(runes[:len(runes)-1])
			}
			return m, nil
		case "ctrl+u":
			m.copyConflictRename = ""
			return m, nil
		}
		if msg.Type == tea.KeyRunes {
			m.copyConflictRename += string(msg.Runes)
			return m, nil
		}
	}

	switch key {
	case "esc", "q":
		m.closeModal()
		return m, nil
	case "left", "up", "shift+tab":
		m.copyConflictAction = copyConflictAction(wrapIndex(int(m.copyConflictAction)-1, len(copyConflictActions())))
	case "right", "down", "tab":
		m.copyConflictAction = copyConflictAction(wrapIndex(int(m.copyConflictAction)+1, len(copyConflictActions())))
	case " ", "space":
		m.copyConflictApplyAll = !m.copyConflictApplyAll
	case "enter":
		return m.applyCopyConflictDecision()
	}
	if m.copyConflictAction == copyConflictRename && strings.TrimSpace(m.copyConflictRename) == "" {
		m.copyConflictRename = suggestedCopyName(m.copyPlans[m.copyConflictIndex].target)
	}
	return m, nil
}

func (m model) applyCopyConflictDecision() (tea.Model, tea.Cmd) {
	index := m.copyConflictIndex
	action := m.copyConflictAction
	if err := applyCopyDecision(m.copyPlans, index, action, m.copyConflictRename); err != nil {
		m.dialogError = err.Error()
		return m, nil
	}
	if m.copyConflictApplyAll {
		for next := index + 1; next < len(m.copyPlans); next++ {
			if !m.copyPlans[next].conflict {
				continue
			}
			rename := ""
			if action == copyConflictRename {
				rename = suggestedCopyName(m.copyPlans[next].target)
			}
			if err := applyCopyDecision(m.copyPlans, next, action, rename); err != nil {
				m.dialogError = err.Error()
				return m, nil
			}
		}
		plans := append([]filesystemCopyPlan(nil), m.copyPlans...)
		m.closeModal()
		return m.runFilesystemCopyPlans(plans)
	}
	next := nextCopyConflict(m.copyPlans, index+1)
	if next < 0 {
		plans := append([]filesystemCopyPlan(nil), m.copyPlans...)
		m.closeModal()
		return m.runFilesystemCopyPlans(plans)
	}
	m.copyConflictIndex = next
	m.copyConflictAction = copyConflictReplace
	m.copyConflictRename = suggestedCopyName(m.copyPlans[next].target)
	m.dialogError = ""
	return m, nil
}

func applyCopyDecision(plans []filesystemCopyPlan, index int, action copyConflictAction, rename string) error {
	if index < 0 || index >= len(plans) {
		return fmt.Errorf("invalid copy conflict index")
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
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("rename target already exists: %s", name)
		} else if !os.IsNotExist(err) {
			return err
		}
		plan.target = target
		plan.conflict = false
	default:
		return fmt.Errorf("unknown copy conflict action")
	}
	return nil
}

func validateCopyRename(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return "", fmt.Errorf("enter a valid new name")
	}
	if strings.ContainsRune(value, 0) || strings.ContainsAny(value, `/\\`) {
		return "", fmt.Errorf("new name must not contain path separators")
	}
	return value, nil
}

func suggestedCopyName(target string) string {
	directory := filepath.Dir(target)
	name := filepath.Base(target)
	extension := filepath.Ext(name)
	stem := strings.TrimSuffix(name, extension)
	for number := 1; ; number++ {
		candidate := fmt.Sprintf("%s (copy %d)%s", stem, number, extension)
		if _, err := os.Lstat(filepath.Join(directory, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

func (m model) renderCopyConflict() string {
	plan := m.copyPlans[m.copyConflictIndex]
	var body strings.Builder
	fmt.Fprintf(&body, "%s already exists.\n\n", filepath.Base(plan.target))
	fmt.Fprintf(&body, "Source:      %s\n", plan.source)
	fmt.Fprintf(&body, "Destination: %s\n\n", plan.target)
	for _, action := range copyConflictActions() {
		marker := "  "
		if action == m.copyConflictAction {
			marker = "> "
		}
		body.WriteString(marker + copyConflictActionLabel(action) + "  ")
	}
	body.WriteString("\n\n")
	if m.copyConflictAction == copyConflictRename {
		body.WriteString("New name:\n")
		body.WriteString(activeFieldStyle.Width(48).Render(m.copyConflictRename))
		body.WriteString("\n\n")
	}
	check := "[ ]"
	if m.copyConflictApplyAll {
		check = "[x]"
	}
	body.WriteString(check + " Apply to all remaining conflicts\n")
	if m.dialogError != "" {
		body.WriteString("\n" + errorStyle.Render(m.dialogError) + "\n")
	}
	body.WriteString("\n" + mutedStyle.Render("Arrows/Tab choose · Space toggles all · Enter confirms · Esc cancels"))
	return body.String()
}
