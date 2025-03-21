package run

import (
	"log/slog"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/town"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/lxn/win"
)

type GoldGambling struct {
	ctx            *context.Status
	lastBoughtItem string
}

func NewGoldGambling() *GoldGambling {
	return &GoldGambling{
		ctx: context.Get(),
	}
}

const (
	sellButtonX  = 480
	sellButtonY  = 560
	weaponSlotX  = 700
	weaponSlotY  = 195
	confirmSellX = 700
	confirmSellY = 195
)

func (a GoldGambling) Name() string {
	return string(config.GoldGamblingRun)
}

func (a GoldGambling) Run() error {
	time.Sleep(time.Millisecond * 3000)
	err := action.WayPoint(area.ThePandemoniumFortress)
	if err != nil {
		return err
	}
	time.Sleep(time.Millisecond * 100)
	a.ctx.HID.PressKey('N')
	time.Sleep(time.Millisecond * 100)
	a.Slot1()
	time.Sleep(time.Millisecond * 100)

	a.Vendor()
	time.Sleep(time.Millisecond * 100)
	a.ctx.RefreshGameData()
	maxSellAttempts := 3
	sellSuccess := false
	for i := 0; i < maxSellAttempts && !sellSuccess; i++ {
		err := a.sellEquippedWeapon()
		if err != nil {
			a.ctx.Logger.Info("Verkauf fehlgeschlagen, versuche erneut", slog.Int("versuch", i+1))
			time.Sleep(time.Millisecond * 100)
			continue
		}

		time.Sleep(time.Millisecond * 200)
		if !a.hasEquippedWeapon() {
			sellSuccess = true
			a.ctx.Logger.Info("Waffe erfolgreich verkauft")
		} else {
			a.ctx.Logger.Info("Waffe wurde nicht verkauft, versuche erneut", slog.Int("versuch", i+1))
			time.Sleep(time.Millisecond * 100)
		}
	}
	time.Sleep(time.Millisecond * 100)

	for {
		shopItems := a.FindItems()
		if len(shopItems.Items) > 0 {
			cycleCount := 0
			maxCycles := 50000
			startTime := time.Now()
			for cycleCount < maxCycles {
				currentGold := a.ctx.Data.Inventory.Gold
				if currentGold >= 870000 {
					step.CloseAllMenus()
					time.Sleep(time.Millisecond * 100)
					action.VendorRefill(false, true)
					time.Sleep(time.Millisecond * 300)
					a.Slot2()
					time.Sleep(time.Millisecond * 100)
					action.Gamble()
					time.Sleep(time.Millisecond * 300)
					a.Slot1()
					time.Sleep(time.Millisecond * 200)
					action.Stash(false)
					time.Sleep(time.Millisecond * 200)
					step.CloseAllMenus()
					time.Sleep(time.Millisecond * 200)
					startTime = time.Now()
					a.Vendor()
				}
				a.ctx.RefreshGameData()
				time.Sleep(time.Millisecond * 200)
				maxBuyAttempts := 3
				buySuccess := false
				for i := 0; i < maxBuyAttempts && !buySuccess; i++ {
					err := a.BuyOne(shopItems)
					if err != nil {
						a.ctx.Logger.Info("Kauf fehlgeschlagen, versuche erneut", slog.Int("versuch", i+1))
						time.Sleep(time.Millisecond * 200)
						continue
					}
					time.Sleep(time.Millisecond * 200)
					if a.hasEquippedWeapon() {
						buySuccess = true
						a.ctx.Logger.Info("Waffe erfolgreich gekauft")
					} else {
						a.ctx.Logger.Info("Waffe wurde nicht gekauft, versuche erneut", slog.Int("versuch", i+1))
						time.Sleep(time.Millisecond * 200)
					}
				}

				if !buySuccess {
					a.ctx.Logger.Info("Konnte keine Waffe kaufen, überprüfe Shop-Angebot")
					shopItems = a.FindItems()
					if len(shopItems.Items) == 0 {
						a.ctx.Logger.Info("Keine passenden Items im Shop, verlasse Stadt...")
						break
					}
					continue
				}
				time.Sleep(time.Millisecond * 200)

				a.ctx.RefreshGameData()
				sellSuccess = false
				for i := 0; i < maxSellAttempts && !sellSuccess; i++ {
					err := a.sellEquippedWeapon()
					if err != nil {
						a.ctx.Logger.Info("Verkauf fehlgeschlagen, versuche erneut", slog.Int("versuch", i+1))
						time.Sleep(time.Millisecond * 200)
						continue
					}
					time.Sleep(time.Millisecond * 200)
					a.ctx.RefreshGameData()
					if !a.hasEquippedWeapon() {
						sellSuccess = true
						a.ctx.Logger.Info("Waffe erfolgreich verkauft")
					} else {
						a.ctx.Logger.Info("Waffe wurde nicht verkauft, versuche erneut", slog.Int("versuch", i+1))
						time.Sleep(time.Millisecond * 200)
					}
				}

				time.Sleep(time.Millisecond * 200)

				cycleCount++
				a.ctx.Logger.Info("Completed cycle",
					slog.Int("cycleNumber", cycleCount),
					slog.Int("remainingCycles", maxCycles-cycleCount),
					slog.Duration("elapsedTime", time.Since(startTime)))
			}
		} else {
			err := a.OutofTown()
			if err != nil {
				return err
			}
		}
	}
}

