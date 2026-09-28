package hlims

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type inventoryDetailFact struct {
	Label string
	Value string
}

type inventoryDetailLink struct {
	Label string
	URL   string
}

type inventoryDetailView struct {
	PublicID         string
	ProductID        string
	Title            string
	ProductName      string
	Manufacturer     string
	Kind             string
	PartNumber       string
	Notes            string
	Facts            []inventoryDetailFact
	WiFiBands        []inventoryDetailFact
	Ports            []string
	Links            []inventoryDetailLink
	Assets           []inventoryAssetView
	SelectedAsset    *inventoryAssetView
	Components       []inventoryComponentView
	PrimaryLinkURL   string
	PrimaryLinkLabel string
}

type inventoryComponentView struct {
	PublicID string
	Name     string
	Model    string
	Kind     string
	Slot     string
	Details  string
	Depth    int64
}

func buildInventoryDetail(snapshot inventorySnapshot, productID, assetID string) (*inventoryDetailView, error) {
	if (productID == "") == (assetID == "") {
		return nil, sql.ErrNoRows
	}
	index := newInventoryIndex(snapshot)
	var selected *database.ListAssetDetailsRow
	if assetID != "" {
		if _, err := ParsePublicID(assetID); err != nil {
			return nil, sql.ErrNoRows
		}
		for i := range snapshot.assets {
			if snapshot.assets[i].PublicID == assetID {
				selected = &snapshot.assets[i]
				productID = selected.ProductPublicID
				break
			}
		}
	}
	if _, err := ParsePublicID(productID); err != nil {
		return nil, sql.ErrNoRows
	}
	var product *database.ListProductDetailsRow
	for i := range snapshot.products {
		if snapshot.products[i].PublicID == productID {
			product = &snapshot.products[i]
			break
		}
	}
	if product == nil {
		return nil, sql.ErrNoRows
	}
	view := &inventoryDetailView{
		PublicID: productID, ProductID: productID, Title: product.Name, ProductName: product.Name,
		Kind: inventoryCategory(product.Kind), PartNumber: product.PartNumber.String, Notes: product.Notes.String,
		Facts: []inventoryDetailFact{}, WiFiBands: []inventoryDetailFact{}, Ports: []string{},
		Links: []inventoryDetailLink{}, Assets: []inventoryAssetView{},
		Components: []inventoryComponentView{},
	}
	for _, maker := range snapshot.makers {
		if maker.PublicID == product.ManufacturerPublicID {
			view.Manufacturer = maker.Name
			break
		}
	}
	add := func(label, value string) {
		if value != "" {
			view.Facts = append(view.Facts, inventoryDetailFact{Label: label, Value: value})
		}
	}
	switch product.Kind {
	case "memory":
		add("Capacity", fmt.Sprintf("%d GiB", product.MemoryCapacityBytes.Int64/(1024*1024*1024)))
		add("Technology", product.MemoryType.String)
		add("Form factor", product.FormFactor.String)
		if product.SpeedMts.Valid {
			add("Speed", fmt.Sprintf("%d MT/s", product.SpeedMts.Int64))
		}
	case "processor":
		add("Cores", fmt.Sprint(product.CoreCount.Int64))
		add("Threads", fmt.Sprint(product.ThreadCount.Int64))
		add("Generation", product.Generation.String)
		if product.BaseClockMhz.Valid {
			add("Base clock", fmt.Sprintf("%d MHz", product.BaseClockMhz.Int64))
		}
	case "drive":
		add("Capacity", consoleByteSize(&product.DriveCapacityBytes.Int64))
		add("Media", strings.ToUpper(product.MediaKind.String))
		add("Interface", strings.ToUpper(product.InterfaceKind.String))
	case "rack":
		add("Height", fmt.Sprintf("%dU", product.RackUnits.Int64))
		add("Mounting", product.MountingStandard.String)
	}
	if product.WifiGeneration.Valid {
		add("Wi-Fi", fmt.Sprintf("Wi-Fi %d · %s", product.WifiGeneration.Int64, product.IeeeStandard.String))
		add("Class", product.WifiClass.String)
		if product.MaxChannelWidthMhz.Valid {
			add("Channel width", fmt.Sprintf("Up to %d MHz", product.MaxChannelWidthMhz.Int64))
		}
	}
	for _, band := range snapshot.bands {
		if band.ProductID == product.ID {
			view.WiFiBands = append(view.WiFiBands, inventoryDetailFact{
				Label: band.BandGhz + " GHz", Value: fmt.Sprintf("up to %d Mbps", band.MaxLinkMbps),
			})
		}
	}
	for _, port := range snapshot.ports {
		if port.ProductID != product.ID {
			continue
		}
		speed := fmt.Sprintf("%d Mbps", port.SpeedMbps)
		if port.SpeedMbps >= 1000 {
			if port.SpeedMbps%1000 == 0 {
				speed = fmt.Sprintf("%d Gbps", port.SpeedMbps/1000)
			} else {
				speed = fmt.Sprintf("%.1f Gbps", float64(port.SpeedMbps)/1000)
			}
		}
		label := fmt.Sprintf("%d× %s · %s", port.PortCount, port.Connector, speed)
		if port.Name.Valid {
			label += " · " + port.Name.String
		}
		view.Ports = append(view.Ports, label)
	}
	for _, link := range snapshot.links {
		if link.ProductID != product.ID {
			continue
		}
		label := link.Label.String
		if label == "" {
			label = link.Kind + " reference"
		}
		view.Links = append(view.Links, inventoryDetailLink{Label: label, URL: link.Url})
		if view.PrimaryLinkURL == "" || link.Kind == "retailer" {
			view.PrimaryLinkURL, view.PrimaryLinkLabel = link.Url, label
		}
	}
	for _, asset := range snapshot.assets {
		if asset.ProductID == product.ID {
			view.Assets = append(view.Assets, inventoryAssetFor(asset, index))
		}
	}
	if selected != nil {
		asset := inventoryAssetFor(*selected, index)
		view.PublicID, view.Title, view.SelectedAsset = asset.PublicID, asset.Name, &asset
	}
	return view, nil
}

