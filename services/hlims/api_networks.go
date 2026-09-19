package hlims

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) ListNetworks(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListNetworks(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Network, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.Network{PublicId: row.PublicID, AreaPublicId: stringPointer(row.AreaPublicID), Name: row.Name, Slug: row.Slug, Kind: api.NetworkKind(row.Kind), Cidr: stringPointer(row.Cidr), Notes: stringPointer(row.Notes)})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Network `json:"items"`
	}{items})
}

func (s apiServer) CreateNetwork(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.NetworkWrite](w, r)
	if !ok {
		return
	}
	areaID, err := s.networkAreaID(r, body.AreaPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, "")
	if err != nil {
		writeInputError(w, err)
		return
	}
	if !body.Kind.Valid() {
		writeInputError(w, errors.New("invalid network kind"))
		return
	}
	cidr, err := normalizeCIDR(body.Cidr)
	if err != nil {
		writeInputError(w, err)
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateNetwork(r.Context(), database.CreateNetworkParams{ID: id, PublicID: publicID, AreaID: areaID, Name: name, Slug: slug, Kind: string(body.Kind), Cidr: cidr, Notes: nullString(body.Notes, false)})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeNetwork(w, r, publicID, http.StatusCreated)
}

func (s apiServer) networkAreaID(r *http.Request, publicID *string) (sql.NullString, error) {
	if publicID == nil {
		return sql.NullString{}, nil
	}
	area, err := s.queries.GetAreaByPublicID(r.Context(), *publicID)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: area.ID, Valid: true}, nil
}

