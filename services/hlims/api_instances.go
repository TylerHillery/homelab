package hlims

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) ListInstances(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListInstances(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Instance, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.Instance{PublicId: row.PublicID, ServicePublicId: row.ServicePublicID, MachinePublicId: row.MachinePublicID, Name: row.Name, Slug: row.Slug, Port: int(row.Port), Notes: stringPointer(row.Notes)})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Instance `json:"items"`
	}{items})
}

func (s apiServer) CreateInstance(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.InstanceWrite](w, r)
	if !ok {
		return
	}
	service, machine, name, slug, err := s.instanceWriteValues(r, body, "")
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateInstance(r.Context(), database.CreateInstanceParams{ID: id, PublicID: publicID, ServiceID: service.ID, MachineID: machine.ID, Name: name, Slug: slug, Port: int64(body.Port), Notes: nullString(body.Notes, false)})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeInstance(w, r, publicID, http.StatusCreated)
}

func (s apiServer) instanceWriteValues(r *http.Request, body api.InstanceWrite, existingSlug string) (database.Service, database.GetMachineByPublicIDRow, string, string, error) {
	service, err := s.queries.GetServiceByPublicID(r.Context(), body.ServicePublicId)
	if err != nil {
		return database.Service{}, database.GetMachineByPublicIDRow{}, "", "", err
	}
	machine, err := s.queries.GetMachineByPublicID(r.Context(), body.MachinePublicId)
	if err != nil {
		return database.Service{}, database.GetMachineByPublicIDRow{}, "", "", err
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existingSlug)
	if err == nil && (body.Port < 1 || body.Port > 65535) {
		err = errors.New("port must be between 1 and 65535")
	}
	return service, machine, name, slug, err
}

func (s apiServer) writeParentOrInputError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeDatabaseError(w, err)
	} else {
		writeInputError(w, err)
	}
}

func (s apiServer) GetInstance(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeInstance(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeInstance(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetInstanceDetailByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, api.Instance{PublicId: row.PublicID, ServicePublicId: row.ServicePublicID, MachinePublicId: row.MachinePublicID, Name: row.Name, Slug: row.Slug, Port: int(row.Port), Notes: stringPointer(row.Notes)})
}

func (s apiServer) UpdateInstance(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.InstanceWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetInstanceByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	service, machine, name, slug, err := s.instanceWriteValues(r, body, existing.Slug)
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	_, err = s.queries.UpdateInstance(r.Context(), database.UpdateInstanceParams{ServiceID: service.ID, MachineID: machine.ID, Name: name, Slug: slug, Port: int64(body.Port), Notes: nullString(body.Notes, false), PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeInstance(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteInstance(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteInstance(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func (s apiServer) ListInstanceEndpoints(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListInstanceEndpoints(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.InstanceEndpoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, endpointResponse(row.PublicID, row.InstancePublicID, row.AddressPublicID, row.Name, row.Scheme, row.Port, row.BasePath, row.IsPreferred, row.Notes))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.InstanceEndpoint `json:"items"`
	}{items})
}

type endpointWriteValues struct {
	instance  database.Instance
	address   database.GetAddressByPublicIDRow
	name      string
	scheme    string
	port      int64
	basePath  string
	preferred int64
	notes     sql.NullString
}

func (s apiServer) CreateInstanceEndpoint(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.InstanceEndpointWrite](w, r)
	if !ok {
		return
	}
	values, err := s.endpointWriteValues(r, body)
	if err != nil {
		s.writeEndpointInputError(w, err)
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
	if values.preferred == 1 {
		if err := queries.ClearAllPreferredInstanceEndpoints(r.Context(), values.instance.ID); err != nil {
			writeDatabaseError(w, err)
			return
		}
	}
	_, err = queries.CreateInstanceEndpoint(r.Context(), database.CreateInstanceEndpointParams{ID: id, PublicID: publicID, InstanceID: values.instance.ID, AddressID: values.address.ID, Name: values.name, Scheme: values.scheme, Port: values.port, BasePath: values.basePath, IsPreferred: values.preferred, Notes: values.notes})
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeEndpoint(w, r, publicID, http.StatusCreated)
}

func (s apiServer) endpointWriteValues(r *http.Request, body api.InstanceEndpointWrite) (endpointWriteValues, error) {
	instance, err := s.queries.GetInstanceByPublicID(r.Context(), body.InstancePublicId)
	if err != nil {
		return endpointWriteValues{}, err
	}
	address, err := s.queries.GetAddressByPublicID(r.Context(), body.AddressPublicId)
	if err != nil {
		return endpointWriteValues{}, err
	}
	if !address.MachineID.Valid || address.MachineID.String != instance.MachineID {
		return endpointWriteValues{}, errors.New("endpoint address must belong to the instance machine")
	}
	name, err := normalizeName(body.Name)
	if err != nil {
		return endpointWriteValues{}, err
	}
	scheme := strings.ToLower(strings.TrimSpace(string(body.Scheme)))
	if !InstanceScheme(scheme).Valid() {
		return endpointWriteValues{}, errInvalidScheme
	}
	if body.Port < 1 || body.Port > 65535 {
		return endpointWriteValues{}, errInvalidPort
	}
	basePath, err := normalizeBasePath(body.BasePath)
	if err != nil {
		return endpointWriteValues{}, err
	}
	return endpointWriteValues{instance: instance, address: address, name: name, scheme: scheme, port: int64(body.Port), basePath: basePath, preferred: boolInt(body.IsPreferred), notes: nullString(body.Notes, false)}, nil
}

func (s apiServer) writeEndpointInputError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeDatabaseError(w, err)
		return
	}
	writeInputError(w, err)
}

func (s apiServer) GetInstanceEndpoint(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeEndpoint(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeEndpoint(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetInstanceEndpointByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, endpointResponse(row.PublicID, row.InstancePublicID, row.AddressPublicID, row.Name, row.Scheme, row.Port, row.BasePath, row.IsPreferred, row.Notes))
}

func endpointResponse(publicID, instancePublicID, addressPublicID, name, scheme string, port int64, basePath string, preferred int64, notes sql.NullString) api.InstanceEndpoint {
	return api.InstanceEndpoint{PublicId: publicID, InstancePublicId: instancePublicID, AddressPublicId: addressPublicID, Name: name, Scheme: api.Scheme(scheme), Port: int(port), BasePath: basePath, IsPreferred: preferred == 1, Notes: stringPointer(notes)}
}

func (s apiServer) UpdateInstanceEndpoint(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.InstanceEndpointWrite](w, r)
	if !ok {
		return
	}
	if _, err := s.queries.GetInstanceEndpointByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	values, err := s.endpointWriteValues(r, body)
	if err != nil {
		s.writeEndpointInputError(w, err)
		return
	}
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	if values.preferred == 1 {
		if err := queries.ClearPreferredInstanceEndpoints(r.Context(), database.ClearPreferredInstanceEndpointsParams{InstanceID: values.instance.ID, PublicID: publicID}); err != nil {
			writeDatabaseError(w, err)
			return
		}
	}
	_, err = queries.UpdateInstanceEndpoint(r.Context(), database.UpdateInstanceEndpointParams{InstanceID: values.instance.ID, AddressID: values.address.ID, Name: values.name, Scheme: values.scheme, Port: values.port, BasePath: values.basePath, IsPreferred: values.preferred, Notes: values.notes, PublicID: publicID})
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeEndpoint(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteInstanceEndpoint(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteInstanceEndpoint(r.Context(), publicID)
	deleteResource(w, rows, err)
}
