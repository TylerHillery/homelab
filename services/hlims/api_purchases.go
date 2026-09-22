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
	notes           sql.NullString
	assetPublicIDs  []string
	assetIDs        []string
	links           []api.PurchaseLink
}

type purchaseWriteRequest struct {
	AssetPublicIds  []api.PublicId      `json:"assetPublicIds"`
	Currency        string              `json:"currency"`
	Links           *[]api.PurchaseLink `json:"links,omitempty"`
	Notes           *string             `json:"notes,omitempty"`
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
	assetsByPurchase := make(map[string][]string)
	for _, asset := range assetRows {
		assetsByPurchase[asset.PurchaseID] = append(assetsByPurchase[asset.PurchaseID], asset.AssetPublicID)
	}
	linksByPurchase := make(map[string][]database.PurchaseLink)
	for _, link := range linkRows {
		linksByPurchase[link.PurchaseID] = append(linksByPurchase[link.PurchaseID], link)
	}
	items := make([]api.Purchase, 0, len(rows))
	for _, row := range rows {
		items = append(items, purchaseResponse(row, assetsByPurchase[row.ID], linksByPurchase[row.ID]))
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
		PurchasedOn: purchase.purchasedOn, Source: purchase.source, Notes: purchase.notes,
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
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, purchaseResponse(row, assets, links))
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
	if _, err = queries.DeletePurchaseLinks(r.Context(), existing.ID); err == nil {
		_, err = queries.DeletePurchaseAssets(r.Context(), existing.ID)
	}
	if err == nil {
		_, err = queries.UpdatePurchase(r.Context(), database.UpdatePurchaseParams{
			TotalPriceCents: purchase.totalPriceCents, PrimaryAssetID: purchase.assetIDs[0], Currency: purchase.currency,
			PurchasedOn: purchase.purchasedOn, Source: purchase.source, Notes: purchase.notes, PublicID: publicID,
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
	purchase := normalizedPurchase{
		totalPriceCents: *body.TotalPriceCents, currency: currency, assetPublicIDs: assetPublicIDs,
		source: nullableTrimmed(body.Source), notes: nullableTrimmed(body.Notes), links: links,
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

func purchaseResponse(row database.Purchase, assetPublicIDs []string, linkRows []database.PurchaseLink) api.Purchase {
	links := make([]api.PurchaseLink, 0, len(linkRows))
	for _, link := range linkRows {
		links = append(links, api.PurchaseLink{Kind: api.PurchaseLinkKind(link.Kind), Label: stringPointer(link.Label), Url: link.Url})
	}
	purchase := api.Purchase{
		PublicId: row.PublicID, AssetPublicIds: assetPublicIDs, TotalPriceCents: row.TotalPriceCents,
		Currency: row.Currency, Source: stringPointer(row.Source), Notes: stringPointer(row.Notes), Links: &links,
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
