package hlims

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type assetReferences struct {
	productID     string
	parentAssetID sql.NullString
	parentSlot    sql.NullString
	areaID        sql.NullString
}

func (s apiServer) ListAssets(w http.ResponseWriter, r *http.Request) {
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	rows, err := queries.ListAssetDetails(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	links, err := queries.ListAssetLinks(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	linksByAsset := make(map[string][]database.AssetLink)
	for _, link := range links {
		linksByAsset[link.AssetID] = append(linksByAsset[link.AssetID], link)
	}
	items := make([]api.Asset, 0, len(rows))
	for _, row := range rows {
		items = append(items, assetListResponse(row, linksByAsset[row.ID]))
	}
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Asset `json:"items"`
	}{Items: items})
}

func (s apiServer) CreateAsset(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.AssetWrite](w, r)
	if !ok {
		return
	}
	links, err := normalizeAssetLinks(body.Links)
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
	references, err := s.assetReferences(r.Context(), queries, body, "")
	if err != nil {
		writeInputOrDatabaseError(w, err)
		return
	}
	row, err := queries.CreateAsset(r.Context(), database.CreateAssetParams{
		ID: id, PublicID: publicID, ProductID: references.productID,
		ParentAssetID: references.parentAssetID, ParentSlot: references.parentSlot, AreaID: references.areaID,
		Name: nullableTrimmed(body.Name), SerialNumber: nullableTrimmed(body.SerialNumber),
		SystemUuid: nullableTrimmed(body.SystemUuid), Notes: nullableTrimmed(body.Notes),
	})
	if err == nil {
		err = insertAssetLinks(r.Context(), queries, row.ID, links)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeAsset(w, r, publicID, http.StatusCreated)
}

