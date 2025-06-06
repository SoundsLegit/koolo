package action

import (
	"fmt"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/d2go/pkg/nip"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/town"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/hectorgimenez/koolo/internal/utils"
	"github.com/lxn/win"
)

func IdentifyAll(skipIdentify bool) error {
	ctx := context.Get()
	ctx.SetLastAction("IdentifyAll")

	items := itemsToIdentify()

	ctx.Logger.Debug("Checking for items to identify...")
	if len(items) == 0 || skipIdentify {
		ctx.Logger.Debug("No items to identify...")
		return nil
	}

	if ctx.CharacterCfg.Game.UseCainIdentify {
		ctx.Logger.Debug("Identifying all item with Cain...")
		// Close any open menus first
		step.CloseAllMenus()
		utils.Sleep(500)

		err := CainIdentify()
		// if identifying with cain fails then we should continue to identify using tome
		if err == nil {
			return nil
		}
		ctx.Logger.Debug("Identifying with Cain failed, continuing with identifying with tome", "err", err)
	}

	idTome, found := ctx.Data.Inventory.Find(item.TomeOfIdentify, item.LocationInventory)
	if !found {
		ctx.Logger.Warn("ID Tome not found, not identifying items")
		return nil
	}

	if st, statFound := idTome.FindStat(stat.Quantity, 0); !statFound || st.Value < len(items) {
		ctx.Logger.Info("Not enough ID scrolls, refilling...")
		VendorRefill(true, false)
	}

	ctx.Logger.Info(fmt.Sprintf("Identifying %d items...", len(items)))

	// Close all menus to prevent issues
	step.CloseAllMenus()
	for !ctx.Data.OpenMenus.Inventory {
		ctx.HID.PressKeyBinding(ctx.Data.KeyBindings.Inventory)
		utils.Sleep(1000) // Add small delay to allow the game to open the inventory
	}

	for _, i := range items {
		identifyItem(idTome, i)
	}
	step.CloseAllMenus()

	return nil
}

func CainIdentify() error {
	ctx := context.Get()
	ctx.SetLastAction("CainIdentify")

	stayAwhileAndListen := town.GetTownByArea(ctx.Data.PlayerUnit.Area).IdentifyNPC()

	// Close any open menus first
	step.CloseAllMenus()
	utils.Sleep(200)

	err := InteractNPC(stayAwhileAndListen)
	if err != nil {
		return fmt.Errorf("error interacting with Cain: %w", err)
	}

	// Verify menu opened
	menuWait := time.Now().Add(2 * time.Second)
	for time.Now().Before(menuWait) {
		ctx.PauseIfNotPriority()
		ctx.RefreshGameData()
		if ctx.Data.OpenMenus.NPCInteract {
			break
		}
		utils.Sleep(100)
	}

	if !ctx.Data.OpenMenus.NPCInteract {
		return fmt.Errorf("NPC menu did not open")
	}

	// Select identify option
	ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_RETURN)
	utils.Sleep(800)

	// Close menu if still open
	if ctx.Data.OpenMenus.NPCInteract {
		step.CloseAllMenus()
	}

	return nil
}

func itemsToIdentify() (items []data.Item) {
	ctx := context.Get()
	ctx.SetLastAction("itemsToIdentify")

	for _, i := range ctx.Data.Inventory.ByLocation(item.LocationInventory) {
		if i.Identified || i.Quality == item.QualityNormal || i.Quality == item.QualitySuperior {
			continue
		}

		// Skip identifying items that fully match a rule when unid
		if _, result := ctx.CharacterCfg.Runtime.Rules.EvaluateAll(i); result == nip.RuleResultFullMatch {
			continue
		}

		items = append(items, i)
	}

	return
}

func HaveItemsToStashUnidentified() bool {
	ctx := context.Get()
	ctx.SetLastAction("HaveItemsToStashUnidentified")

	items := ctx.Data.Inventory.ByLocation(item.LocationInventory)
	for _, i := range items {
		if !i.Identified {
			if _, result := ctx.CharacterCfg.Runtime.Rules.EvaluateAll(i); result == nip.RuleResultFullMatch {
				return true
			}
		}
	}

	return false
}

func identifyItem(idTome data.Item, i data.Item) {
	ctx := context.Get()
	screenPos := ui.GetScreenCoordsForItem(idTome)

	utils.Sleep(500)
	ctx.HID.Click(game.RightButton, screenPos.X, screenPos.Y)
	utils.Sleep(1000)

	screenPos = ui.GetScreenCoordsForItem(i)

	ctx.HID.Click(game.LeftButton, screenPos.X, screenPos.Y)
	utils.Sleep(350)
}
