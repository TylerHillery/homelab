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

func newIDs() (string, string, error) {
	id, err := NewInternalID()
	if err != nil {
		return "", "", err
	}
	publicID, err := NewPublicID()
	return string(id), string(publicID), err
}

func readWriteBody[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	if err := decodeJSON(r.Header.Get("Content-Type"), r.Body, &body); err != nil {
		if errors.Is(err, errUnsupportedMediaType) {
			writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", err.Error())
		} else {
			writeAPIError(w, http.StatusBadRequest, "invalid_input", err.Error())
		}
		return body, false
	}
	return body, true
}

func writeInputError(w http.ResponseWriter, err error) {
	writeAPIError(w, http.StatusBadRequest, "invalid_input", err.Error())
}

func writeDatabaseError(w http.ResponseWriter, err error) {
	message := strings.ToLower(err.Error())
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeAPIError(w, http.StatusNotFound, "not_found", "resource not found")
	case strings.Contains(message, "must belong"):
		writeAPIError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case strings.Contains(message, "constraint failed") || strings.Contains(message, "foreign key"):
		writeAPIError(w, http.StatusConflict, "conflict", "resource conflicts with existing data")
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "database operation failed")
	}
}

func deleteResource(w http.ResponseWriter, rows int64, err error) {
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	if rows == 0 {
		writeAPIError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s apiServer) ListMachineProviders(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListMachineProviders(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.MachineProvider, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.MachineProvider{PublicId: row.PublicID, Name: row.Name, Slug: row.Slug, HasLogo: row.HasLogo})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.MachineProvider `json:"items"`
	}{items})
}

func (s apiServer) CreateMachineProvider(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.MachineProviderWrite](w, r)
	if !ok {
		return
	}
	name, err := normalizeName(body.Name)
	if err != nil {
		writeInputError(w, err)
		return
	}
	slug, err := writeSlug(name, body.Slug, "")
	if err != nil {
		writeInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	row, err := s.queries.CreateMachineProvider(r.Context(), database.CreateMachineProviderParams{ID: id, PublicID: publicID, Name: name, Slug: slug})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, machineProviderResponse(row, false))
}

func (s apiServer) GetMachineProvider(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	row, err := s.queries.GetMachineProviderByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	hasLogo, err := s.queries.MachineProviderHasLogo(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, machineProviderResponse(row, hasLogo))
}

func (s apiServer) UpdateMachineProvider(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.MachineProviderWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetMachineProviderByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, err := normalizeName(body.Name)
	if err != nil {
		writeInputError(w, err)
		return
	}
	slug, err := writeSlug(name, body.Slug, existing.Slug)
	if err != nil {
		writeInputError(w, err)
		return
	}
	row, err := s.queries.UpdateMachineProvider(r.Context(), database.UpdateMachineProviderParams{Name: name, Slug: slug, PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	hasLogo, err := s.queries.MachineProviderHasLogo(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, machineProviderResponse(row, hasLogo))
}

func (s apiServer) DeleteMachineProvider(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteMachineProvider(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func machineProviderResponse(row database.MachineProvider, hasLogo bool) api.MachineProvider {
	return api.MachineProvider{PublicId: row.PublicID, Name: row.Name, Slug: row.Slug, HasLogo: hasLogo}
}

func (s apiServer) ListManufacturers(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListManufacturers(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Manufacturer, 0, len(rows))
	for _, row := range rows {
		items = append(items, manufacturerResponse(row))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Manufacturer `json:"items"`
	}{items})
}

func (s apiServer) CreateManufacturer(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.ManufacturerWrite](w, r)
	if !ok {
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
	row, err := s.queries.CreateManufacturer(r.Context(), database.CreateManufacturerParams{ID: id, PublicID: publicID, Name: name, Slug: slug})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, manufacturerResponse(row))
}

func (s apiServer) GetManufacturer(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	row, err := s.queries.GetManufacturerByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, manufacturerResponse(row))
}

func (s apiServer) UpdateManufacturer(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.ManufacturerWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetManufacturerByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existing.Slug)
	if err != nil {
		writeInputError(w, err)
		return
	}
	row, err := s.queries.UpdateManufacturer(r.Context(), database.UpdateManufacturerParams{Name: name, Slug: slug, PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, manufacturerResponse(row))
}

func (s apiServer) DeleteManufacturer(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteManufacturer(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func manufacturerResponse(row database.Manufacturer) api.Manufacturer {
	return api.Manufacturer{PublicId: row.PublicID, Name: row.Name, Slug: row.Slug}
}

func normalizedNamedWrite(name string, supplied *string, existing string) (string, string, error) {
	name, err := normalizeName(name)
	if err != nil {
		return "", "", err
	}
	slug, err := writeSlug(name, supplied, existing)
	return name, slug, err
}

func (s apiServer) ListServices(w http.ResponseWriter, r *http.Request, params api.ListServicesParams) {
	rows, err := s.queries.ListServices(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Service, 0, len(rows))
	for _, row := range rows {
		if params.Slug != nil && row.Slug != *params.Slug {
			continue
		}
		items = append(items, api.Service{PublicId: row.PublicID, Name: row.Name, Slug: row.Slug, Description: stringPointer(row.Description), HasLogo: row.HasLogo})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Service `json:"items"`
	}{items})
}

func (s apiServer) CreateService(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.ServiceWrite](w, r)
	if !ok {
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
	row, err := s.queries.CreateService(r.Context(), database.CreateServiceParams{ID: id, PublicID: publicID, Name: name, Slug: slug, Description: nullString(body.Description, false)})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, serviceResponse(row, false))
}

func (s apiServer) GetService(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	row, err := s.queries.GetServiceByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	hasLogo, err := s.queries.ServiceHasLogo(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, serviceResponse(row, hasLogo))
}

func (s apiServer) UpdateService(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.ServiceWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetServiceByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existing.Slug)
	if err != nil {
		writeInputError(w, err)
		return
	}
	row, err := s.queries.UpdateService(r.Context(), database.UpdateServiceParams{Name: name, Slug: slug, Description: nullString(body.Description, false), PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	hasLogo, err := s.queries.ServiceHasLogo(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, serviceResponse(row, hasLogo))
}

func (s apiServer) DeleteService(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteService(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func serviceResponse(row database.Service, hasLogo bool) api.Service {
	return api.Service{PublicId: row.PublicID, Name: row.Name, Slug: row.Slug, Description: stringPointer(row.Description), HasLogo: hasLogo}
}

func lookupAreaAndProvider(ctx context.Context, queries *database.Queries, areaPublicID, providerPublicID string) (database.Area, database.MachineProvider, error) {
	provider, err := queries.GetMachineProviderByPublicID(ctx, providerPublicID)
	if err != nil {
		return database.Area{}, database.MachineProvider{}, err
	}
	area, err := queries.GetAreaByPublicID(ctx, areaPublicID)
	return area, provider, err
}
