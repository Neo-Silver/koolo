package action

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/hectorgimenez/koolo/internal/utils"
)

type Ingredients struct {
	Name  string
	Runes []string
	Bases []string
}

// Mehr einfügen falls notwendig
var (
	Runeword = []Ingredients{

		{
			Name:  "Stealth",
			Runes: []string{"TalRune", "EthRune"},
			Bases: []string{"QuiltedArmor", "StuddedLeather", "HardLeatherArmor", "LeatherArmor"},
		},

		{
			Name:  "Spirit Sword",
			Runes: []string{"TalRune", "ThulRune", "OrtRune", "AmnRune"},
			Bases: []string{"CrystalSword"},
		},

		{
			Name:  "Lore",
			Runes: []string{"OrtRune", "SolRune"},
			Bases: []string{"Cap"},
		},

		{
			Name:  "Insight",
			Runes: []string{"RalRune", "TirRune", "TalRune", "SolRune"},
			Bases: []string{"Voulge", "Halberd", "Poleaxe", "Scythe"},
		},

		{
			Name:  "Rhyme",
			Runes: []string{"ShaelRune", "EthRune"},
			Bases: []string{"SmallShield", "BoneShield"},
		},
	}
)

func RwMaker() error {
	ctx := context.Get()
	ctx.SetLastAction("RwMaker")

	if !ctx.CharacterCfg.RwMaker.Enabled {
		ctx.Logger.Debug("Runewords disabled, skipping")
		return nil
	}

	// Ist ein Rezept aktiviert?
	if len(ctx.CharacterCfg.RwMaker.EnabledRws) == 0 {
		ctx.Logger.Debug("No recipes activated.")
		return nil
	}

	if !HasEnoughFreeRows(ctx.CharacterCfg.Inventory.InventoryLock) {
		ctx.Logger.Debug("Not enough free inventory rows, skipping Runeword creation")
		return nil
	}
	for {
		foundActiveRecipe := false
		// Eine Runde je Runenwort
		for _, currentRuneword := range Runeword {
			if !slices.Contains(ctx.CharacterCfg.RwMaker.EnabledRws, currentRuneword.Name) {
				continue
			}

			usedItemTracker := make(map[int]bool)
			stashItems := ctx.Data.Inventory.ByLocation(item.LocationStash, item.LocationSharedStash)
			unusedStashItems := filterUnusedItems(stashItems, usedItemTracker)
			remainingItems := make(map[int]data.Item)
			for _, item := range unusedStashItems {
				remainingItems[item.ID] = item
			}

			matchedItems, hasItems := ItemsForRunewordWithTracking(ctx, remainingItems, currentRuneword, usedItemTracker)
			if !hasItems {
				ctx.Logger.Debug("Could not find items for Runeword",
					slog.String("Runeword", currentRuneword.Name))
				continue
			}

			foundActiveRecipe = true

			if !HasEnoughFreeRows(ctx.CharacterCfg.Inventory.InventoryLock) {
				ctx.Logger.Debug("Not enough free inventory rows")
				return nil
			}

			// Benutzte Items
			for _, item := range matchedItems {
				usedItemTracker[item.ID] = true
				ctx.Logger.Debug("Preparing to use item",
					slog.String("ItemName", string(item.Name)),
					slog.Int("ItemID", item.ID))
			}

			err := ItemsToInventory(ctx, matchedItems)
			if err != nil {
				ctx.Logger.Error("Failed to move items to inventory", slog.Any("error", err))
				continue
			}

			_, err = CreateRuneword(ctx, currentRuneword, matchedItems)
			if err != nil {
				ctx.Logger.Error("Failed to create Runeword",
					slog.String("Runeword", currentRuneword.Name),
					slog.Any("error", err))
				utils.Sleep(200)
				continue
			}
			err = BackToStash(ctx)
			if err != nil {
				ctx.Logger.Error("Failed to move Runeword back to stash", slog.Any("error", err))
			}

			utils.Sleep(500)
		}

		if !foundActiveRecipe {
			break
		}
		CleanupInventory()
		utils.Sleep(200)
	}
	step.CloseAllMenus()
	return nil
}

