package hlims

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type inventoryAssetView struct {
	PublicID            string
	Name                string
	Placement           string
	Notes               string
	MachineName         string
	MachinePublicID     string
	PurchasePublicID    string
	ParentAssetPublicID string
	ParentAssetName     string
	Status              string
	IsSpare             bool
}

type inventoryProductView struct {
	PublicID      string
	Name          string
	Manufacturer  string
	Specification string
	Notes         string
	LinkURL       string
	LinkLabel     string
	Assets        []inventoryAssetView
}

type inventoryGroupView struct {
	Name     string
	Count    int
	Products []inventoryProductView
}

type inventoryTab struct {
	Name   string
	URL    string
	Active bool
}

type inventoryPage struct {
	View   string
	Filter string
	Tabs   []inventoryTab
	Groups []inventoryGroupView
	Detail *inventoryDetailView
}

type inventorySnapshot struct {
	products []database.ListProductDetailsRow
	assets   []database.ListAssetDetailsRow
	machines []database.ListMachineDetailsRow
	areas    []database.ListAreasRow
	makers   []database.Manufacturer
	links    []database.ProductLink
	ports    []database.ProductPortProfile
	bands    []database.WifiBandSpec
}

func loadInventory(ctx context.Context, queries *database.Queries) (inventorySnapshot, error) {
	var result inventorySnapshot
	var err error
	if result.products, err = queries.ListProductDetails(ctx); err != nil {
		return result, fmt.Errorf("list products: %w", err)
	}
	if result.assets, err = queries.ListAssetDetails(ctx); err != nil {
		return result, fmt.Errorf("list assets: %w", err)
	}
	if result.machines, err = queries.ListMachineDetails(ctx); err != nil {
		return result, fmt.Errorf("list machines: %w", err)
	}
	if result.areas, err = queries.ListAreas(ctx); err != nil {
		return result, fmt.Errorf("list areas: %w", err)
	}
	if result.makers, err = queries.ListManufacturers(ctx); err != nil {
		return result, fmt.Errorf("list manufacturers: %w", err)
	}
	if result.links, err = queries.ListProductLinks(ctx); err != nil {
		return result, fmt.Errorf("list product links: %w", err)
	}
	if result.ports, err = queries.ListProductPortProfiles(ctx); err != nil {
		return result, fmt.Errorf("list port profiles: %w", err)
	}
	if result.bands, err = queries.ListWiFiBands(ctx); err != nil {
		return result, fmt.Errorf("list Wi-Fi bands: %w", err)
	}
	return result, nil
}

type inventoryIndex struct {
	assetsByID      map[string]database.ListAssetDetailsRow
	productsByID    map[string]database.ListProductDetailsRow
	machinesByAsset map[string]database.ListMachineDetailsRow
	areasByID       map[string]string
}

func newInventoryIndex(snapshot inventorySnapshot) inventoryIndex {
	index := inventoryIndex{
		assetsByID:      make(map[string]database.ListAssetDetailsRow, len(snapshot.assets)),
		productsByID:    make(map[string]database.ListProductDetailsRow, len(snapshot.products)),
		machinesByAsset: make(map[string]database.ListMachineDetailsRow),
		areasByID:       make(map[string]string, len(snapshot.areas)),
	}
	for _, asset := range snapshot.assets {
		index.assetsByID[asset.ID] = asset
	}
	for _, product := range snapshot.products {
		index.productsByID[product.ID] = product
	}
	for _, machine := range snapshot.machines {
		if machine.AssetID.Valid {
			index.machinesByAsset[machine.AssetID.String] = machine
		}
	}
	for _, area := range snapshot.areas {
		index.areasByID[area.ID] = area.Name
	}
	return index
}

func (index inventoryIndex) machineFor(asset database.ListAssetDetailsRow) (database.ListMachineDetailsRow, bool) {
	for seen := 0; seen <= len(index.assetsByID); seen++ {
		if machine, ok := index.machinesByAsset[asset.ID]; ok {
			return machine, true
		}
		if !asset.ParentAssetID.Valid {
			break
		}
		parent, ok := index.assetsByID[asset.ParentAssetID.String]
		if !ok {
			break
		}
		asset = parent
	}
	return database.ListMachineDetailsRow{}, false
}

