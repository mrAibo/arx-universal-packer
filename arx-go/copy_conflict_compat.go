package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) runFilesystemCopy(entries []fileEntry, destination string, overwrite bool) (tea.Model, tea.Cmd) {
	plans, err := filesystemCopyPlans(entries, destination)
	if err != nil {
		m.showError(err)
		return m, nil
	}
	if overwrite {
		for index := range plans {
			if plans[index].conflict {
				plans[index].overwrite = true
			}
		}
	} else {
		for _, plan := range plans {
			if plan.conflict {
				m.showError(fmt.Errorf("destination already contains: %s", plan.target))
				return m, nil
			}
		}
	}
	return m.runFilesystemCopyPlans(plans)
}