// Verwendete Items
func filterUnusedItems(items []data.Item, usedItemTracker map[int]bool) []data.Item {
	unusedItems := []data.Item{}
	for _, item := range items {
		if !usedItemTracker[item.ID] {
			unusedItems = append(unusedItems, item)
		}
	}
	return unusedItems
}

func ItemsForRunewordWithTracking(
	ctx *context.Status,
	remainingItems map[int]data.Item,
	runeword Ingredients,
	usedItemTracker map[int]bool,
) ([]data.Item, bool) {
	requiredRunes := make(map[string]int)
	for _, rune := range runeword.Runes {
		requiredRunes[rune]++
	}

	requiredBases := make(map[string]bool)
	for _, base := range runeword.Bases {
		requiredBases[base] = true
	}
	usedItemIDs := make(map[int]bool)

	matchedItems := []data.Item{}
	//Base Suchen
	foundBaseItem := false
	for _, item := range remainingItems {
		if usedItemTracker[item.ID] {
			continue
		}

		if requiredBases[string(item.Name)] && !item.IsRuneword {
			matchedItems = append(matchedItems, item)
			usedItemIDs[item.ID] = true
			foundBaseItem = true
			break
		}
	}

	if !foundBaseItem {
		return nil, false
	}
	//Runen Suchen
	for _, requiredRune := range runeword.Runes {
		runesFound := 0
		for _, item := range remainingItems {
			if usedItemTracker[item.ID] || usedItemIDs[item.ID] {
				continue
			}

			if string(item.Name) == requiredRune {
				matchedItems = append(matchedItems, item)
				usedItemIDs[item.ID] = true
				runesFound++
				if runesFound == requiredRunes[requiredRune] {
					break
				}
			}
		}
		if runesFound < requiredRunes[requiredRune] {
			return nil, false
		}
	}

	return matchedItems, true
}

func uniqueItemKey(item data.Item) string {
	return fmt.Sprintf("%s_%d_%d", item.Name, item.Position.X, item.Position.Y)
}

func HasEnoughFreeRows(lockConfig [][]int) bool {
	const requiredCols = 4
	const totalRows = 4
	const totalCols = 10

	if len(lockConfig) < totalRows {
		return false
	}

	for startCol := 0; startCol <= totalCols-requiredCols; startCol++ {
		for row := 0; row < totalRows; row++ {
			isFree := true

			for colOffset := 0; colOffset < requiredCols; colOffset++ {
				if lockConfig[row][startCol+colOffset] != 0 {
					isFree = false
					break
				}
			}

			if isFree {
				return true
			}
		}
	}

	return false
}

func ItemsToInventory(ctx *context.Status, matchedItems []data.Item) error {
	processedKeys := make(map[string]bool)
	usedItemIDs := make(map[int]bool)

	if !ctx.Data.OpenMenus.Stash {
		err := OpenStash()
		if err != nil {
			ctx.Logger.Error("Failed to open stash", slog.Any("error", err))
			return err
		}
	}

	itemsCopy := make([]data.Item, len(matchedItems))
	copy(itemsCopy, matchedItems)

	for _, itm := range itemsCopy {
		key := uniqueItemKey(itm)
		if usedItemIDs[itm.ID] {
			ctx.Logger.Debug("Skipping already used item",
				slog.String("key", key),
				slog.Int("itemID", itm.ID))
			continue
		}

		if processedKeys[key] {
			continue
		}
		processedKeys[key] = true
		usedItemIDs[itm.ID] = true

		ctx.Logger.Debug("Attempting to move item to inventory",
			slog.String("key", key),
			slog.String("ItemName", string(itm.Name)),
			slog.Int("ItemID", itm.ID))

		if itm.Location.LocationType != item.LocationStash && itm.Location.LocationType != item.LocationSharedStash {
			continue
		}

		switch itm.Location.LocationType {
		case item.LocationStash:
			SwitchStashTab(1)
		case item.LocationSharedStash:
			SwitchStashTab(itm.Location.Page + 1)
		}

		ctx.Logger.Debug("Moving Item",
			slog.String("key", key),
			slog.Int("itemID", itm.ID))
		screenPos := ui.GetScreenCoordsForItem(itm)

		ctx.HID.ClickWithModifier(game.LeftButton, screenPos.X, screenPos.Y, game.CtrlKey)
		utils.Sleep(500)
	}

	ctx.RefreshGameData()
	utils.Sleep(200)
	return nil
}

