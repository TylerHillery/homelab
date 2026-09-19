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

func (s apiServer) ListAreas(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListAreas(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Area, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.Area{PublicId: row.PublicID, MachineProviderPublicId: row.MachineProviderPublicID, Name: row.Name, Slug: row.Slug, ProviderCode: stringPointer(row.ProviderCode), Notes: stringPointer(row.Notes)})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Area `json:"items"`
	}{items})
}

func (s apiServer) CreateArea(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.AreaWrite](w, r)
	if !ok {
		return
	}
	provider, err := s.queries.GetMachineProviderByPublicID(r.Context(), body.MachineProviderPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, "")
	if err != nil {
		writeInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateArea(r.Context(), database.CreateAreaParams{ID: id, PublicID: publicID, MachineProviderID: provider.ID, Name: name, Slug: slug, ProviderCode: nullString(body.ProviderCode, true), Notes: nullString(body.Notes, false)})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeArea(w, r, publicID, http.StatusCreated)
}

func (s apiServer) GetArea(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeArea(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeArea(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetAreaDetailByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, api.Area{PublicId: row.PublicID, MachineProviderPublicId: row.MachineProviderPublicID, Name: row.Name, Slug: row.Slug, ProviderCode: stringPointer(row.ProviderCode), Notes: stringPointer(row.Notes)})
}

func (s apiServer) UpdateArea(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.AreaWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetAreaByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	provider, err := s.queries.GetMachineProviderByPublicID(r.Context(), body.MachineProviderPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existing.Slug)
	if err != nil {
		writeInputError(w, err)
		return
	}
	_, err = s.queries.UpdateArea(r.Context(), database.UpdateAreaParams{MachineProviderID: provider.ID, Name: name, Slug: slug, ProviderCode: nullString(body.ProviderCode, true), Notes: nullString(body.Notes, false), PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeArea(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteArea(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteArea(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func (s apiServer) CreateMachine(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.MachineWrite](w, r)
	if !ok {
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, "")
	if err != nil {
		writeInputError(w, err)
		return
	}
	values, err := s.normalizedMachine(r.Context(), body, "")
	if err != nil {
		writeInputOrDatabaseError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateMachine(r.Context(), values.createParams(id, publicID, name, slug))
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeMachine(w, r, publicID, http.StatusCreated)
}

func (s apiServer) GetMachine(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeMachine(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeMachine(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetMachineByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, machineDetailResponse(row))
}

func machineDetailResponse(row database.GetMachineByPublicIDRow) api.Machine {
	return machineResponse(machineRecord{
		publicID: row.PublicID, machineProviderPublicID: row.MachineProviderPublicID, areaPublicID: row.AreaPublicID,
		assetPublicID: row.AssetPublicID, parentMachinePublicID: row.ParentMachinePublicID,
		name: row.Name, slug: row.Slug, kind: row.Kind, virtualizationPlatform: row.VirtualizationPlatform,
		isFavorite: row.IsFavorite,
		hostname:   row.Hostname, osMachineID: row.OsMachineID, operatingSystem: row.OperatingSystem,
		operatingSystemVersion: row.OperatingSystemVersion, kernel: row.Kernel, architecture: row.Architecture,
		cpuCount: row.CpuCount, cpuAllocation: row.CpuAllocation, cpuVendor: row.CpuVendor,
		memoryBytes: row.MemoryBytes, storageBytes: row.StorageBytes, storageMediaKind: row.StorageMediaKind,
		storageInterfaceKind: row.StorageInterfaceKind, estimatedMonthlyCostCents: row.EstimatedMonthlyCostCents,
		costCurrency: row.CostCurrency, notes: row.Notes,
	})
}

func (s apiServer) UpdateMachine(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.MachineWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetMachineByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existing.Slug)
	if err != nil {
		writeInputError(w, err)
		return
	}
	values, err := s.normalizedMachine(r.Context(), body, existing.ID)
	if err != nil {
		writeInputOrDatabaseError(w, err)
		return
	}
	_, err = s.queries.UpdateMachine(r.Context(), values.updateParams(publicID, name, slug))
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeMachine(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteMachine(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteMachine(r.Context(), publicID)
	deleteResource(w, rows, err)
}

type normalizedMachineValues struct {
	machineProviderID         string
	areaID                    string
	assetID                   sql.NullString
	parentMachineID           sql.NullString
	kind                      string
	isFavorite                bool
	virtualizationPlatform    sql.NullString
	hostname                  sql.NullString
	osMachineID               sql.NullString
	operatingSystem           sql.NullString
	operatingSystemVersion    sql.NullString
	kernel                    sql.NullString
	architecture              sql.NullString
	cpuCount                  sql.NullInt64
	cpuAllocation             sql.NullString
	cpuVendor                 sql.NullString
	memoryBytes               sql.NullInt64
	storageBytes              sql.NullInt64
	storageMediaKind          sql.NullString
	storageInterfaceKind      sql.NullString
	estimatedMonthlyCostCents sql.NullInt64
	costCurrency              sql.NullString
	notes                     sql.NullString
}

func (s apiServer) normalizedMachine(ctx context.Context, body api.MachineWrite, currentID string) (normalizedMachineValues, error) {
	area, provider, err := lookupAreaAndProvider(ctx, s.queries, body.AreaPublicId, body.MachineProviderPublicId)
	if err != nil {
		return normalizedMachineValues{}, err
	}
	if area.MachineProviderID != provider.ID {
		return normalizedMachineValues{}, errors.New("area must belong to machine provider")
	}
	if !body.Kind.Valid() {
		return normalizedMachineValues{}, errors.New("kind must be bare_metal or virtual_machine")
	}
	hostname, err := normalizeHostname(body.Hostname)
	if err != nil {
		return normalizedMachineValues{}, err
	}
	values := normalizedMachineValues{
		machineProviderID: provider.ID, areaID: area.ID, kind: string(body.Kind), isFavorite: body.IsFavorite != nil && *body.IsFavorite, hostname: hostname,
		virtualizationPlatform: nullableTrimmed(body.VirtualizationPlatform),
		osMachineID:            nullableLower(body.OsMachineId), operatingSystem: nullableTrimmed(body.OperatingSystem),
		operatingSystemVersion: nullableTrimmed(body.OperatingSystemVersion), kernel: nullableTrimmed(body.Kernel),
		architecture: nullableLower(body.Architecture), cpuCount: nullInt(body.CpuCount),
		cpuAllocation: nullableEnum(body.CpuAllocation), cpuVendor: nullableTrimmed(body.CpuVendor),
		memoryBytes: nullInt64(body.MemoryBytes), storageBytes: nullInt64(body.StorageBytes),
		storageMediaKind: nullableEnum(body.StorageMediaKind), storageInterfaceKind: nullableEnum(body.StorageInterfaceKind),
		estimatedMonthlyCostCents: nullInt64(body.EstimatedMonthlyCostCents),
		costCurrency:              nullableUpper(body.CostCurrency), notes: nullableTrimmed(body.Notes),
	}
	if body.CpuCount != nil && *body.CpuCount < 1 {
		return normalizedMachineValues{}, errors.New("cpuCount must be positive")
	}
	if body.CpuAllocation != nil && !body.CpuAllocation.Valid() {
		return normalizedMachineValues{}, errors.New("cpuAllocation must be shared or dedicated")
	}
	if body.MemoryBytes != nil && *body.MemoryBytes < 1 {
		return normalizedMachineValues{}, errors.New("memoryBytes must be positive")
	}
	if body.StorageBytes != nil && *body.StorageBytes < 1 {
		return normalizedMachineValues{}, errors.New("storageBytes must be positive")
	}
	if body.StorageMediaKind != nil && !body.StorageMediaKind.Valid() {
		return normalizedMachineValues{}, errors.New("storageMediaKind must be hdd or ssd")
	}
	if body.StorageInterfaceKind != nil && !body.StorageInterfaceKind.Valid() {
		return normalizedMachineValues{}, errors.New("storageInterfaceKind is invalid")
	}
	if (body.EstimatedMonthlyCostCents == nil) != (body.CostCurrency == nil) {
		return normalizedMachineValues{}, errors.New("estimatedMonthlyCostCents and costCurrency must be supplied together")
	}
	if body.EstimatedMonthlyCostCents != nil && *body.EstimatedMonthlyCostCents < 0 {
		return normalizedMachineValues{}, errors.New("estimatedMonthlyCostCents must not be negative")
	}
	if values.costCurrency.Valid && len(values.costCurrency.String) != 3 {
		return normalizedMachineValues{}, errors.New("costCurrency must contain three ASCII letters")
	}

	if body.AssetPublicId != nil {
		asset, err := s.queries.GetAssetByPublicID(ctx, *body.AssetPublicId)
		if err != nil {
			return normalizedMachineValues{}, err
		}
		product, err := s.queries.GetProductByPublicID(ctx, asset.ProductPublicID)
		if err != nil {
			return normalizedMachineValues{}, err
		}
		if product.Kind != string(api.System) {
			return normalizedMachineValues{}, errors.New("machine asset must have a system product")
		}
		if !asset.EffectiveAreaPublicID.Valid || asset.EffectiveAreaPublicID.String != body.AreaPublicId {
			return normalizedMachineValues{}, errors.New("machine asset must be placed in the machine area")
		}
		values.assetID = sql.NullString{String: asset.ID, Valid: true}
	}
	if body.ParentMachinePublicId != nil {
		parent, err := s.queries.GetMachineByPublicID(ctx, *body.ParentMachinePublicId)
		if err != nil {
			return normalizedMachineValues{}, err
		}
		if parent.ID == currentID {
			return normalizedMachineValues{}, errors.New("machine cannot host itself")
		}
		if parent.MachineProviderID != provider.ID || parent.AreaID != area.ID {
			return normalizedMachineValues{}, errors.New("local parent machine must share provider and area")
		}
		values.parentMachineID = sql.NullString{String: parent.ID, Valid: true}
	}
	if body.Kind == api.BareMetal {
		if values.parentMachineID.Valid || values.virtualizationPlatform.Valid {
			return normalizedMachineValues{}, errors.New("bare metal machines cannot have a parent or virtualizationPlatform")
		}
		if body.CpuAllocation != nil && *body.CpuAllocation != api.Dedicated {
			return normalizedMachineValues{}, errors.New("bare metal cpuAllocation must be dedicated")
		}
	} else if values.assetID.Valid {
		return normalizedMachineValues{}, errors.New("virtual machines cannot reference a physical asset")
	}
	return values, nil
}

func (v normalizedMachineValues) createParams(id, publicID, name, slug string) database.CreateMachineParams {
	return database.CreateMachineParams{
		ID: id, PublicID: publicID, AssetID: v.assetID, ParentMachineID: v.parentMachineID,
		MachineProviderID: v.machineProviderID, AreaID: v.areaID, Name: name, Slug: slug, Kind: v.kind,
		IsFavorite:             v.isFavorite,
		VirtualizationPlatform: v.virtualizationPlatform, Hostname: v.hostname, OsMachineID: v.osMachineID,
		OperatingSystem: v.operatingSystem, OperatingSystemVersion: v.operatingSystemVersion,
		Kernel: v.kernel, Architecture: v.architecture, CpuCount: v.cpuCount, CpuAllocation: v.cpuAllocation,
		CpuVendor: v.cpuVendor, MemoryBytes: v.memoryBytes, StorageBytes: v.storageBytes,
		StorageMediaKind: v.storageMediaKind, StorageInterfaceKind: v.storageInterfaceKind,
		EstimatedMonthlyCostCents: v.estimatedMonthlyCostCents, CostCurrency: v.costCurrency, Notes: v.notes,
	}
}

func (v normalizedMachineValues) updateParams(publicID, name, slug string) database.UpdateMachineParams {
	return database.UpdateMachineParams{
		AssetID: v.assetID, ParentMachineID: v.parentMachineID, MachineProviderID: v.machineProviderID,
		AreaID: v.areaID, Name: name, Slug: slug, Kind: v.kind, VirtualizationPlatform: v.virtualizationPlatform,
		IsFavorite: v.isFavorite,
		Hostname:   v.hostname, OsMachineID: v.osMachineID, OperatingSystem: v.operatingSystem,
		OperatingSystemVersion: v.operatingSystemVersion, Kernel: v.kernel, Architecture: v.architecture,
		CpuCount: v.cpuCount, CpuAllocation: v.cpuAllocation, CpuVendor: v.cpuVendor,
		MemoryBytes: v.memoryBytes, StorageBytes: v.storageBytes, StorageMediaKind: v.storageMediaKind,
		StorageInterfaceKind: v.storageInterfaceKind, EstimatedMonthlyCostCents: v.estimatedMonthlyCostCents,
		CostCurrency: v.costCurrency, Notes: v.notes, PublicID: publicID,
	}
}

type machineRecord struct {
	publicID, machineProviderPublicID, areaPublicID string
	assetPublicID, parentMachinePublicID            sql.NullString
	name, slug, kind                                string
	isFavorite                                      bool
	virtualizationPlatform, hostname, osMachineID   sql.NullString
	operatingSystem, operatingSystemVersion         sql.NullString
	kernel, architecture                            sql.NullString
	cpuCount                                        sql.NullInt64
	cpuAllocation, cpuVendor                        sql.NullString
	memoryBytes, storageBytes                       sql.NullInt64
	storageMediaKind, storageInterfaceKind          sql.NullString
	estimatedMonthlyCostCents                       sql.NullInt64
	costCurrency, notes                             sql.NullString
}

func machineListResponse(row database.ListMachineDetailsRow) api.Machine {
	return machineResponse(machineRecord{
		publicID: row.PublicID, machineProviderPublicID: row.MachineProviderPublicID, areaPublicID: row.AreaPublicID,
		assetPublicID: row.AssetPublicID, parentMachinePublicID: row.ParentMachinePublicID,
		name: row.Name, slug: row.Slug, kind: row.Kind, virtualizationPlatform: row.VirtualizationPlatform,
		isFavorite: row.IsFavorite,
		hostname:   row.Hostname, osMachineID: row.OsMachineID, operatingSystem: row.OperatingSystem,
		operatingSystemVersion: row.OperatingSystemVersion, kernel: row.Kernel, architecture: row.Architecture,
		cpuCount: row.CpuCount, cpuAllocation: row.CpuAllocation, cpuVendor: row.CpuVendor,
		memoryBytes: row.MemoryBytes, storageBytes: row.StorageBytes, storageMediaKind: row.StorageMediaKind,
		storageInterfaceKind: row.StorageInterfaceKind, estimatedMonthlyCostCents: row.EstimatedMonthlyCostCents,
		costCurrency: row.CostCurrency, notes: row.Notes,
	})
}

func machineResponse(row machineRecord) api.Machine {
	return api.Machine{
		PublicId: row.publicID, MachineProviderPublicId: row.machineProviderPublicID, AreaPublicId: row.areaPublicID,
		AssetPublicId: stringPointer(row.assetPublicID), ParentMachinePublicId: stringPointer(row.parentMachinePublicID),
		Name: row.name, Slug: row.slug, Kind: api.MachineKind(row.kind), IsFavorite: row.isFavorite,
		VirtualizationPlatform: stringPointer(row.virtualizationPlatform), Hostname: stringPointer(row.hostname),
		OsMachineId: stringPointer(row.osMachineID), OperatingSystem: stringPointer(row.operatingSystem),
		OperatingSystemVersion: stringPointer(row.operatingSystemVersion), Kernel: stringPointer(row.kernel),
		Architecture: stringPointer(row.architecture), CpuCount: intPointer(row.cpuCount),
		CpuAllocation: cpuAllocationPointer(row.cpuAllocation), CpuVendor: stringPointer(row.cpuVendor),
		MemoryBytes: int64Pointer(row.memoryBytes), StorageBytes: int64Pointer(row.storageBytes),
		StorageMediaKind:          storageMediaPointer(row.storageMediaKind),
		StorageInterfaceKind:      productStorageInterfacePointer(row.storageInterfaceKind),
		EstimatedMonthlyCostCents: int64Pointer(row.estimatedMonthlyCostCents),
		CostCurrency:              stringPointer(row.costCurrency), Notes: stringPointer(row.notes),
	}
}

func cpuAllocationPointer(value sql.NullString) *api.CpuAllocation {
	if !value.Valid {
		return nil
	}
	result := api.CpuAllocation(value.String)
	return &result
}

func storageMediaPointer(value sql.NullString) *api.StorageMediaKind {
	if !value.Valid {
		return nil
	}
	result := api.StorageMediaKind(value.String)
	return &result
}

func nullableLower(value *string) sql.NullString {
	result := nullableTrimmed(value)
	if result.Valid {
		result.String = strings.ToLower(result.String)
	}
	return result
}

func nullableUpper(value *string) sql.NullString {
	result := nullableTrimmed(value)
	if result.Valid {
		result.String = strings.ToUpper(result.String)
	}
	return result
}

func nullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}
