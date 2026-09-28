package hlims

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type orderLineView struct {
	Description       string
	Quantity          int64
	Subtotal          string
	ProductPublicID   string
	IncludedInHomelab bool
}

type orderAssetView struct {
	PublicID string
	Name     string
}

type orderLinkView struct {
	URL   string
	Label string
}

type orderCardView struct {
	PublicID           string
	Source             string
	Reference          string
	Date               string
	DateSort           string
	Total              string
	TrackedSubtotal    string
	HomelabSubtotal    string
	ExcludedCount      int
	UntrackedRemainder string
	Notes              string
	Lines              []orderLineView
	Assets             []orderAssetView
	Links              []orderLinkView
}

type orderGroupView struct {
	Label  string
	Orders []orderCardView
}

type orderCurrencyView struct {
	Currency        string
	HomelabSubtotal string
	UnknownCount    int
	homelabCents    int64
}

type ordersPage struct {
	View       string
	OrderCount int
	Groups     []orderGroupView
	Totals     []orderCurrencyView
}

func (s consoleServer) orders(w http.ResponseWriter, r *http.Request) {
	snapshot, err := loadInventory(r.Context(), s.api.queries)
	if err != nil {
		s.renderError(w, err)
		return
	}
	purchases, err := s.api.queries.ListPurchases(r.Context())
	if err != nil {
		s.renderError(w, fmt.Errorf("list purchases: %w", err))
		return
	}
	assetRows, err := s.api.queries.ListPurchaseAssets(r.Context())
	if err != nil {
		s.renderError(w, fmt.Errorf("list purchased assets: %w", err))
		return
	}
	lineRows, err := s.api.queries.ListPurchaseLines(r.Context())
	if err != nil {
		s.renderError(w, fmt.Errorf("list purchase lines: %w", err))
		return
	}
	linkRows, err := s.api.queries.ListPurchaseLinks(r.Context())
	if err != nil {
		s.renderError(w, fmt.Errorf("list purchase links: %w", err))
		return
	}
	index := newInventoryIndex(snapshot)
	assetByPublicID := make(map[string]database.ListAssetDetailsRow, len(snapshot.assets))
	for _, asset := range snapshot.assets {
		assetByPublicID[asset.PublicID] = asset
	}
	byPurchase := make(map[string]int, len(purchases))
	currencies := make(map[string]string, len(purchases))
	view := ordersPage{View: "orders", OrderCount: len(purchases), Groups: []orderGroupView{}, Totals: []orderCurrencyView{}}
	currencyTotals := make(map[string]*orderCurrencyView)
	cards := make([]orderCardView, 0, len(purchases))
	for _, purchase := range purchases {
		source := purchase.Source.String
		if source == "" {
			source = "Source not recorded"
		}
		cards = append(cards, orderCardView{
			PublicID: purchase.PublicID, Source: source, Reference: purchase.OrderReference.String,
			Date: displayPurchaseDate(purchase.PurchasedOn), DateSort: purchase.PurchasedOn.String,
			Total: consoleMoney(purchase.TotalPriceCents, purchase.Currency), Notes: purchase.Notes.String,
			Lines: []orderLineView{}, Assets: []orderAssetView{}, Links: []orderLinkView{},
		})
		byPurchase[purchase.ID] = len(cards) - 1
		currencies[purchase.ID] = purchase.Currency
		if currencyTotals[purchase.Currency] == nil {
			currencyTotals[purchase.Currency] = &orderCurrencyView{Currency: purchase.Currency}
		}
	}
	for _, row := range assetRows {
		cardIndex, exists := byPurchase[row.PurchaseID]
		asset, ok := assetByPublicID[row.AssetPublicID]
		if !exists || !ok {
			continue
		}
		card := &cards[cardIndex]
		card.Assets = append(card.Assets, orderAssetView{PublicID: asset.PublicID, Name: assetLabel(asset, index)})
	}
	for _, row := range linkRows {
		cardIndex, exists := byPurchase[row.PurchaseID]
		if !exists {
			continue
		}
		card := &cards[cardIndex]
		label := row.Label.String
		if label == "" {
			switch row.Kind {
			case "receipt":
				label = "Order details"
			case "listing":
				label = "Listing"
			default:
				label = "Related link"
			}
		}
		card.Links = append(card.Links, orderLinkView{URL: row.Url, Label: label})
	}
	known := make(map[string]bool, len(purchases))
	homelabKnown := make(map[string]bool, len(purchases))
	for _, row := range purchases {
		known[row.ID] = true
		homelabKnown[row.ID] = true
	}
	lineTotals := make(map[string]int64)
	homelabTotals := make(map[string]int64)
	for _, row := range lineRows {
		cardIndex, exists := byPurchase[row.PurchaseID]
		if !exists {
			continue
		}
		card := &cards[cardIndex]
		line := orderLineView{
			Description: row.Description, Quantity: row.Quantity,
			ProductPublicID:   row.ProductPublicID.String,
			IncludedInHomelab: row.IncludeInHomelabTotal == 1,
		}
		if !line.IncludedInHomelab {
			card.ExcludedCount++
		}
		if row.SubtotalCents.Valid {
			line.Subtotal = consoleMoney(row.SubtotalCents.Int64, currencies[row.PurchaseID])
			lineTotals[row.PurchaseID] += row.SubtotalCents.Int64
			if line.IncludedInHomelab {
				homelabTotals[row.PurchaseID] += row.SubtotalCents.Int64
			}
		} else {
			known[row.PurchaseID] = false
			if line.IncludedInHomelab {
				homelabKnown[row.PurchaseID] = false
			}
		}
		card.Lines = append(card.Lines, line)
	}
	for _, purchase := range purchases {
		card := &cards[byPurchase[purchase.ID]]
		totals := currencyTotals[purchase.Currency]
		if len(card.Lines) > 0 && known[purchase.ID] {
			card.TrackedSubtotal = consoleMoney(lineTotals[purchase.ID], purchase.Currency)
			if remainder := purchase.TotalPriceCents - lineTotals[purchase.ID]; remainder > 0 {
				card.UntrackedRemainder = consoleMoney(remainder, purchase.Currency)
			}
		}
		if len(card.Lines) > 0 && homelabKnown[purchase.ID] {
			card.HomelabSubtotal = consoleMoney(homelabTotals[purchase.ID], purchase.Currency)
			totals.homelabCents += homelabTotals[purchase.ID]
		} else {
			totals.UnknownCount++
		}
	}
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].DateSort != cards[j].DateSort {
			return cards[i].DateSort > cards[j].DateSort
		}
		return cards[i].PublicID < cards[j].PublicID
	})
	for _, card := range cards {
		group := "Date not recorded"
		if len(card.DateSort) >= 4 {
			group = card.DateSort[:4]
		}
		if len(view.Groups) == 0 || view.Groups[len(view.Groups)-1].Label != group {
			view.Groups = append(view.Groups, orderGroupView{Label: group, Orders: []orderCardView{}})
		}
		view.Groups[len(view.Groups)-1].Orders = append(view.Groups[len(view.Groups)-1].Orders, card)
	}
	for _, total := range currencyTotals {
		total.HomelabSubtotal = consoleMoney(total.homelabCents, total.Currency)
		view.Totals = append(view.Totals, *total)
	}
	sort.Slice(view.Totals, func(i, j int) bool { return strings.Compare(view.Totals[i].Currency, view.Totals[j].Currency) < 0 })
	s.render(w, "page", view)
}