func (s apiServer) GetAsset(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeAsset(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeAsset(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	row, err := queries.GetAssetByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	links, err := queries.ListAssetLinksByAssetID(r.Context(), row.ID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, assetDetailResponse(row, links))
}

func (s apiServer) UpdateAsset(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.AssetWrite](w, r)
	if !ok {
		return
	}
	links, err := normalizeAssetLinks(body.Links)
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
	existing, err := queries.GetAssetByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	references, err := s.assetReferences(r.Context(), queries, body, existing.ID)
	if err != nil {
		writeInputOrDatabaseError(w, err)
		return
	}
	if _, err = queries.DeleteAssetLinks(r.Context(), existing.ID); err == nil {
		_, err = queries.UpdateAsset(r.Context(), database.UpdateAssetParams{
			ProductID: references.productID, ParentAssetID: references.parentAssetID, ParentSlot: references.parentSlot, AreaID: references.areaID,
			Name: nullableTrimmed(body.Name), SerialNumber: nullableTrimmed(body.SerialNumber),
			SystemUuid: nullableTrimmed(body.SystemUuid), Notes: nullableTrimmed(body.Notes), PublicID: publicID,
		})
	}
	if err == nil {
		err = insertAssetLinks(r.Context(), queries, existing.ID, links)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeAsset(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteAsset(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteAsset(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func (s apiServer) assetReferences(ctx context.Context, queries *database.Queries, body api.AssetWrite, currentID string) (assetReferences, error) {
	product, err := queries.GetProductByPublicID(ctx, body.ProductPublicId)
	if err != nil {
		return assetReferences{}, err
	}
	references := assetReferences{productID: product.ID}
	if !body.Placement.Type.Valid() {
		return assetReferences{}, errors.New("placement type must be area, asset, or unplaced")
	}
	switch body.Placement.Type {
	case api.AssetPlacementTypeArea:
		if body.Placement.AreaPublicId == nil || body.Placement.ParentAssetPublicId != nil || body.Placement.Slot != nil {
			return assetReferences{}, errors.New("area placement requires only areaPublicId")
		}
		area, err := queries.GetAreaByPublicID(ctx, *body.Placement.AreaPublicId)
		if err != nil {
			return assetReferences{}, err
		}
		references.areaID = sql.NullString{String: area.ID, Valid: true}
	case api.AssetPlacementTypeAsset:
		if body.Placement.ParentAssetPublicId == nil || body.Placement.AreaPublicId != nil {
			return assetReferences{}, errors.New("asset placement requires only parentAssetPublicId")
		}
		parent, err := queries.GetAssetByPublicID(ctx, *body.Placement.ParentAssetPublicId)
		if err != nil {
			return assetReferences{}, err
		}
		if parent.ID == currentID {
			return assetReferences{}, errors.New("asset cannot contain itself")
		}
		parentProduct, err := queries.GetProductByPublicID(ctx, parent.ProductPublicID)
		if err != nil {
			return assetReferences{}, err
		}
		if parentProduct.Kind != string(api.System) && parentProduct.Kind != string(api.Rack) {
			return assetReferences{}, errors.New("parent asset must have a system or rack product")
		}
		references.parentAssetID = sql.NullString{String: parent.ID, Valid: true}
		references.parentSlot = nullableTrimmed(body.Placement.Slot)
	case api.AssetPlacementTypeUnplaced:
		if body.Placement.AreaPublicId != nil || body.Placement.ParentAssetPublicId != nil || body.Placement.Slot != nil {
			return assetReferences{}, errors.New("unplaced placement must not include areaPublicId or parentAssetPublicId")
		}
	}
	return references, nil
}

func writeInputOrDatabaseError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeDatabaseError(w, err)
		return
	}
	writeInputError(w, err)
}

func assetDetailResponse(row database.GetAssetByPublicIDRow, links []database.AssetLink) api.Asset {
	return assetResponse(row.PublicID, row.ProductPublicID, row.ParentAssetPublicID, row.AreaPublicID,
		row.EffectiveAreaPublicID, row.ParentSlot, row.Name, row.SerialNumber, row.SystemUuid,
		row.PurchasePublicID, row.Notes, links)
}

func assetListResponse(row database.ListAssetDetailsRow, links []database.AssetLink) api.Asset {
	return assetResponse(row.PublicID, row.ProductPublicID, row.ParentAssetPublicID, row.AreaPublicID,
		row.EffectiveAreaPublicID, row.ParentSlot, row.Name, row.SerialNumber, row.SystemUuid,
		row.PurchasePublicID, row.Notes, links)
}

func assetResponse(
	publicID, productPublicID string,
	parentAssetPublicID, areaPublicID, effectiveAreaPublicID, parentSlot, name, serialNumber, systemUUID sql.NullString,
	purchasePublicID, notes sql.NullString,
	linkRows []database.AssetLink,
) api.Asset {
	placement := api.AssetPlacement{Type: api.AssetPlacementTypeUnplaced}
	if parentAssetPublicID.Valid {
		placement.Type = api.AssetPlacementTypeAsset
		placement.ParentAssetPublicId = stringPointer(parentAssetPublicID)
		placement.Slot = stringPointer(parentSlot)
	} else if areaPublicID.Valid {
		placement.Type = api.AssetPlacementTypeArea
		placement.AreaPublicId = stringPointer(areaPublicID)
	}
	links := make([]api.AssetLink, 0, len(linkRows))
	for _, link := range linkRows {
		links = append(links, api.AssetLink{Kind: api.AssetLinkKind(link.Kind), Label: stringPointer(link.Label), Url: link.Url})
	}
	return api.Asset{
		PublicId: publicID, ProductPublicId: productPublicID, Placement: placement,
		EffectiveAreaPublicId: stringPointer(effectiveAreaPublicID), Name: stringPointer(name),
		SerialNumber: stringPointer(serialNumber), SystemUuid: stringPointer(systemUUID),
		PurchasePublicId: stringPointer(purchasePublicID), Notes: stringPointer(notes), Links: &links,
	}
}

func normalizeAssetLinks(input *[]api.AssetLink) ([]api.AssetLink, error) {
	if input == nil {
		return []api.AssetLink{}, nil
	}
	links := append([]api.AssetLink(nil), (*input)...)
	seenURLs := make(map[string]struct{}, len(links))
	for index := range links {
		link := &links[index]
		if !link.Kind.Valid() {
			return nil, errors.New("asset link kind is invalid")
		}
		link.Label = stringPointer(nullableTrimmed(link.Label))
		var err error
		link.Url, err = normalizeInventoryURL(link.Url)
		if err != nil {
			return nil, err
		}
		if _, exists := seenURLs[link.Url]; exists {
			return nil, errors.New("asset links must have unique URLs")
		}
		seenURLs[link.Url] = struct{}{}
	}
	return links, nil
}

func insertAssetLinks(ctx context.Context, queries *database.Queries, assetID string, links []api.AssetLink) error {
	for position, link := range links {
		if _, err := queries.CreateAssetLink(ctx, database.CreateAssetLinkParams{
			AssetID: assetID, Position: int64(position), Kind: string(link.Kind),
			Label: nullableTrimmed(link.Label), Url: link.Url,
		}); err != nil {
			return err
		}
	}
	return nil
}