func CreateRuneword(ctx *context.Status, runeword Ingredients, items []data.Item) (*data.Item, error) {
	itemsInInventory := ctx.Data.Inventory.ByLocation(item.LocationInventory)

	requiredRunes := make(map[string]int)
	for _, runeName := range runeword.Runes {
		requiredRunes[runeName]++
	}

	var baseItem data.Item
	availableRunes := []data.Item{}

	for _, invItem := range itemsInInventory {
		if baseItem.Name == "" && requiredRunes[string(invItem.Name)] == 0 {
			for _, base := range runeword.Bases {
				if string(invItem.Name) == base {
					baseItem = invItem
					break
				}
			}
		}

		if count, exists := requiredRunes[string(invItem.Name)]; exists && count > 0 {
			availableRunes = append(availableRunes, invItem)
			requiredRunes[string(invItem.Name)]--
		}
	}

	if baseItem.Name == "" {
		return nil, fmt.Errorf("no base item found for runeword %s", runeword.Name)
	}

	remainingRunes := 0
	for _, count := range requiredRunes {
		remainingRunes += count
	}

	if remainingRunes > 0 {
		return nil, fmt.Errorf("not enough runes for runeword %s, missing %d runes", runeword.Name, remainingRunes)
	}

	sortedRunes := []data.Item{}
	for _, runeName := range runeword.Runes {
		for i, rune := range availableRunes {
			if string(rune.Name) == runeName {
				sortedRunes = append(sortedRunes, rune)
				availableRunes = append(availableRunes[:i], availableRunes[i+1:]...)
				break
			}
		}
	}

	baseScreenPos := ui.GetScreenCoordsForItem(baseItem)
	if baseScreenPos.X == 0 && baseScreenPos.Y == 0 {
		return nil, fmt.Errorf("could not find base item screen position")
	}

	for _, currentRune := range sortedRunes {
		runeScreenPos := ui.GetScreenCoordsForItem(currentRune)
		if runeScreenPos.X == 0 && runeScreenPos.Y == 0 {
			return nil, fmt.Errorf("could not find rune screen position for %s", currentRune.Name)
		}
		ctx.HID.MovePointer(runeScreenPos.X, runeScreenPos.Y)
		utils.Sleep(200)
		ctx.HID.Click(game.LeftButton, runeScreenPos.X, runeScreenPos.Y)
		utils.Sleep(200)
		ctx.HID.MovePointer(baseScreenPos.X, baseScreenPos.Y)
		utils.Sleep(200)
		ctx.HID.Click(game.LeftButton, baseScreenPos.X, baseScreenPos.Y)
		utils.Sleep(200)
	}

	utils.Sleep(500)
	ctx.Logger.Info(fmt.Sprintf("Successfully created Runeword: %s", runeword.Name))

	return &baseItem, nil
}

func FindCreatedRuneword(ctx *context.Status) (*data.Item, bool) {
	itemsInInventory := ctx.Data.Inventory.ByLocation(item.LocationInventory)
	for _, invItem := range itemsInInventory {
		if invItem.IsRuneword {
			ctx.Logger.Info("Found Runeword in inventory",
				slog.String("Name", string(invItem.Name)),
				slog.Bool("IsRuneword", invItem.IsRuneword))
			return &invItem, true
		}
	}
	ctx.Logger.Debug("No Runeword found in inventory")
	return nil, false
}

