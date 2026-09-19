package hlims

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type normalizedProduct struct {
	kind          api.ProductKind
	name          string
	partNumber    sql.NullString
	notes         sql.NullString
	processorSpec *api.ProcessorSpec
	memorySpec    *api.MemorySpec
	driveSpec     *api.DriveSpec
}

func (s apiServer) ListProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListProductDetails(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Product, 0, len(rows))
	for _, row := range rows {
		items = append(items, productListResponse(row))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Product `json:"items"`
	}{Items: items})
}

func (s apiServer) CreateProduct(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.ProductWrite](w, r)
	if !ok {
		return
	}
	product, err := normalizeProduct(body)
	if err != nil {
		writeInputError(w, err)
		return
	}
	manufacturer, err := s.queries.GetManufacturerByPublicID(r.Context(), body.ManufacturerPublicId)
	if err != nil {
		writeDatabaseError(w, err)
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
	row, err := queries.CreateProduct(r.Context(), database.CreateProductParams{
		ID: id, PublicID: publicID, ManufacturerID: manufacturer.ID, Kind: string(product.kind),
		Name: product.name, PartNumber: product.partNumber, Notes: product.notes,
	})
	if err == nil {
		err = insertProductSpec(r.Context(), queries, row.ID, product)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeProduct(w, r, publicID, http.StatusCreated)
}

func (s apiServer) GetProduct(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeProduct(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeProduct(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetProductByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, productDetailResponse(row))
}

func (s apiServer) UpdateProduct(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.ProductWrite](w, r)
	if !ok {
		return
	}
	product, err := normalizeProduct(body)
	if err != nil {
		writeInputError(w, err)
		return
	}
	existing, err := s.queries.GetProductByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	manufacturer, err := s.queries.GetManufacturerByPublicID(r.Context(), body.ManufacturerPublicId)
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
	if err = deleteProductSpecs(r.Context(), queries, existing.ID); err == nil {
		_, err = queries.UpdateProduct(r.Context(), database.UpdateProductParams{
			ManufacturerID: manufacturer.ID, Kind: string(product.kind), Name: product.name,
			PartNumber: product.partNumber, Notes: product.notes, PublicID: publicID,
		})
	}
	if err == nil {
		err = insertProductSpec(r.Context(), queries, existing.ID, product)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeProduct(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteProduct(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteProduct(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func normalizeProduct(body api.ProductWrite) (normalizedProduct, error) {
	name, err := normalizeName(body.Name)
	if err != nil {
		return normalizedProduct{}, err
	}
	if !body.Kind.Valid() {
		return normalizedProduct{}, errors.New("kind must be system, processor, memory, or drive")
	}
	product := normalizedProduct{
		kind: body.Kind, name: name, partNumber: nullableTrimmed(body.PartNumber), notes: nullableTrimmed(body.Notes),
		processorSpec: body.ProcessorSpec, memorySpec: body.MemorySpec, driveSpec: body.DriveSpec,
	}
	specCount := 0
	if body.ProcessorSpec != nil {
		specCount++
	}
	if body.MemorySpec != nil {
		specCount++
	}
	if body.DriveSpec != nil {
		specCount++
	}
	switch body.Kind {
	case api.System:
		if specCount != 0 {
			return normalizedProduct{}, errors.New("system products must not include a specification")
		}
	case api.Processor:
		if specCount != 1 || body.ProcessorSpec == nil {
			return normalizedProduct{}, errors.New("processor products require only processorSpec")
		}
		if body.ProcessorSpec.CoreCount < 1 || body.ProcessorSpec.ThreadCount < body.ProcessorSpec.CoreCount {
			return normalizedProduct{}, errors.New("processorSpec threadCount must be at least coreCount and both must be positive")
		}
		if body.ProcessorSpec.BaseClockMhz != nil && *body.ProcessorSpec.BaseClockMhz < 1 {
			return normalizedProduct{}, errors.New("processorSpec baseClockMhz must be positive")
		}
	case api.Memory:
		if specCount != 1 || body.MemorySpec == nil {
			return normalizedProduct{}, errors.New("memory products require only memorySpec")
		}
		if body.MemorySpec.CapacityBytes < 1 || strings.TrimSpace(body.MemorySpec.MemoryType) == "" {
			return normalizedProduct{}, errors.New("memorySpec requires positive capacityBytes and memoryType")
		}
		if body.MemorySpec.SpeedMts != nil && *body.MemorySpec.SpeedMts < 1 {
			return normalizedProduct{}, errors.New("memorySpec speedMts must be positive")
		}
	case api.Drive:
		if specCount != 1 || body.DriveSpec == nil {
			return normalizedProduct{}, errors.New("drive products require only driveSpec")
		}
		if body.DriveSpec.CapacityBytes < 1 || !body.DriveSpec.MediaKind.Valid() {
			return normalizedProduct{}, errors.New("driveSpec requires positive capacityBytes and valid mediaKind")
		}
		if body.DriveSpec.InterfaceKind != nil && !body.DriveSpec.InterfaceKind.Valid() {
			return normalizedProduct{}, errors.New("driveSpec interfaceKind is invalid")
		}
	}
	return product, nil
}

func insertProductSpec(ctx context.Context, queries *database.Queries, productID string, product normalizedProduct) error {
	switch product.kind {
	case api.System:
		return nil
	case api.Processor:
		spec := product.processorSpec
		_, err := queries.CreateProcessorSpec(ctx, database.CreateProcessorSpecParams{
			ProductID: productID, CoreCount: int64(spec.CoreCount), ThreadCount: int64(spec.ThreadCount),
			BaseClockMhz: nullInt(spec.BaseClockMhz), Virtualization: nullableTrimmed(spec.Virtualization),
		})
		return err
	case api.Memory:
		spec := product.memorySpec
		_, err := queries.CreateMemorySpec(ctx, database.CreateMemorySpecParams{
			ProductID: productID, CapacityBytes: spec.CapacityBytes, MemoryType: strings.TrimSpace(spec.MemoryType),
			FormFactor: nullableTrimmed(spec.FormFactor), SpeedMts: nullInt(spec.SpeedMts),
		})
		return err
	case api.Drive:
		spec := product.driveSpec
		_, err := queries.CreateDriveSpec(ctx, database.CreateDriveSpecParams{
			ProductID: productID, CapacityBytes: spec.CapacityBytes, MediaKind: string(spec.MediaKind),
			InterfaceKind: nullableEnum(spec.InterfaceKind),
		})
		return err
	default:
		return errors.New("unsupported product kind")
	}
}

func deleteProductSpecs(ctx context.Context, queries *database.Queries, productID string) error {
	if _, err := queries.DeleteProcessorSpec(ctx, productID); err != nil {
		return err
	}
	if _, err := queries.DeleteMemorySpec(ctx, productID); err != nil {
		return err
	}
	_, err := queries.DeleteDriveSpec(ctx, productID)
	return err
}

func productDetailResponse(row database.GetProductByPublicIDRow) api.Product {
	return productResponse(row.PublicID, row.ManufacturerPublicID, row.Kind, row.Name, row.PartNumber, row.Notes,
		row.CoreCount, row.ThreadCount, row.BaseClockMhz, row.Virtualization,
		row.MemoryCapacityBytes, row.MemoryType, row.FormFactor, row.SpeedMts,
		row.DriveCapacityBytes, row.MediaKind, row.InterfaceKind)
}

func productListResponse(row database.ListProductDetailsRow) api.Product {
	return productResponse(row.PublicID, row.ManufacturerPublicID, row.Kind, row.Name, row.PartNumber, row.Notes,
		row.CoreCount, row.ThreadCount, row.BaseClockMhz, row.Virtualization,
		row.MemoryCapacityBytes, row.MemoryType, row.FormFactor, row.SpeedMts,
		row.DriveCapacityBytes, row.MediaKind, row.InterfaceKind)
}

func productResponse(
	publicID, manufacturerPublicID, kind, name string,
	partNumber, notes sql.NullString,
	coreCount, threadCount, baseClockMhz sql.NullInt64,
	virtualization sql.NullString,
	memoryCapacity sql.NullInt64,
	memoryType, formFactor sql.NullString,
	speedMts, driveCapacity sql.NullInt64,
	mediaKind, interfaceKind sql.NullString,
) api.Product {
	product := api.Product{
		PublicId: publicID, ManufacturerPublicId: manufacturerPublicID, Kind: api.ProductKind(kind), Name: name,
		PartNumber: stringPointer(partNumber), Notes: stringPointer(notes),
	}
	if coreCount.Valid {
		product.ProcessorSpec = &api.ProcessorSpec{
			CoreCount: int(coreCount.Int64), ThreadCount: int(threadCount.Int64), BaseClockMhz: intPointer(baseClockMhz),
			Virtualization: stringPointer(virtualization),
		}
	}
	if memoryCapacity.Valid {
		product.MemorySpec = &api.MemorySpec{
			CapacityBytes: memoryCapacity.Int64, MemoryType: memoryType.String, FormFactor: stringPointer(formFactor),
			SpeedMts: intPointer(speedMts),
		}
	}
	if driveCapacity.Valid {
		product.DriveSpec = &api.DriveSpec{
			CapacityBytes: driveCapacity.Int64, MediaKind: api.StorageMediaKind(mediaKind.String),
			InterfaceKind: productStorageInterfacePointer(interfaceKind),
		}
	}
	return product
}

func productStorageInterfacePointer(value sql.NullString) *api.StorageInterfaceKind {
	if !value.Valid {
		return nil
	}
	result := api.StorageInterfaceKind(value.String)
	return &result
}
