package hlims

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type assetReferences struct {
	productID     string
	parentAssetID sql.NullString
	areaID        sql.NullString
}

func (s apiServer) ListAssets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListAssetDetails(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Asset, 0, len(rows))
	for _, row := range rows {
		items = append(items, assetListResponse(row))
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
	references, err := s.assetReferences(r, body, "")
	if err != nil {
		writeInputOrDatabaseError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateAsset(r.Context(), database.CreateAssetParams{
		ID: id, PublicID: publicID, ProductID: references.productID,
		ParentAssetID: references.parentAssetID, AreaID: references.areaID,
		Name: nullableTrimmed(body.Name), SerialNumber: nullableTrimmed(body.SerialNumber),
		SystemUuid: nullableTrimmed(body.SystemUuid), Notes: nullableTrimmed(body.Notes),
	})
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
	row, err := s.queries.GetAssetByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, assetDetailResponse(row))
}

func (s apiServer) UpdateAsset(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.AssetWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetAssetByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	references, err := s.assetReferences(r, body, existing.ID)
	if err != nil {
		writeInputOrDatabaseError(w, err)
		return
	}
	_, err = s.queries.UpdateAsset(r.Context(), database.UpdateAssetParams{
		ProductID: references.productID, ParentAssetID: references.parentAssetID, AreaID: references.areaID,
		Name: nullableTrimmed(body.Name), SerialNumber: nullableTrimmed(body.SerialNumber),
		SystemUuid: nullableTrimmed(body.SystemUuid), Notes: nullableTrimmed(body.Notes), PublicID: publicID,
	})
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

func (s apiServer) assetReferences(r *http.Request, body api.AssetWrite, currentID string) (assetReferences, error) {
	product, err := s.queries.GetProductByPublicID(r.Context(), body.ProductPublicId)
	if err != nil {
		return assetReferences{}, err
	}
	references := assetReferences{productID: product.ID}
	if !body.Placement.Type.Valid() {
		return assetReferences{}, errors.New("placement type must be area, asset, or unplaced")
	}
	switch body.Placement.Type {
	case api.AssetPlacementTypeArea:
		if body.Placement.AreaPublicId == nil || body.Placement.ParentAssetPublicId != nil {
			return assetReferences{}, errors.New("area placement requires only areaPublicId")
		}
		area, err := s.queries.GetAreaByPublicID(r.Context(), *body.Placement.AreaPublicId)
		if err != nil {
			return assetReferences{}, err
		}
		references.areaID = sql.NullString{String: area.ID, Valid: true}
	case api.AssetPlacementTypeAsset:
		if body.Placement.ParentAssetPublicId == nil || body.Placement.AreaPublicId != nil {
			return assetReferences{}, errors.New("asset placement requires only parentAssetPublicId")
		}
		parent, err := s.queries.GetAssetByPublicID(r.Context(), *body.Placement.ParentAssetPublicId)
		if err != nil {
			return assetReferences{}, err
		}
		if parent.ID == currentID {
			return assetReferences{}, errors.New("asset cannot contain itself")
		}
		parentProduct, err := s.queries.GetProductByPublicID(r.Context(), parent.ProductPublicID)
		if err != nil {
			return assetReferences{}, err
		}
		if parentProduct.Kind != string(api.System) {
			return assetReferences{}, errors.New("parent asset must have a system product")
		}
		references.parentAssetID = sql.NullString{String: parent.ID, Valid: true}
	case api.AssetPlacementTypeUnplaced:
		if body.Placement.AreaPublicId != nil || body.Placement.ParentAssetPublicId != nil {
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

func assetDetailResponse(row database.GetAssetByPublicIDRow) api.Asset {
	return assetResponse(row.PublicID, row.ProductPublicID, row.ParentAssetPublicID, row.AreaPublicID,
		row.EffectiveAreaPublicID, row.Name, row.SerialNumber, row.SystemUuid, row.Notes)
}

func assetListResponse(row database.ListAssetDetailsRow) api.Asset {
	return assetResponse(row.PublicID, row.ProductPublicID, row.ParentAssetPublicID, row.AreaPublicID,
		row.EffectiveAreaPublicID, row.Name, row.SerialNumber, row.SystemUuid, row.Notes)
}

func assetResponse(
	publicID, productPublicID string,
	parentAssetPublicID, areaPublicID, effectiveAreaPublicID, name, serialNumber, systemUUID, notes sql.NullString,
) api.Asset {
	placement := api.AssetPlacement{Type: api.AssetPlacementTypeUnplaced}
	if parentAssetPublicID.Valid {
		placement.Type = api.AssetPlacementTypeAsset
		placement.ParentAssetPublicId = stringPointer(parentAssetPublicID)
	} else if areaPublicID.Valid {
		placement.Type = api.AssetPlacementTypeArea
		placement.AreaPublicId = stringPointer(areaPublicID)
	}
	return api.Asset{
		PublicId: publicID, ProductPublicId: productPublicID, Placement: placement,
		EffectiveAreaPublicId: stringPointer(effectiveAreaPublicID), Name: stringPointer(name),
		SerialNumber: stringPointer(serialNumber), SystemUuid: stringPointer(systemUUID), Notes: stringPointer(notes),
	}
}