func BackToStash(ctx *context.Status) error {
	if !ctx.Data.OpenMenus.Stash {
		err := OpenStash()
		if err != nil {
			ctx.Logger.Error("Failed to open stash", slog.Any("error", err))
			return err
		}
	}
	runewordItems := []data.Item{}
	itemsInInventory := ctx.Data.Inventory.ByLocation(item.LocationInventory)

	for _, invItem := range itemsInInventory {
		if invItem.IsRuneword {
			runewordItems = append(runewordItems, invItem)
		}
	}
	if len(runewordItems) == 0 {
		ctx.Logger.Error("No Runewords found in inventory to move back to stash")
		return fmt.Errorf("no runewords in inventory")
	}
	if len(runewordItems) > 1 {
		ctx.Logger.Warn("Multiple Runewords found in inventory",
			slog.Int("count", len(runewordItems)))
	}
	runewordItem := runewordItems[0]
	maxStashTabs := 4
	for tab := 1; tab <= maxStashTabs; tab++ {
		SwitchStashTab(tab)
		screenPos := ui.GetScreenCoordsForItem(runewordItem)
		if screenPos.X == 0 && screenPos.Y == 0 {
			ctx.Logger.Error("Could not find Runeword item position in inventory")
			continue
		}
		ctx.HID.ClickWithModifier(game.LeftButton, screenPos.X, screenPos.Y, game.ShiftKey)
		utils.Sleep(300)

		itemsInInventory = ctx.Data.Inventory.ByLocation(item.LocationInventory)
		itemStillInInventory := false
		for _, invItem := range itemsInInventory {
			if invItem.ID == runewordItem.ID {
				itemStillInInventory = true
				break
			}
		}

		if !itemStillInInventory {
			ctx.Logger.Info("Runeword item successfully moved to stash",
				slog.String("Item", string(runewordItem.Name)),
				slog.Int("StashTab", tab))
			return nil
		}
	}

	ctx.Logger.Error("Failed to move Runeword item to any stash tab",
		slog.String("Item", string(runewordItem.Name)))
	return fmt.Errorf("could not move item to stash")
}

func CleanupInventory() error {
	ctx := context.Get()
	ctx.SetLastAction("CleanupInventory")
	currentTab := 1
	if ctx.CharacterCfg.Character.StashToShared {
		currentTab = 2
	}
	SwitchStashTab(currentTab)
	inventoryItems := filterStashableItems(ctx.Data.Inventory.ByLocation(item.LocationInventory))
	stagedItems := 0
	for _, invItem := range inventoryItems {
		if ctx.CharacterCfg.Inventory.InventoryLock[invItem.Position.Y][invItem.Position.X] == 0 {
			continue
		}

		screenPos := ui.GetScreenCoordsForItem(invItem)
		if screenPos.X == 0 && screenPos.Y == 0 {
			ctx.Logger.Error("Could not find item position",
				slog.String("ItemName", string(invItem.Name)))
			continue
		}

		if stashItemAction(invItem, "", "", true) {
			stagedItems++
			ctx.RefreshGameData()

			if currentTab < 5 {
				currentTab++
				SwitchStashTab(currentTab)
			} else {
				ctx.Logger.Warn("Stash might be full",
					slog.Int("stagedItems", stagedItems))
				break
			}
		}
	}

	ctx.Logger.Info("Inventory Cleanup Complete",
		slog.Int("stagedItems", stagedItems))
	return nil
}

func filterStashableItems(items []data.Item) []data.Item {
	filteredItems := []data.Item{}

	for _, i := range items {
		if i.IsPotion() {
			continue
		}

		if i.Name == item.TomeOfTownPortal ||
			i.Name == item.TomeOfIdentify ||
			i.Name == item.Key ||
			i.Name == "WirtsLeg" {
			continue
		}

		stashIt, _, _ := shouldStashIt(i, false)
		if stashIt {
			filteredItems = append(filteredItems, i)
		}
	}

	return filteredItems
}