func (index inventoryIndex) placement(asset database.ListAssetDetailsRow) string {
	if asset.ParentAssetID.Valid {
		parent := index.assetsByID[asset.ParentAssetID.String]
		name := parent.Name.String
		if !parent.Name.Valid {
			name = index.productsByID[parent.ProductID].Name
		}
		if asset.ParentSlot.Valid {
			return name + " / " + asset.ParentSlot.String
		}
		return "Inside " + name
	}
	if asset.AreaID.Valid {
		return index.areasByID[asset.AreaID.String]
	}
	return "Unplaced"
}

func inventoryCategory(kind string) string {
	switch kind {
	case "system":
		return "Systems"
	case "memory":
		return "Memory"
	case "processor":
		return "Processors"
	case "drive":
		return "Drives"
	case "router", "switch", "access_point", "network_adapter":
		return "Network equipment"
	case "rack":
		return "Racks"
	default:
		return "Other"
	}
}

func inventorySpec(product database.ListProductDetailsRow) string {
	switch product.Kind {
	case "memory":
		parts := []string{fmt.Sprintf("%d GiB", product.MemoryCapacityBytes.Int64/(1024*1024*1024)), product.MemoryType.String}
		if product.SpeedMts.Valid {
			parts = append(parts, fmt.Sprintf("%d MT/s", product.SpeedMts.Int64))
		}
		return strings.Join(parts, " · ")
	case "processor":
		parts := []string{fmt.Sprintf("%d cores / %d threads", product.CoreCount.Int64, product.ThreadCount.Int64)}
		if product.Generation.Valid {
			parts = append(parts, product.Generation.String)
		}
		return strings.Join(parts, " · ")
	case "rack":
		return fmt.Sprintf("%dU · %s", product.RackUnits.Int64, product.MountingStandard.String)
	case "drive":
		return fmt.Sprintf("%s · %s", consoleByteSize(&product.DriveCapacityBytes.Int64), product.InterfaceKind.String)
	case "router", "access_point":
		if product.WifiGeneration.Valid {
			return fmt.Sprintf("Wi-Fi %d · %s", product.WifiGeneration.Int64, product.WifiClass.String)
		}
	}
	return ""
}

func inventoryFilterItems() []inventoryTab {
	return []inventoryTab{
		{Name: "All assets", URL: "/console/inventory/"},
		{Name: "Memory", URL: "/console/inventory/?filter=memory"},
		{Name: "Uninstalled parts", URL: "/console/inventory/?filter=uninstalled"},
		{Name: "Systems", URL: "/console/inventory/?filter=system"},
		{Name: "Network", URL: "/console/inventory/?filter=network"},
		{Name: "Racks", URL: "/console/inventory/?filter=rack"},
	}
}

func filterInventory(asset inventoryAssetView, category, filter string) bool {
	switch filter {
	case "memory":
		return category == "Memory"
	case "uninstalled":
		return asset.IsSpare
	case "system":
		return category == "Systems"
	case "network":
		return category == "Network equipment"
	case "rack":
		return category == "Racks"
	default:
		return true
	}
}

func buildInventoryPage(snapshot inventorySnapshot, filter string) inventoryPage {
	index := newInventoryIndex(snapshot)
	view := inventoryPage{View: "inventory", Filter: filter, Tabs: inventoryFilterItems(), Groups: []inventoryGroupView{}}
	if filter == "" {
		view.Filter = "all"
	}
	for i := range view.Tabs {
		view.Tabs[i].Active = strings.ToLower(strings.ReplaceAll(view.Tabs[i].Name, " ", "-")) == view.Filter
	}
	view.Tabs[0].Active = view.Filter == "all"
	view.Tabs[2].Active = view.Filter == "uninstalled"
	view.Tabs[3].Active = view.Filter == "system"
	view.Tabs[4].Active = view.Filter == "network"
	view.Tabs[5].Active = view.Filter == "rack"

	makers := make(map[string]string, len(snapshot.makers))
	for _, maker := range snapshot.makers {
		makers[maker.PublicID] = maker.Name
	}
	productLinks := make(map[string]database.ProductLink)
	for _, link := range snapshot.links {
		if _, ok := productLinks[link.ProductID]; !ok || link.Kind == "retailer" {
			productLinks[link.ProductID] = link
		}
	}
	products := make(map[string]inventoryProductView)
	for _, product := range snapshot.products {
		item := inventoryProductView{
			PublicID: product.PublicID, Name: product.Name,
			Manufacturer: makers[product.ManufacturerPublicID], Specification: inventorySpec(product),
			Notes: product.Notes.String, Assets: []inventoryAssetView{},
		}
		if link, ok := productLinks[product.ID]; ok {
			item.LinkURL = link.Url
			item.LinkLabel = link.Label.String
			if item.LinkLabel == "" {
				item.LinkLabel = "Product reference"
			}
		}
		products[product.PublicID] = item
	}
	for _, asset := range snapshot.assets {
		product := index.productsByID[asset.ProductID]
		item := inventoryAssetFor(asset, index)
		category := inventoryCategory(product.Kind)
		if !filterInventory(item, category, view.Filter) {
			continue
		}
		card := products[product.PublicID]
		card.Assets = append(card.Assets, item)
		products[product.PublicID] = card
	}
	order := []string{"Systems", "Memory", "Processors", "Drives", "Network equipment", "Racks", "Other"}
	for _, name := range order {
		group := inventoryGroupView{Name: name, Products: []inventoryProductView{}}
		for _, product := range snapshot.products {
			if inventoryCategory(product.Kind) != name {
				continue
			}
			card := products[product.PublicID]
			if len(card.Assets) == 0 {
				continue
			}
			group.Count += len(card.Assets)
			group.Products = append(group.Products, card)
		}
		if len(group.Products) == 0 {
			continue
		}
		sort.Slice(group.Products, func(i, j int) bool { return group.Products[i].Name < group.Products[j].Name })
		view.Groups = append(view.Groups, group)
	}
	return view
}

