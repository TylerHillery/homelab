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
	rackSpec      *api.RackSpec
	portProfiles  []api.PortProfile
	links         []api.ProductLink
}

func (s apiServer) ListProducts(w http.ResponseWriter, r *http.Request) {
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	rows, err := queries.ListProductDetails(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	profiles, err := queries.ListProductPortProfiles(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	links, err := queries.ListProductLinks(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	profilesByProduct := make(map[string][]database.ProductPortProfile)
	for _, profile := range profiles {
		profilesByProduct[profile.ProductID] = append(profilesByProduct[profile.ProductID], profile)
	}
	linksByProduct := make(map[string][]database.ProductLink)
	for _, link := range links {
		linksByProduct[link.ProductID] = append(linksByProduct[link.ProductID], link)
	}
	items := make([]api.Product, 0, len(rows))
	for _, row := range rows {
		items = append(items, productListResponse(row, profilesByProduct[row.ID], linksByProduct[row.ID]))
	}
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
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
	tx, err := s.db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	row, err := queries.GetProductByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	profiles, err := queries.ListProductPortProfilesByProductID(r.Context(), row.ID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	links, err := queries.ListProductLinksByProductID(r.Context(), row.ID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, productDetailResponse(row, profiles, links))
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
		return normalizedProduct{}, errors.New("kind must be system, processor, memory, drive, rack, router, switch, access_point, or network_adapter")
	}
	product := normalizedProduct{
		kind: body.Kind, name: name, partNumber: nullableTrimmed(body.PartNumber), notes: nullableTrimmed(body.Notes),
		processorSpec: body.ProcessorSpec, memorySpec: body.MemorySpec, driveSpec: body.DriveSpec,
		rackSpec: body.RackSpec,
	}
	if body.PortProfiles != nil {
		product.portProfiles = append(product.portProfiles, (*body.PortProfiles)...)
	}
	if body.Links != nil {
		product.links = append(product.links, (*body.Links)...)
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
	if body.RackSpec != nil {
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
	case api.Rack:
		if specCount != 1 || body.RackSpec == nil {
			return normalizedProduct{}, errors.New("rack products require only rackSpec")
		}
		body.RackSpec.MountingStandard = strings.TrimSpace(body.RackSpec.MountingStandard)
		if body.RackSpec.RackUnits < 1 || body.RackSpec.MountingStandard == "" {
			return normalizedProduct{}, errors.New("rackSpec requires positive rackUnits and mountingStandard")
		}
	case api.Router, api.Switch, api.AccessPoint, api.NetworkAdapter:
		if specCount != 0 {
			return normalizedProduct{}, errors.New("network equipment products must not include a scalar specification")
		}
	}
	if body.Kind != api.Router && body.Kind != api.Switch && body.Kind != api.AccessPoint && body.Kind != api.NetworkAdapter && len(product.portProfiles) > 0 {
		return normalizedProduct{}, errors.New("portProfiles are allowed only for router, switch, access_point, or network_adapter products")
	}
	for index := range product.portProfiles {
		profile := &product.portProfiles[index]
		profile.Connector = strings.TrimSpace(profile.Connector)
		profile.Name = stringPointer(nullableTrimmed(profile.Name))
		if profile.PortCount < 1 || profile.SpeedMbps < 1 || profile.Connector == "" {
			return normalizedProduct{}, errors.New("portProfiles require positive portCount and speedMbps and a connector")
		}
	}
	seenURLs := make(map[string]struct{}, len(product.links))
	for index := range product.links {
		link := &product.links[index]
		if !link.Kind.Valid() {
			return normalizedProduct{}, errors.New("product link kind is invalid")
		}
		link.Label = stringPointer(nullableTrimmed(link.Label))
		link.Url, err = normalizeInventoryURL(link.Url)
		if err != nil {
			return normalizedProduct{}, err
		}
		if _, exists := seenURLs[link.Url]; exists {
			return normalizedProduct{}, errors.New("product links must have unique URLs")
		}
		seenURLs[link.Url] = struct{}{}
	}
	return product, nil
}

func insertProductSpec(ctx context.Context, queries *database.Queries, productID string, product normalizedProduct) error {
	switch product.kind {
	case api.System:
	case api.Router, api.Switch, api.AccessPoint, api.NetworkAdapter:
	case api.Processor:
		spec := product.processorSpec
		_, err := queries.CreateProcessorSpec(ctx, database.CreateProcessorSpecParams{
			ProductID: productID, CoreCount: int64(spec.CoreCount), ThreadCount: int64(spec.ThreadCount),
			BaseClockMhz: nullInt(spec.BaseClockMhz), Virtualization: nullableTrimmed(spec.Virtualization),
		})
		if err != nil {
			return err
		}
	case api.Memory:
		spec := product.memorySpec
		_, err := queries.CreateMemorySpec(ctx, database.CreateMemorySpecParams{
			ProductID: productID, CapacityBytes: spec.CapacityBytes, MemoryType: strings.TrimSpace(spec.MemoryType),
			FormFactor: nullableTrimmed(spec.FormFactor), SpeedMts: nullInt(spec.SpeedMts),
		})
		if err != nil {
			return err
		}
	case api.Drive:
		spec := product.driveSpec
		_, err := queries.CreateDriveSpec(ctx, database.CreateDriveSpecParams{
			ProductID: productID, CapacityBytes: spec.CapacityBytes, MediaKind: string(spec.MediaKind),
			InterfaceKind: nullableEnum(spec.InterfaceKind),
		})
		if err != nil {
			return err
		}
	case api.Rack:
		spec := product.rackSpec
		_, err := queries.CreateRackSpec(ctx, database.CreateRackSpecParams{
			ProductID: productID, RackUnits: int64(spec.RackUnits), MountingStandard: spec.MountingStandard,
		})
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported product kind")
	}
	for position, profile := range product.portProfiles {
		if _, err := queries.CreateProductPortProfile(ctx, database.CreateProductPortProfileParams{
			ProductID: productID, Position: int64(position), Name: nullableTrimmed(profile.Name),
			PortCount: int64(profile.PortCount), Connector: profile.Connector, SpeedMbps: int64(profile.SpeedMbps),
		}); err != nil {
			return err
		}
	}
	for position, link := range product.links {
		if _, err := queries.CreateProductLink(ctx, database.CreateProductLinkParams{
			ProductID: productID, Position: int64(position), Kind: string(link.Kind),
			Label: nullableTrimmed(link.Label), Url: link.Url,
		}); err != nil {
			return err
		}
	}
	return nil
}

func deleteProductSpecs(ctx context.Context, queries *database.Queries, productID string) error {
	if _, err := queries.DeleteProductLinks(ctx, productID); err != nil {
		return err
	}
	if _, err := queries.DeleteProductPortProfiles(ctx, productID); err != nil {
		return err
	}
	if _, err := queries.DeleteProcessorSpec(ctx, productID); err != nil {
		return err
	}
	if _, err := queries.DeleteMemorySpec(ctx, productID); err != nil {
		return err
	}
	if _, err := queries.DeleteDriveSpec(ctx, productID); err != nil {
		return err
	}
	_, err := queries.DeleteRackSpec(ctx, productID)
	return err
}

func productDetailResponse(row database.GetProductByPublicIDRow, profiles []database.ProductPortProfile, links []database.ProductLink) api.Product {
	return productResponse(row.PublicID, row.ManufacturerPublicID, row.Kind, row.Name, row.PartNumber, row.Notes,
		row.CoreCount, row.ThreadCount, row.BaseClockMhz, row.Virtualization,
		row.MemoryCapacityBytes, row.MemoryType, row.FormFactor, row.SpeedMts,
		row.DriveCapacityBytes, row.MediaKind, row.InterfaceKind, row.RackUnits, row.MountingStandard, profiles, links)
}

func productListResponse(row database.ListProductDetailsRow, profiles []database.ProductPortProfile, links []database.ProductLink) api.Product {
	return productResponse(row.PublicID, row.ManufacturerPublicID, row.Kind, row.Name, row.PartNumber, row.Notes,
		row.CoreCount, row.ThreadCount, row.BaseClockMhz, row.Virtualization,
		row.MemoryCapacityBytes, row.MemoryType, row.FormFactor, row.SpeedMts,
		row.DriveCapacityBytes, row.MediaKind, row.InterfaceKind, row.RackUnits, row.MountingStandard, profiles, links)
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
	rackUnits sql.NullInt64,
	mountingStandard sql.NullString,
	profileRows []database.ProductPortProfile,
	linkRows []database.ProductLink,
) api.Product {
	profiles := make([]api.PortProfile, 0, len(profileRows))
	for _, profile := range profileRows {
		profiles = append(profiles, api.PortProfile{
			Name: stringPointer(profile.Name), PortCount: int(profile.PortCount),
			Connector: profile.Connector, SpeedMbps: int(profile.SpeedMbps),
		})
	}
	links := make([]api.ProductLink, 0, len(linkRows))
	for _, link := range linkRows {
		links = append(links, api.ProductLink{Kind: api.ProductLinkKind(link.Kind), Label: stringPointer(link.Label), Url: link.Url})
	}
	product := api.Product{
		PublicId: publicID, ManufacturerPublicId: manufacturerPublicID, Kind: api.ProductKind(kind), Name: name,
		PartNumber: stringPointer(partNumber), Notes: stringPointer(notes), PortProfiles: &profiles, Links: &links,
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
	if rackUnits.Valid {
		product.RackSpec = &api.RackSpec{RackUnits: int(rackUnits.Int64), MountingStandard: mountingStandard.String}
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
