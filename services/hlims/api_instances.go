package hlims

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) ListInstances(w http.ResponseWriter, r *http.Request, params api.ListInstancesParams) {
	rows, err := s.queries.ListInstances(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Instance, 0, len(rows))
	var serviceIDs map[string]string
	if params.Service != nil {
		services, err := s.queries.ListServices(r.Context())
		if err != nil {
			writeDatabaseError(w, err)
			return
		}
		serviceIDs = make(map[string]string, len(services))
		for _, service := range services {
			serviceIDs[service.PublicID] = service.Slug
		}
	}
	var machineIDs map[string]string
	if params.Machine != nil {
		machines, err := s.queries.ListMachineDetails(r.Context())
		if err != nil {
			writeDatabaseError(w, err)
			return
		}
		machineIDs = make(map[string]string, len(machines))
		for _, machine := range machines {
			machineIDs[machine.PublicID] = machine.Slug
		}
	}
	for _, row := range rows {
		if params.Service != nil && serviceIDs[row.ServicePublicID] != *params.Service ||
			params.Machine != nil && machineIDs[row.MachinePublicID.String] != *params.Machine ||
			params.Slug != nil && row.Slug != *params.Slug ||
			params.HostingKind != nil && row.HostingKind != string(*params.HostingKind) {
			continue
		}
		items = append(items, instanceResponse(row.PublicID, row.ServicePublicID, row.MachinePublicID, row.HostingKind, row.ManagedProvider, row.Name, row.Slug, row.Port, row.Notes))
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
	service, machineID, kind, provider, name, slug, port, err := s.instanceWriteValues(r, body, "")
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateInstance(r.Context(), database.CreateInstanceParams{ID: id, PublicID: publicID, ServiceID: service.ID, MachineID: machineID, HostingKind: kind, ManagedProvider: provider, Name: name, Slug: slug, Port: port, Notes: nullString(body.Notes, false)})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeInstance(w, r, publicID, http.StatusCreated)
}

func (s apiServer) instanceWriteValues(r *http.Request, body api.InstanceWrite, existingSlug string) (database.Service, sql.NullString, string, sql.NullString, string, string, sql.NullInt64, error) {
	service, err := s.queries.GetServiceByPublicID(r.Context(), body.ServicePublicId)
	if err != nil {
		return database.Service{}, sql.NullString{}, "", sql.NullString{}, "", "", sql.NullInt64{}, err
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existingSlug)
	if err != nil {
		return service, sql.NullString{}, "", sql.NullString{}, "", "", sql.NullInt64{}, err
	}
	kind := "machine"
	if body.HostingKind != nil {
		kind = string(*body.HostingKind)
	}
	switch kind {
	case "machine":
		if body.MachinePublicId == nil || body.ManagedProvider != nil {
			return service, sql.NullString{}, "", sql.NullString{}, "", "", sql.NullInt64{}, errors.New("machine instance requires machinePublicId, without managedProvider")
		}
		if body.Port != nil && (*body.Port < 1 || *body.Port > 65535) {
			return service, sql.NullString{}, "", sql.NullString{}, "", "", sql.NullInt64{}, errInvalidPort
		}
		machine, err := s.queries.GetMachineByPublicID(r.Context(), *body.MachinePublicId)
		return service, sql.NullString{String: machine.ID, Valid: err == nil}, kind, sql.NullString{}, name, slug, nullInt(body.Port), err
	case "managed":
		if body.MachinePublicId != nil || body.Port != nil || body.ManagedProvider == nil || strings.TrimSpace(*body.ManagedProvider) == "" {
			return service, sql.NullString{}, "", sql.NullString{}, "", "", sql.NullInt64{}, errors.New("managed instance requires managedProvider, without machinePublicId or port")
		}
		return service, sql.NullString{}, kind, nullableTrimmed(body.ManagedProvider), name, slug, sql.NullInt64{}, nil
	default:
		return service, sql.NullString{}, "", sql.NullString{}, "", "", sql.NullInt64{}, errors.New("hostingKind must be machine or managed")
	}
}

func instanceResponse(publicID, serviceID string, machineID sql.NullString, kind string, provider sql.NullString, name, slug string, port sql.NullInt64, notes sql.NullString) api.Instance {
	hostingKind := api.InstanceHostingKind(kind)
	return api.Instance{PublicId: publicID, ServicePublicId: serviceID, MachinePublicId: stringPointer(machineID), HostingKind: &hostingKind, ManagedProvider: stringPointer(provider), Name: name, Slug: slug, Port: intPointer(port), Notes: stringPointer(notes)}
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
	writeJSON(w, status, instanceResponse(row.PublicID, row.ServicePublicID, row.MachinePublicID, row.HostingKind, row.ManagedProvider, row.Name, row.Slug, row.Port, row.Notes))
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
	service, machineID, kind, provider, name, slug, port, err := s.instanceWriteValues(r, body, existing.Slug)
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	_, err = s.queries.UpdateInstance(r.Context(), database.UpdateInstanceParams{ServiceID: service.ID, MachineID: machineID, HostingKind: kind, ManagedProvider: provider, Name: name, Slug: slug, Port: port, Notes: nullString(body.Notes, false), PublicID: publicID})
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

func (s apiServer) ListInstanceEndpoints(w http.ResponseWriter, r *http.Request, params api.ListInstanceEndpointsParams) {
	rows, err := s.queries.ListInstanceEndpoints(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.InstanceEndpoint, 0, len(rows))
	for _, row := range rows {
		if params.InstancePublicId != nil && row.InstancePublicID != *params.InstancePublicId {
			continue
		}
		items = append(items, endpointResponse(row.PublicID, row.InstancePublicID, row.AddressPublicID, row.DirectUrl, row.Name, row.Scheme, row.Port, row.BasePath, row.HostType, row.IsPreferred, row.Notes, row.DnsRecordPublicID))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.InstanceEndpoint `json:"items"`
	}{items})
}

type endpointWriteValues struct {
	instance    database.Instance
	addressID   sql.NullString
	directURL   sql.NullString
	dnsRecordID sql.NullString
	name        string
	scheme      sql.NullString
	port        sql.NullInt64
	basePath    string
	hostType    string
	preferred   int64
	notes       sql.NullString
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
	_, err = queries.CreateInstanceEndpoint(r.Context(), database.CreateInstanceEndpointParams{ID: id, PublicID: publicID, InstanceID: values.instance.ID, AddressID: values.addressID, DirectUrl: values.directURL, DnsRecordID: values.dnsRecordID, Name: values.name, Scheme: values.scheme, Port: values.port, BasePath: values.basePath, HostType: values.hostType, IsPreferred: values.preferred, Notes: values.notes})
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
	name, err := normalizeName(body.Name)
	if err != nil {
		return endpointWriteValues{}, err
	}
	basePath, err := normalizeBasePath(body.BasePath)
	if err != nil {
		return endpointWriteValues{}, err
	}
	values := endpointWriteValues{instance: instance, name: name, basePath: basePath, hostType: "auto", preferred: boolInt(body.IsPreferred), notes: nullString(body.Notes, false)}
	if instance.HostingKind == "managed" {
		if body.DirectUrl == nil || body.AddressPublicId != nil || body.DnsRecordPublicId != nil || body.Scheme != nil || body.Port != nil || body.HostType != nil || basePath != "" {
			return endpointWriteValues{}, errors.New("managed endpoint requires only directUrl, without address, scheme, port, or basePath")
		}
		direct, err := url.Parse(strings.TrimSpace(*body.DirectUrl))
		if err != nil || direct.Scheme != "https" || direct.Hostname() == "" || direct.User != nil || direct.Fragment != "" {
			return endpointWriteValues{}, errors.New("directUrl must be an absolute HTTPS URL without credentials or a fragment")
		}
		values.directURL = sql.NullString{String: direct.String(), Valid: true}
		return values, nil
	}
	if body.DirectUrl != nil || body.AddressPublicId == nil || body.Scheme == nil || body.Port == nil {
		return endpointWriteValues{}, errors.New("machine endpoint requires addressPublicId, scheme, and port, without directUrl")
	}
	address, err := s.queries.GetAddressByPublicID(r.Context(), *body.AddressPublicId)
	if err != nil {
		return endpointWriteValues{}, err
	}
	if !address.MachineID.Valid || address.MachineID.String != instance.MachineID.String {
		return endpointWriteValues{}, errors.New("endpoint address must belong to the instance machine")
	}
	scheme := strings.ToLower(strings.TrimSpace(string(*body.Scheme)))
	if !InstanceScheme(scheme).Valid() {
		return endpointWriteValues{}, errInvalidScheme
	}
	if *body.Port < 1 || *body.Port > 65535 {
		return endpointWriteValues{}, errInvalidPort
	}
	hostType := "auto"
	if body.HostType != nil {
		hostType = string(*body.HostType)
	}
	var dnsRecordID sql.NullString
	if body.DnsRecordPublicId != nil {
		record, err := s.queries.GetDNSRecordByPublicID(r.Context(), *body.DnsRecordPublicId)
		if err != nil {
			return endpointWriteValues{}, err
		}
		if record.AddressPublicID != *body.AddressPublicId {
			return endpointWriteValues{}, errors.New("DNS record must point to the endpoint address")
		}
		dnsRecordID = sql.NullString{String: record.ID, Valid: true}
		if body.HostType == nil {
			hostType = "dns"
		}
	}
	if hostType != "auto" && hostType != "dns" && hostType != "ip" {
		return endpointWriteValues{}, errors.New("hostType must be auto, dns, or ip")
	}
	if dnsRecordID.Valid && hostType != "dns" {
		return endpointWriteValues{}, errors.New("an endpoint with a DNS record must use the DNS host type")
	}
	if hostType == "dns" && !address.DnsName.Valid && !dnsRecordID.Valid {
		return endpointWriteValues{}, errors.New("DNS endpoint requires an address with a DNS name")
	}
	values.addressID = sql.NullString{String: address.ID, Valid: true}
	values.dnsRecordID = dnsRecordID
	values.scheme = sql.NullString{String: scheme, Valid: true}
	values.port = sql.NullInt64{Int64: int64(*body.Port), Valid: true}
	values.hostType = hostType
	return values, nil
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
	writeJSON(w, status, endpointResponse(row.PublicID, row.InstancePublicID, row.AddressPublicID, row.DirectUrl, row.Name, row.Scheme, row.Port, row.BasePath, row.HostType, row.IsPreferred, row.Notes, row.DnsRecordPublicID))
}

func endpointResponse(publicID, instancePublicID string, addressPublicID, directURL sql.NullString, name string, scheme sql.NullString, port sql.NullInt64, basePath, hostType string, preferred int64, notes, recordID sql.NullString) api.InstanceEndpoint {
	mode := api.EndpointHostType(hostType)
	response := api.InstanceEndpoint{PublicId: publicID, InstancePublicId: instancePublicID, AddressPublicId: stringPointer(addressPublicID), DirectUrl: stringPointer(directURL), DnsRecordPublicId: stringPointer(recordID), Name: name, Port: intPointer(port), BasePath: basePath, HostType: &mode, IsPreferred: preferred == 1, Notes: stringPointer(notes)}
	if scheme.Valid {
		value := api.Scheme(scheme.String)
		response.Scheme = &value
	}
	return response
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
	_, err = queries.UpdateInstanceEndpoint(r.Context(), database.UpdateInstanceEndpointParams{InstanceID: values.instance.ID, AddressID: values.addressID, DirectUrl: values.directURL, DnsRecordID: values.dnsRecordID, Name: values.name, Scheme: values.scheme, Port: values.port, BasePath: values.basePath, HostType: values.hostType, IsPreferred: values.preferred, Notes: values.notes, PublicID: publicID})
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
