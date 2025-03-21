package character

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/skill"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/game"
)

type Goldie struct {
	BaseCharacter
}

func (s *Goldie) ItemPickup() {
	action.ItemPickup(40)
}

func (s *Goldie) CheckKeyBindings() []skill.ID {
	requireKeybindings := []skill.ID{}
	missingKeybindings := []skill.ID{}

	for _, cskill := range requireKeybindings {
		if _, found := s.Data.KeyBindings.KeyBindingForSkill(cskill); !found {
			missingKeybindings = append(missingKeybindings, cskill)
		}
	}

	if len(missingKeybindings) > 0 {
		s.Logger.Debug("There are missing required key bindings.", slog.Any("Bindings", missingKeybindings))
	}

	return missingKeybindings
}

func (s *Goldie) BuffSkills() []skill.ID {
	skillsList := make([]skill.ID, 0)
	if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.BattleCommand); found {
		skillsList = append(skillsList, skill.BattleCommand)
	}
	if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.Shout); found {
		skillsList = append(skillsList, skill.Shout)
	}
	if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.BattleOrders); found {
		skillsList = append(skillsList, skill.BattleOrders)
	}
	return skillsList
}

func (s Goldie) PreCTABuffSkills() []skill.ID {
	return []skill.ID{}
}

func (s Goldie) KillMonsterSequence(
	monsterSelector func(d game.Data) (data.UnitID, bool),
	skipOnImmunities []stat.Resist,
) error {
	completedAttackLoops := 0
	previousUnitID := 0

	for {
		id, found := monsterSelector(*s.Data)
		if !found {
			return nil
		}
		if previousUnitID != int(id) {
			completedAttackLoops = 0
		}

		if !s.preBattleChecks(id, skipOnImmunities) {
			return nil
		}

		if completedAttackLoops >= paladinLevelingMaxAttacksLoop {
			return nil
		}

		monster, found := s.Data.Monsters.FindByID(id)
		if !found {
			s.Logger.Info("Monster not found", slog.String("monster", fmt.Sprintf("%v", monster)))
			return nil
		}

		numOfAttacks := 5

		if s.Data.PlayerUnit.Skills[skill.BlessedHammer].Level > 0 {
			s.Logger.Debug("Using Blessed Hammer")
			// Add a random movement, maybe hammer is not hitting the target
			if previousUnitID == int(id) {
				if monster.Stats[stat.Life] > 0 {
					s.PathFinder.RandomMovement()
				}
				return nil
			}
			step.PrimaryAttack(id, numOfAttacks, false, step.Distance(2, 7), step.EnsureAura(skill.Concentration))

		} else {
			if s.Data.PlayerUnit.Skills[skill.Zeal].Level > 0 {
				s.Logger.Debug("Using Zeal")
				numOfAttacks = 1
			}
			s.Logger.Debug("Using primary attack with Holy Fire aura")
			step.PrimaryAttack(id, numOfAttacks, false, step.Distance(1, 3), step.EnsureAura(skill.HolyFire))
		}

		completedAttackLoops++
		previousUnitID = int(id)
	}
}

func (s Goldie) killMonster(_ npc.ID) error {
	return nil
}

func (s *Goldie) KillCountess() error {
	return s.killMonster(npc.DarkStalker)
}

func (s Goldie) KillAndariel() error {
	return s.killMonster(npc.Andariel)
}

func (s Goldie) KillSummoner() error {
	return s.killMonster(npc.Summoner)
}

func (s Goldie) KillDuriel() error {
	return s.killMonster(npc.Duriel)
}

func (s Goldie) KillCouncil() error {
	return s.killMonster(npc.CouncilMember)
}

func (s Goldie) KillMephisto() error {
	return s.killMonster(npc.Mephisto)
}

func (s Goldie) KillIzual() error {
	return s.killMonster(npc.Izual)
}

func (s Goldie) KillDiablo() error {
	timeout := time.Second * 20
	startTime := time.Now()
	diabloFound := false

	for {
		if time.Since(startTime) > timeout && !diabloFound {
			s.Logger.Error("Diablo was not found, timeout reached")
			return nil
		}

		diablo, found := s.Data.Monsters.FindOne(npc.Diablo, data.MonsterTypeUnique)
		if !found || diablo.Stats[stat.Life] <= 0 {
			if diabloFound {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		diabloFound = true
		s.Logger.Info("Diablo detected, attacking")

		return s.killMonster(npc.Diablo)
	}
}

func (s Goldie) KillPindle() error {
	return s.killMonster(npc.DefiledWarrior)
}

func (s Goldie) KillNihlathak() error {
	return s.killMonster(npc.Nihlathak)
}

func (s Goldie) KillBaal() error {
	return s.killMonster(npc.BaalCrab)
}