func (a *GoldGambling) hasEquippedWeapon() bool {
	items := a.ctx.Data.Inventory.ByLocation(item.LocationEquipped)
	for _, it := range items {
		if it.Location.BodyLocation == item.LocLeftArm {
			return true
		}
	}
	return false
}

type ShopItemResult struct {
	Items       []data.Item
	ItemsByPage map[int][]data.Item
}

func (a *GoldGambling) FindItems() *ShopItemResult {
	result := &ShopItemResult{
		Items:       []data.Item{},
		ItemsByPage: make(map[int][]data.Item),
	}

	items := a.ctx.Data.Inventory.ByLocation(item.LocationVendor)
	for _, it := range items {
		if it.Location.LocationType != item.LocationVendor {
			continue
		}
		hasDesiredStat := false
		hasUndesiredStat := false
		for _, stat := range it.Stats {
			if stat.ID == 218 && stat.Value == 1 {
				hasDesiredStat = true
			}
			if stat.ID == 39 || stat.ID == 41 || stat.ID == 43 || stat.ID == 45 ||
				stat.ID == 55 || stat.ID == 57 || stat.ID == 58 || stat.ID == 59 ||
				stat.ID == 62 || stat.ID == 60 || stat.ID == 74 || stat.ID == 93 ||
				stat.ID == 117 || stat.ID == 195 || stat.ID == 196 || stat.ID == 197 ||
				stat.ID == 198 || stat.ID == 199 || stat.ID == 201 || stat.ID == 204 ||
				stat.ID == 224 || stat.ID == 226 || stat.ID == 227 || stat.ID == 228 {
				hasUndesiredStat = true
				break
			}
		}

		if hasDesiredStat && !hasUndesiredStat && isIncludedItemName(it.Name) {
			result.Items = append(result.Items, it)
			result.ItemsByPage[it.Location.Page] = append(result.ItemsByPage[it.Location.Page], it)
			a.ctx.Logger.Info("Found matching item in shop",
				slog.String("name", string(it.Name)),
				slog.Int("page", it.Location.Page))
		}
	}

	a.ctx.Logger.Info("Shop search completed",
		slog.Int("totalItems", len(result.Items)),
		slog.Int("totalPages", len(result.ItemsByPage)))

	return result
}

