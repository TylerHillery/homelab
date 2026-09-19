package hlims

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type apiServer struct {
	db       *sql.DB
	queries  *database.Queries
	resolver resolver
}

func registerAPIHandlers(mux *http.ServeMux, server apiServer) {
	api.HandlerFromMuxWithBaseURL(server, mux, "/api/v1")
}

func (s apiServer) ListMachines(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListMachineDetails(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to list machines")
		return
	}

	items := make([]api.Machine, 0, len(rows))
	for _, row := range rows {
		items = append(items, machineListResponse(row))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Machine `json:"items"`
	}{Items: items})
}

func (s apiServer) ListMachineInstances(w http.ResponseWriter, r *http.Request, machine api.MachineSlug) {
	rows, err := s.queries.ListInstancesByMachineSlug(r.Context(), machine)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to list instances")
		return
	}

	items := make([]api.MachineInstance, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.MachineInstance{
			PublicId:        row.PublicID,
			Name:            row.Name,
			Slug:            row.Slug,
			Port:            int(row.Port),
			ServicePublicId: row.ServicePublicID,
			ServiceName:     row.ServiceName,
			ServiceSlug:     row.ServiceSlug,
		})
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.MachineInstance `json:"items"`
	}{Items: items})
}

func (s apiServer) ResolveInstance(
	w http.ResponseWriter,
	r *http.Request,
	machine api.MachineSlug,
	service api.ServiceSlug,
	instance api.InstanceSlug,
	params api.ResolveInstanceParams,
) {
	via := ""
	if params.Via != nil {
		via = string(*params.Via)
	}
	destination, err := s.resolver.instance(r.Context(), machine, service, instance, via, "", r.URL.Query())
	if err != nil {
		writeResolveError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.ResolvedDestination{
		Url: destination.URL,
		Via: api.Via(destination.Via),
	})
}

func (s apiServer) ResolveMachinePort(
	w http.ResponseWriter,
	r *http.Request,
	machine api.MachineSlug,
	port int,
	params api.ResolveMachinePortParams,
) {
	via := ""
	if params.Via != nil {
		via = string(*params.Via)
	}
	scheme := ""
	if params.Scheme != nil {
		scheme = string(*params.Scheme)
	}
	destination, err := s.resolver.machinePort(r.Context(), machine, port, via, scheme, "", r.URL.Query())
	if err != nil {
		writeResolveError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.ResolvedDestination{
		Url: destination.URL,
		Via: api.Via(destination.Via),
	})
}

func writeResolveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errDestinationNotFound):
		writeAPIError(w, http.StatusNotFound, "not_found", "destination not found")
	case errors.Is(err, errInvalidPort), errors.Is(err, errInvalidScheme), errors.Is(err, errInvalidVia):
		writeAPIError(w, http.StatusBadRequest, "invalid_destination", err.Error())
	default:
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to resolve destination")
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, api.Error{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