func (s apiServer) GetNetwork(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeNetwork(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeNetwork(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetNetworkByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, api.Network{PublicId: row.PublicID, AreaPublicId: stringPointer(row.AreaPublicID), Name: row.Name, Slug: row.Slug, Kind: api.NetworkKind(row.Kind), Cidr: stringPointer(row.Cidr), Notes: stringPointer(row.Notes)})
}

func (s apiServer) UpdateNetwork(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.NetworkWrite](w, r)
	if !ok {
		return
	}
	existing, err := s.queries.GetNetworkByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	areaID, err := s.networkAreaID(r, body.AreaPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	name, slug, err := normalizedNamedWrite(body.Name, body.Slug, existing.Slug)
	if err != nil {
		writeInputError(w, err)
		return
	}
	if !body.Kind.Valid() {
		writeInputError(w, errors.New("invalid network kind"))
		return
	}
	cidr, err := normalizeCIDR(body.Cidr)
	if err != nil {
		writeInputError(w, err)
		return
	}
	_, err = s.queries.UpdateNetwork(r.Context(), database.UpdateNetworkParams{AreaID: areaID, Name: name, Slug: slug, Kind: string(body.Kind), Cidr: cidr, Notes: nullString(body.Notes, false), PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeNetwork(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteNetwork(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteNetwork(r.Context(), publicID)
	deleteResource(w, rows, err)
}

func (s apiServer) ListAddresses(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListAddresses(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.Address, 0, len(rows))
	for _, row := range rows {
		items = append(items, addressResponse(row.PublicID, row.NetworkPublicID, row.MachinePublicID, row.AreaPublicID, row.Name, row.Address, row.DnsName, row.InterfaceName, row.IsPrimary))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Address `json:"items"`
	}{items})
}

func (s apiServer) CreateAddress(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.AddressWrite](w, r)
	if !ok {
		return
	}
	params, err := s.addressParams(r, body)
	if err != nil {
		var inputErr inputValidationError
		if errors.Is(err, errInvalidAddressOwner) || errors.As(err, &inputErr) {
			writeInputError(w, err)
		} else {
			writeDatabaseError(w, err)
		}
		return
	}
	id, publicID, err := newIDs()
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	_, err = s.queries.CreateAddress(r.Context(), database.CreateAddressParams{ID: id, PublicID: publicID, NetworkID: params.networkID, MachineID: params.machineID, AreaID: params.areaID, Name: params.name, Address: params.address, DnsName: params.dnsName, InterfaceName: params.interfaceName, IsPrimary: params.isPrimary})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeAddress(w, r, publicID, http.StatusCreated)
}

var errInvalidAddressOwner = errors.New("address must reference exactly one machine or area")

type normalizedAddress struct {
	networkID, address           string
	machineID, areaID            sql.NullString
	name, dnsName, interfaceName sql.NullString
	isPrimary                    int64
}

func (s apiServer) addressParams(r *http.Request, body api.AddressWrite) (normalizedAddress, error) {
	if (body.MachinePublicId == nil) == (body.AreaPublicId == nil) {
		return normalizedAddress{}, errInvalidAddressOwner
	}
	network, err := s.queries.GetNetworkByPublicID(r.Context(), body.NetworkPublicId)
	if err != nil {
		return normalizedAddress{}, err
	}
	result := normalizedAddress{networkID: network.ID, name: nullString(body.Name, true), interfaceName: nullString(body.InterfaceName, true), isPrimary: boolInt(body.IsPrimary)}
	if body.MachinePublicId != nil {
		machine, err := s.queries.GetMachineByPublicID(r.Context(), *body.MachinePublicId)
		if err != nil {
			return normalizedAddress{}, err
		}
		result.machineID = sql.NullString{String: machine.ID, Valid: true}
	} else {
		area, err := s.queries.GetAreaByPublicID(r.Context(), *body.AreaPublicId)
		if err != nil {
			return normalizedAddress{}, err
		}
		result.areaID = sql.NullString{String: area.ID, Valid: true}
	}
	result.address, err = normalizeIP(body.Address)
	if err != nil {
		return normalizedAddress{}, errInvalidAddressOwnerWrap(err)
	}
	result.dnsName, err = normalizeHostname(body.DnsName)
	if err != nil {
		return normalizedAddress{}, errInvalidAddressOwnerWrap(err)
	}
	return result, nil
}

func errInvalidAddressOwnerWrap(err error) error { return inputValidationError{err} }

type inputValidationError struct{ error }

func (s apiServer) GetAddress(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeAddress(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeAddress(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetAddressByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, addressResponse(row.PublicID, row.NetworkPublicID, row.MachinePublicID, row.AreaPublicID, row.Name, row.Address, row.DnsName, row.InterfaceName, row.IsPrimary))
}

func addressResponse(publicID, networkPublicID string, machinePublicID, areaPublicID, name sql.NullString, address string, dnsName, interfaceName sql.NullString, isPrimary int64) api.Address {
	return api.Address{PublicId: publicID, NetworkPublicId: networkPublicID, MachinePublicId: stringPointer(machinePublicID), AreaPublicId: stringPointer(areaPublicID), Name: stringPointer(name), Address: address, DnsName: stringPointer(dnsName), InterfaceName: stringPointer(interfaceName), IsPrimary: isPrimary == 1}
}

func (s apiServer) UpdateAddress(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.AddressWrite](w, r)
	if !ok {
		return
	}
	if _, err := s.queries.GetAddressByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	params, err := s.addressParams(r, body)
	if err != nil {
		var inputErr inputValidationError
		if errors.Is(err, errInvalidAddressOwner) || errors.As(err, &inputErr) {
			writeInputError(w, err)
		} else {
			writeDatabaseError(w, err)
		}
		return
	}
	_, err = s.queries.UpdateAddress(r.Context(), database.UpdateAddressParams{NetworkID: params.networkID, MachineID: params.machineID, AreaID: params.areaID, Name: params.name, Address: params.address, DnsName: params.dnsName, InterfaceName: params.interfaceName, IsPrimary: params.isPrimary, PublicID: publicID})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeAddress(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteAddress(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteAddress(r.Context(), publicID)
	deleteResource(w, rows, err)
}
