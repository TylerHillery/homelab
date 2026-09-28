package hlims

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) ListDNSZones(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListDNSZones(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.DNSZone, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.DNSZone{PublicId: row.PublicID, Name: row.Name, Registrar: stringPointer(row.Registrar), Notes: stringPointer(row.Notes)})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.DNSZone `json:"items"`
	}{items})
}

func (s apiServer) CreateDNSZone(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.DNSZoneWrite](w, r)
	if !ok {
		return
	}
	name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(body.Name)), ".")
	if _, err := normalizeHostname(&name); err != nil || !strings.Contains(name, ".") {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "DNS zone must be a valid domain name")
		return
	}
	id, publicID, err := newIDs()
	if err == nil {
		_, err = s.queries.CreateDNSZone(r.Context(), database.CreateDNSZoneParams{
			ID: id, PublicID: publicID, Name: name,
			Registrar: nullableTrimmed(body.Registrar), Notes: nullableTrimmed(body.Notes),
		})
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	row, err := s.queries.GetDNSZoneByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, api.DNSZone{PublicId: row.PublicID, Name: row.Name, Registrar: stringPointer(row.Registrar), Notes: stringPointer(row.Notes)})
}

func dnsRecordResponse(publicID, zonePublicID, name, kind, addressPublicID, address, fqdn string) api.DNSRecord {
	return api.DNSRecord{PublicId: publicID, ZonePublicId: zonePublicID, Name: name, Kind: api.DNSRecordKind(kind), AddressPublicId: addressPublicID, Address: address, Fqdn: fqdn}
}

func (s apiServer) ListDNSRecords(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListDNSRecords(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.DNSRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, dnsRecordResponse(row.PublicID, row.ZonePublicID, row.Name, row.Kind, row.AddressPublicID, row.Address, row.Fqdn))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.DNSRecord `json:"items"`
	}{items})
}

func (s apiServer) CreateDNSRecord(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.DNSRecordWrite](w, r)
	if !ok {
		return
	}
	name := strings.ToLower(strings.TrimSpace(body.Name))
	if name != "@" {
		if _, err := normalizeHostname(&name); err != nil {
			writeInputError(w, err)
			return
		}
	}
	if body.Kind != api.DNSRecordKind("A") && body.Kind != api.DNSRecordKind("AAAA") {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "DNS record kind must be A or AAAA")
		return
	}
	zone, err := s.queries.GetDNSZoneByPublicID(r.Context(), body.ZonePublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	address, err := s.queries.GetAddressByPublicID(r.Context(), body.AddressPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	ip, err := netip.ParseAddr(address.Address)
	if err != nil || (body.Kind == "A" && !ip.Is4()) || (body.Kind == "AAAA" && !ip.Is6()) {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "DNS record kind must match the address IP family")
		return
	}
	id, publicID, err := newIDs()
	if err == nil {
		_, err = s.queries.CreateDNSRecord(r.Context(), database.CreateDNSRecordParams{
			ID: id, PublicID: publicID, ZoneID: zone.ID, Name: name, Kind: string(body.Kind), AddressID: address.ID,
		})
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	row, err := s.queries.GetDNSRecordByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dnsRecordResponse(row.PublicID, row.ZonePublicID, row.Name, row.Kind, row.AddressPublicID, row.Address, row.Fqdn))
}

func ingressRouteResponse(endpointID, instanceID, service, kind, target string) api.IngressRoute {
	return api.IngressRoute{EndpointPublicId: endpointID, IngressInstancePublicId: instanceID, IngressServiceName: service, Kind: api.IngressRouteKind(kind), Target: target}
}

func (s apiServer) ListIngressRoutes(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListIngressRoutes(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.IngressRoute, 0, len(rows))
	for _, row := range rows {
		items = append(items, ingressRouteResponse(row.EndpointPublicID, row.IngressInstancePublicID, row.IngressServiceName, row.Kind, row.Target))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.IngressRoute `json:"items"`
	}{items})
}

func (s apiServer) CreateIngressRoute(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.IngressRouteWrite](w, r)
	if !ok {
		return
	}
	if (body.Kind != "proxy" && body.Kind != "static" && body.Kind != "redirect") || strings.TrimSpace(body.Target) == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "ingress kind must be proxy, static, or redirect and target must not be empty")
		return
	}
	endpoint, err := s.queries.GetEndpointIngressIDs(r.Context(), body.EndpointPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	ingress, err := s.queries.GetInstanceByPublicID(r.Context(), body.IngressInstancePublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	if endpoint.MachineID != ingress.MachineID {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "ingress and served instance must share a machine")
		return
	}
	_, err = s.queries.CreateIngressRoute(r.Context(), database.CreateIngressRouteParams{
		EndpointID: endpoint.ID, IngressInstanceID: ingress.ID, Kind: string(body.Kind), Target: strings.TrimSpace(body.Target),
	})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	row, err := s.queries.GetIngressRouteByEndpointPublicID(r.Context(), body.EndpointPublicId)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ingressRouteResponse(row.EndpointPublicID, row.IngressInstancePublicID, row.IngressServiceName, row.Kind, row.Target))
}