func (s consoleServer) loadInventoryDetail(ctx context.Context, snapshot inventorySnapshot, productID, assetID string) (*inventoryDetailView, error) {
	view, err := buildInventoryDetail(snapshot, productID, assetID)
	if err != nil || view.SelectedAsset == nil {
		return view, err
	}
	rows, err := s.api.queries.ListContainedAssets(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("list contained Assets: %w", err)
	}
	products := make(map[string]database.ListProductDetailsRow, len(snapshot.products))
	for _, product := range snapshot.products {
		products[product.PublicID] = product
	}
	for _, row := range rows {
		name := row.Name.String
		if !row.Name.Valid {
			name = row.ProductName
		}
		view.Components = append(view.Components, inventoryComponentView{
			PublicID: row.PublicID, Name: name, Model: row.ProductName,
			Kind: strings.ReplaceAll(row.ProductKind, "_", " "), Slot: row.ParentSlot.String,
			Details: inventorySpec(products[row.ProductPublicID]), Depth: row.Depth,
		})
	}
	return view, nil
}

func (s consoleServer) inventoryDetail(w http.ResponseWriter, r *http.Request) {
	snapshot, err := loadInventory(r.Context(), s.api.queries)
	if err != nil {
		s.renderError(w, err)
		return
	}
	detail, err := s.loadInventoryDetail(r.Context(), snapshot, r.URL.Query().Get("product"), r.URL.Query().Get("asset"))
	if err != nil {
		http.Error(w, "inventory record not found", http.StatusNotFound)
		return
	}
	s.render(w, "inventory-drawer", detail)
}
