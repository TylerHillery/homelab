package hlims

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type normalizedPurchase struct {
	totalPriceCents int64
	currency        string
	purchasedOn     sql.NullString
	source          sql.NullString
	orderReference  sql.NullString
	notes           sql.NullString
	assetPublicIDs  []string
	assetIDs        []string
	links           []api.PurchaseLink
	lines           []api.PurchaseLine
}

type purchaseWriteRequest struct {
	AssetPublicIds  []api.PublicId      `json:"assetPublicIds"`
	Currency        string              `json:"currency"`
	Links           *[]api.PurchaseLink `json:"links,omitempty"`
	Lines           *[]api.PurchaseLine `json:"lines,omitempty"`
	Notes           *string             `json:"notes,omitempty"`
	OrderReference  *string             `json:"orderReference,omitempty"`
	PurchasedOn     *openapi_types.Date `json:"purchasedOn,omitempty"`
	Source          *string             `json:"source,omitempty"`
	TotalPriceCents *int64              `json:"totalPriceCents"`
}

func (s apiServer) ListPurchases(w http.ResponseWriter, r *http.Request) {
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	rows, err := queries.ListPurchases(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	assetRows, err := queries.ListPurchaseAssets(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	linkRows, err := queries.ListPurchaseLinks(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	lineRows, err := queries.ListPurchaseLines(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	assetsByPurchase := make(map[string][]string)
	for _, asset := range assetRows {
		assetsByPurchase[asset.PurchaseID] = append(assetsByPurchase[asset.PurchaseID], asset.AssetPublicID)
	}
	linksByPurchase := make(map[string][]database.PurchaseLink)
	for _, link := range linkRows {
		linksByPurchase[link.PurchaseID] = append(linksByPurchase[link.PurchaseID], link)
	}
	linesByPurchase := make(map[string][]api.PurchaseLine)
	for _, line := range lineRows {
		linesByPurchase[line.PurchaseID] = append(linesByPurchase[line.PurchaseID], purchaseLineResponse(line.ProductPublicID, line.Description, line.Quantity, line.SubtotalCents, line.IncludeInHomelabTotal))
	}
	items := make([]api.Purchase, 0, len(rows))
	for _, row := range rows {
		items = append(items, purchaseResponse(row, assetsByPurchase[row.ID], linksByPurchase[row.ID], linesByPurchase[row.ID]))
	}
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Purchase `json:"items"`
	}{Items: items})
}

func (s apiServer) CreatePurchase(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[purchaseWriteRequest](w, r)
	if !ok {
		return
	}
	purchase, err := normalizePurchase(body)
	if err != nil {
		writeInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	purchase.assetIDs, err = resolvePurchaseAssets(r.Context(), queries, purchase.assetPublicIDs)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	row, err := queries.CreatePurchase(r.Context(), database.CreatePurchaseParams{
		ID: id, PublicID: publicID, PrimaryAssetID: purchase.assetIDs[0],
		TotalPriceCents: purchase.totalPriceCents, Currency: purchase.currency,
		PurchasedOn: purchase.purchasedOn, Source: purchase.source, OrderReference: purchase.orderReference, Notes: purchase.notes,
	})
	if err == nil {
		err = insertPurchaseChildren(r.Context(), queries, row.ID, purchase)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writePurchase(w, r, publicID, http.StatusCreated)
}

func (s apiServer) GetPurchase(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writePurchase(w, r, publicID, http.StatusOK)
}

func (s apiServer) writePurchase(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	row, err := queries.GetPurchaseByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	assets, err := queries.ListPurchaseAssetsByPurchaseID(r.Context(), row.ID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	links, err := queries.ListPurchaseLinksByPurchaseID(r.Context(), row.ID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	lineRows, err := queries.ListPurchaseLinesByPurchaseID(r.Context(), row.ID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	lines := make([]api.PurchaseLine, 0, len(lineRows))
	for _, line := range lineRows {
		lines = append(lines, purchaseLineResponse(line.ProductPublicID, line.Description, line.Quantity, line.SubtotalCents, line.IncludeInHomelabTotal))
	}
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, purchaseResponse(row, assets, links, lines))
}

func (s apiServer) UpdatePurchase(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[purchaseWriteRequest](w, r)
	if !ok {
		return
	}
	purchase, err := normalizePurchase(body)
	if err != nil {
		writeInputError(w, err)
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	existing, err := queries.GetPurchaseByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	purchase.assetIDs, err = resolvePurchaseAssets(r.Context(), queries, purchase.assetPublicIDs)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	if _, err = queries.DeletePurchaseLines(r.Context(), existing.ID); err == nil {
		_, err = queries.DeletePurchaseLinks(r.Context(), existing.ID)
	}
	if err == nil {
		_, err = queries.DeletePurchaseAssets(r.Context(), existing.ID)
	}
	if err == nil {
		_, err = queries.UpdatePurchase(r.Context(), database.UpdatePurchaseParams{
			TotalPriceCents: purchase.totalPriceCents, PrimaryAssetID: purchase.assetIDs[0], Currency: purchase.currency,
			PurchasedOn: purchase.purchasedOn, Source: purchase.source, OrderReference: purchase.orderReference, Notes: purchase.notes, PublicID: publicID,
		})
	}
	if err == nil {
		err = insertPurchaseChildren(r.Context(), queries, existing.ID, purchase)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writePurchase(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeletePurchase(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeletePurchase(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func normalizePurchase(body purchaseWriteRequest) (normalizedPurchase, error) {
	if body.TotalPriceCents == nil {
		return normalizedPurchase{}, errors.New("totalPriceCents is required")
	}
	if *body.TotalPriceCents < 0 {
		return normalizedPurchase{}, errors.New("totalPriceCents must not be negative")
	}
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	if len(currency) != 3 {
		return normalizedPurchase{}, errors.New("currency must contain three ASCII letters")
	}
	for _, char := range currency {
		if char < 'A' || char > 'Z' {
			return normalizedPurchase{}, errors.New("currency must contain three ASCII letters")
		}
	}
	if len(body.AssetPublicIds) == 0 {
		return normalizedPurchase{}, errors.New("assetPublicIds must contain at least one Asset")
	}
	seenAssets := make(map[string]struct{}, len(body.AssetPublicIds))
	assetPublicIDs := make([]string, 0, len(body.AssetPublicIds))
	for _, publicID := range body.AssetPublicIds {
		if _, exists := seenAssets[publicID]; exists {
			return normalizedPurchase{}, errors.New("assetPublicIds must not contain duplicates")
		}
		seenAssets[publicID] = struct{}{}
		assetPublicIDs = append(assetPublicIDs, publicID)
	}
	links := []api.PurchaseLink{}
	if body.Links != nil {
		links = append(links, (*body.Links)...)
	}
	seenURLs := make(map[string]struct{}, len(links))
	for index := range links {
		link := &links[index]
		if !link.Kind.Valid() {
			return normalizedPurchase{}, errors.New("purchase link kind is invalid")
		}
		link.Label = stringPointer(nullableTrimmed(link.Label))
		var err error
		link.Url, err = normalizeInventoryURL(link.Url)
		if err != nil {
			return normalizedPurchase{}, err
		}
		if _, exists := seenURLs[link.Url]; exists {
			return normalizedPurchase{}, errors.New("purchase links must have unique URLs")
		}
		seenURLs[link.Url] = struct{}{}
	}
	lines := []api.PurchaseLine{}
	if body.Lines != nil {
		lines = append(lines, (*body.Lines)...)
	}
	var trackedTotal int64
	allPriced := true
	for index := range lines {
		line := &lines[index]
		line.Description = strings.TrimSpace(line.Description)
		if line.IncludeInHomelabTotal == nil {
			included := true
			line.IncludeInHomelabTotal = &included
		}
		if line.Description == "" || line.Quantity < 1 {
			return normalizedPurchase{}, errors.New("purchase lines require a description and positive quantity")
		}
		if line.SubtotalCents == nil {
			allPriced = false
			continue
		}
		if *line.SubtotalCents < 0 || *line.SubtotalCents > math.MaxInt64-trackedTotal {
			return normalizedPurchase{}, errors.New("purchase line subtotal must be nonnegative and within range")
		}
		trackedTotal += *line.SubtotalCents
	}
	if allPriced && trackedTotal > *body.TotalPriceCents {
		return normalizedPurchase{}, errors.New("tracked line subtotals cannot exceed the order total")
	}
	source := nullableTrimmed(body.Source)
	orderReference := nullableTrimmed(body.OrderReference)
	if orderReference.Valid && !source.Valid {
		return normalizedPurchase{}, errors.New("orderReference requires source")
	}
	purchase := normalizedPurchase{
		totalPriceCents: *body.TotalPriceCents, currency: currency, assetPublicIDs: assetPublicIDs,
		source: source, orderReference: orderReference, notes: nullableTrimmed(body.Notes), links: links, lines: lines,
	}
	if body.PurchasedOn != nil {
		purchase.purchasedOn = sql.NullString{String: body.PurchasedOn.Format(time.DateOnly), Valid: true}
	}
	return purchase, nil
}

func insertPurchaseChildren(ctx context.Context, queries *database.Queries, purchaseID string, purchase normalizedPurchase) error {
	for _, assetID := range purchase.assetIDs[1:] {
		if err := queries.CreatePurchaseAsset(ctx, database.CreatePurchaseAssetParams{PurchaseID: purchaseID, AssetID: assetID}); err != nil {
			return err
		}
	}
	for position, link := range purchase.links {
		if _, err := queries.CreatePurchaseLink(ctx, database.CreatePurchaseLinkParams{
			PurchaseID: purchaseID, Position: int64(position), Kind: string(link.Kind),
			Label: nullableTrimmed(link.Label), Url: link.Url,
		}); err != nil {
			return err
		}
	}
	for position, line := range purchase.lines {
		var productID sql.NullString
		if line.ProductPublicId != nil {
			product, err := queries.GetProductByPublicID(ctx, *line.ProductPublicId)
			if err != nil {
				return err
			}
			productID = sql.NullString{String: product.ID, Valid: true}
		}
		if err := queries.CreatePurchaseLine(ctx, database.CreatePurchaseLineParams{
			PurchaseID: purchaseID, Position: int64(position), ProductID: productID,
			Description: line.Description, Quantity: int64(line.Quantity), SubtotalCents: nullInt64(line.SubtotalCents),
			IncludeInHomelabTotal: boolInt(line.IncludeInHomelabTotal),
		}); err != nil {
			return err
		}
	}
	return nil
}

func resolvePurchaseAssets(ctx context.Context, queries *database.Queries, publicIDs []string) ([]string, error) {
	assetIDs := make([]string, 0, len(publicIDs))
	for _, publicID := range publicIDs {
		asset, err := queries.GetAssetByPublicID(ctx, publicID)
		if err != nil {
			return nil, err
		}
		assetIDs = append(assetIDs, asset.ID)
	}
	return assetIDs, nil
}

func purchaseLineResponse(productID sql.NullString, description string, quantity int64, subtotal sql.NullInt64, included int64) api.PurchaseLine {
	isIncluded := included == 1
	return api.PurchaseLine{ProductPublicId: stringPointer(productID), Description: description, Quantity: int(quantity), SubtotalCents: int64Pointer(subtotal), IncludeInHomelabTotal: &isIncluded}
}

func purchaseResponse(row database.Purchase, assetPublicIDs []string, linkRows []database.PurchaseLink, lineRows []api.PurchaseLine) api.Purchase {
	links := make([]api.PurchaseLink, 0, len(linkRows))
	for _, link := range linkRows {
		links = append(links, api.PurchaseLink{Kind: api.PurchaseLinkKind(link.Kind), Label: stringPointer(link.Label), Url: link.Url})
	}
	purchase := api.Purchase{
		PublicId: row.PublicID, AssetPublicIds: assetPublicIDs, TotalPriceCents: row.TotalPriceCents,
		Currency: row.Currency, Source: stringPointer(row.Source), OrderReference: stringPointer(row.OrderReference),
		Notes: stringPointer(row.Notes), Links: &links,
	}
	lines := append([]api.PurchaseLine{}, lineRows...)
	purchase.Lines = &lines
	if len(lines) > 0 {
		var trackedTotal int64
		var homelabTotal int64
		complete := true
		homelabComplete := true
		for _, line := range lines {
			included := line.IncludeInHomelabTotal == nil || *line.IncludeInHomelabTotal
			if line.SubtotalCents == nil {
				complete = false
				if included {
					homelabComplete = false
				}
				continue
			}
			trackedTotal += *line.SubtotalCents
			if included {
				homelabTotal += *line.SubtotalCents
			}
		}
		if complete {
			purchase.TrackedSubtotalCents = &trackedTotal
		}
		if homelabComplete {
			purchase.HomelabSubtotalCents = &homelabTotal
		}
	}
	if row.PurchasedOn.Valid {
		parsed, _ := time.Parse(time.DateOnly, row.PurchasedOn.String)
		purchase.PurchasedOn = &openapi_types.Date{Time: parsed}
	}
	return purchase
}

func (s apiServer) GetPurchaseSummary(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListPurchaseAmounts(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	totalsByCurrency := make(map[string]api.PurchaseTotal)
	for _, row := range rows {
		total := totalsByCurrency[row.Currency]
		if row.TotalPriceCents > math.MaxInt64-total.TotalPriceCents || row.AssetCount > math.MaxInt64-total.AssetCount {
			writeDatabaseError(w, errors.New("purchase summary exceeds supported integer range"))
			return
		}
		total.Currency = row.Currency
		total.TotalPriceCents += row.TotalPriceCents
		total.PurchaseCount++
		total.AssetCount += row.AssetCount
		totalsByCurrency[row.Currency] = total
	}
	currencies := make([]string, 0, len(totalsByCurrency))
	for currency := range totalsByCurrency {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	totals := make([]api.PurchaseTotal, 0, len(currencies))
	for _, currency := range currencies {
		totals = append(totals, totalsByCurrency[currency])
	}
	writeJSON(w, http.StatusOK, api.PurchaseSummary{Totals: totals})
}