func (a *GoldGambling) BuyOne(shopItems *ShopItemResult) error {
	if shopItems == nil || len(shopItems.Items) == 0 {
		a.ctx.Logger.Info("No items found in shop, skipping purchase")
		return nil
	}

	itemToBuy := shopItems.Items[0]
	itemPage := itemToBuy.Location.Page

	currentPage := a.ctx.Data.Inventory.ByLocation(item.LocationVendor)[0].Location.Page
	if itemPage != currentPage {
		var tabBtnX, tabBtnY float64
		switch itemPage {
		case 0:
			tabBtnX = ui.ShopTabBtnXClassic0
			tabBtnY = ui.ShopTabBtnYClassic0
		case 1:
			tabBtnX = ui.ShopTabBtnXClassic1
			tabBtnY = ui.ShopTabBtnYClassic1
		case 2:
			tabBtnX = ui.ShopTabBtnXClassic2
			tabBtnY = ui.ShopTabBtnYClassic2
		case 3:
			tabBtnX = ui.ShopTabBtnXClassic3
			tabBtnY = ui.ShopTabBtnYClassic3
		}
		a.ctx.HID.Click(game.LeftButton, int(tabBtnX), int(tabBtnY))
		time.Sleep(time.Millisecond * 400)
	}
	a.ctx.Logger.Info("Buying single item from shop",
		slog.String("name", string(itemToBuy.Name)),
		slog.Int("page", itemToBuy.Location.Page))

	town.BuyItem(itemToBuy, 1)
	time.Sleep(time.Millisecond * 100)
	a.lastBoughtItem = string(itemToBuy.Name)

	a.ctx.Logger.Info("Successfully purchased item",
		slog.String("name", string(itemToBuy.Name)))

	return nil
}

func (a *GoldGambling) OutofTown() error {
	step.CloseAllMenus()
	time.Sleep(time.Millisecond * 200)
	err := action.WayPoint(area.RiverOfFlame)
	if err != nil {
		return err
	}
	a.ctx.RefreshGameData()
	time.Sleep(time.Millisecond * 500)
	err = action.WayPoint(area.ThePandemoniumFortress)
	if err != nil {
		return err
	}
	a.ctx.RefreshGameData()
	time.Sleep(time.Millisecond * 500)
	a.Vendor()
	time.Sleep(time.Millisecond * 500)
	return nil
}

func (a *GoldGambling) Slot1() {
	ctx := context.Get()
	ctx.RefreshGameData()
	time.Sleep(100 * time.Millisecond)

	ctx.Logger.Info("Current weapon slot", slog.Int("ID", ctx.Data.ActiveWeaponSlot))

	if ctx.Data.ActiveWeaponSlot == 1 {
		ctx.Logger.Info("Switching to weapon slot 1")
		a.ctx.HID.PressKey('W')
		time.Sleep(200 * time.Millisecond)
	}
}

func (a *GoldGambling) Slot2() {
	ctx := context.Get()
	ctx.RefreshGameData()
	time.Sleep(100 * time.Millisecond)

	ctx.Logger.Info("Current weapon slot", slog.Int("ID", ctx.Data.ActiveWeaponSlot))

	if ctx.Data.ActiveWeaponSlot == 1 {
		ctx.Logger.Info("Already in weapon slot 2")
		time.Sleep(200 * time.Millisecond)
	} else {
		ctx.Logger.Info("Switching to weapon slot 2")
		a.ctx.HID.PressKey('W')
	}
}

func isIncludedItemName(name item.Name) bool {
	excludedNames := []string{
		"Cinquedeas",
		"Zweihander",
		"WarClub",
		"GothicSword",
		"AncientSword",
		"Naga",
		"AncientAxe",
		"Yari",
		"Partizan",
		"Lance",
		"Stiletto",
	}

	for _, excluded := range excludedNames {
		if string(name) == excluded {
			return true
		}
	}
	return false
}

func (a *GoldGambling) sellEquippedWeapon() error {
	items := a.ctx.Data.Inventory.ByLocation(item.LocationEquipped)
	for _, it := range items {
		if it.Location.BodyLocation == item.LocLeftArm {
			a.ctx.HID.Click(game.LeftButton, sellButtonX, sellButtonY)
			time.Sleep(time.Millisecond * 120)
			a.ctx.HID.Click(game.LeftButton, weaponSlotX, weaponSlotY)
			time.Sleep(time.Millisecond * 120)
			a.ctx.HID.Click(game.LeftButton, confirmSellX, confirmSellY)
			time.Sleep(time.Millisecond * 120)
			return nil
		}
	}
	return nil
}

func (a *GoldGambling) Vendor() error {
	RepairNPC := town.GetTownByArea(a.ctx.Data.PlayerUnit.Area).RepairNPC()
	err := action.InteractNPC(RepairNPC)
	if err != nil {
		return err
	}
	if RepairNPC != npc.Halbu {
		a.ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_RETURN)
	} else {
		a.ctx.HID.KeySequence(win.VK_HOME, win.VK_RETURN)
	}
	return nil
}