func inventoryAssetFor(asset database.ListAssetDetailsRow, index inventoryIndex) inventoryAssetView {
	product := index.productsByID[asset.ProductID]
	item := inventoryAssetView{
		PublicID: asset.PublicID, Name: asset.Name.String,
		PurchasePublicID: asset.PurchasePublicID.String, Placement: index.placement(asset), Notes: asset.Notes.String,
	}
	if !asset.Name.Valid {
		item.Name = product.Name
	}
	if asset.ParentAssetID.Valid {
		parent := index.assetsByID[asset.ParentAssetID.String]
		item.ParentAssetPublicID = parent.PublicID
		item.ParentAssetName = parent.Name.String
		if !parent.Name.Valid {
			item.ParentAssetName = index.productsByID[parent.ProductID].Name
		}
	}
	if machine, ok := index.machineFor(asset); ok {
		item.MachineName, item.MachinePublicID, item.Status = machine.Name, machine.PublicID, "Installed"
	} else if asset.ParentAssetID.Valid || asset.AreaID.Valid {
		item.Status = "Placed"
	} else {
		item.Status = "Unplaced"
	}
	item.IsSpare = item.MachinePublicID == "" && (product.Kind == "memory" || product.Kind == "processor" || product.Kind == "drive" || product.Kind == "network_adapter")
	if item.IsSpare {
		item.Status = "Uninstalled"
	}
	return item
}

func (s consoleServer) inventory(w http.ResponseWriter, r *http.Request) {
	snapshot, err := loadInventory(r.Context(), s.api.queries)
	if err != nil {
		s.renderError(w, err)
		return
	}
	filter := r.URL.Query().Get("filter")
	switch filter {
	case "", "all", "memory", "uninstalled", "system", "network", "rack":
	default:
		http.Error(w, "unknown inventory filter", http.StatusBadRequest)
		return
	}
	page := buildInventoryPage(snapshot, filter)
	if r.URL.Query().Has("product") || r.URL.Query().Has("asset") {
		page.Detail, err = s.loadInventoryDetail(r.Context(), snapshot, r.URL.Query().Get("product"), r.URL.Query().Get("asset"))
		if err != nil {
			http.Error(w, "inventory record not found", http.StatusNotFound)
			return
		}
	}
	s.render(w, "page", page)
}

func consoleMoney(cents int64, currency string) string {
	amount := fmt.Sprintf("%d", cents/100)
	for index := len(amount) - 3; index > 0; index -= 3 {
		amount = amount[:index] + "," + amount[index:]
	}
	value := fmt.Sprintf("%s.%02d", amount, cents%100)
	if currency == "USD" {
		return "$" + value
	}
	return currency + " " + value
}

func assetLabel(asset database.ListAssetDetailsRow, index inventoryIndex) string {
	label := asset.Name.String
	if !asset.Name.Valid {
		label = index.productsByID[asset.ProductID].Name
	}
	if machine, ok := index.machineFor(asset); ok && asset.ID != machine.AssetID.String {
		return label + " · " + machine.Name
	}
	return label
}

func displayPurchaseDate(value sql.NullString) string {
	if !value.Valid {
		return "Date unknown"
	}
	parsed, err := time.Parse(time.DateOnly, value.String)
	if err != nil {
		return value.String
	}
	return parsed.Format("Jan 2, 2006")
}
