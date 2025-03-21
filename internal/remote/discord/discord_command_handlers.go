package discord

import (
	"slices"
)

func (b *Bot) supervisorExists(supervisor string) bool {
	supervisors := b.manager.AvailableSupervisors()
	return slices.Contains(supervisors, supervisor)
}
