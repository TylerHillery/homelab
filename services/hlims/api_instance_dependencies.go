package hlims

import (
	"errors"
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func dependencyResponse(id, consumer, provider, kind string, notes *string) api.InstanceDependency {
	return api.InstanceDependency{PublicId: id, ConsumerInstancePublicId: consumer, ProviderInstancePublicId: provider, Kind: &kind, Notes: notes}
}

func (s apiServer) ListInstanceDependencies(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queries.ListInstanceDependencies(r.Context())
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	items := make([]api.InstanceDependency, 0, len(rows))
	for _, row := range rows {
		items = append(items, dependencyResponse(row.PublicID, row.ConsumerPublicID, row.ProviderPublicID, row.Kind, stringPointer(row.Notes)))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.InstanceDependency `json:"items"`
	}{items})
}

func (s apiServer) dependencyParams(r *http.Request, body api.InstanceDependencyWrite) (database.CreateInstanceDependencyParams, error) {
	if body.ConsumerInstancePublicId == body.ProviderInstancePublicId {
		return database.CreateInstanceDependencyParams{}, errors.New("an instance cannot use itself")
	}
	kind := "uses"
	if body.Kind != nil {
		kind = *body.Kind
	}
	if kind != "uses" && kind != "served_by" {
		return database.CreateInstanceDependencyParams{}, errors.New("dependency kind must be uses or served_by")
	}
	consumer, err := s.queries.GetInstanceByPublicID(r.Context(), body.ConsumerInstancePublicId)
	if err != nil {
		return database.CreateInstanceDependencyParams{}, err
	}
	provider, err := s.queries.GetInstanceByPublicID(r.Context(), body.ProviderInstancePublicId)
	if err != nil {
		return database.CreateInstanceDependencyParams{}, err
	}
	return database.CreateInstanceDependencyParams{ConsumerInstanceID: consumer.ID, ProviderInstanceID: provider.ID, Kind: kind, Notes: nullableTrimmed(body.Notes)}, nil
}

func (s apiServer) CreateInstanceDependency(w http.ResponseWriter, r *http.Request) {
	body, ok := readWriteBody[api.InstanceDependencyWrite](w, r)
	if !ok {
		return
	}
	params, err := s.dependencyParams(r, body)
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	params.ID, params.PublicID, err = newIDs()
	if err == nil {
		_, err = s.queries.CreateInstanceDependency(r.Context(), params)
	}
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeInstanceDependency(w, r, params.PublicID, http.StatusCreated)
}

func (s apiServer) GetInstanceDependency(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	s.writeInstanceDependency(w, r, publicID, http.StatusOK)
}

func (s apiServer) writeInstanceDependency(w http.ResponseWriter, r *http.Request, publicID string, status int) {
	row, err := s.queries.GetInstanceDependencyByPublicID(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeJSON(w, status, dependencyResponse(row.PublicID, row.ConsumerPublicID, row.ProviderPublicID, row.Kind, stringPointer(row.Notes)))
}

func (s apiServer) UpdateInstanceDependency(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	body, ok := readWriteBody[api.InstanceDependencyWrite](w, r)
	if !ok {
		return
	}
	if _, err := s.queries.GetInstanceDependencyByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	params, err := s.dependencyParams(r, body)
	if err != nil {
		s.writeParentOrInputError(w, err)
		return
	}
	_, err = s.queries.UpdateInstanceDependency(r.Context(), database.UpdateInstanceDependencyParams{
		ConsumerInstanceID: params.ConsumerInstanceID, ProviderInstanceID: params.ProviderInstanceID,
		Kind: params.Kind, Notes: params.Notes, PublicID: publicID,
	})
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	s.writeInstanceDependency(w, r, publicID, http.StatusOK)
}

func (s apiServer) DeleteInstanceDependency(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	rows, err := s.queries.DeleteInstanceDependency(r.Context(), publicID)
	deleteResource(w, rows, err)
}
