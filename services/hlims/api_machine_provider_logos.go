package hlims

import (
	"net/http"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) GetMachineProviderLogo(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	row, err := s.queries.GetMachineProviderLogo(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeLogo(w, r, row.ContentType, row.ImageData)
}

func (s apiServer) UpdateMachineProviderLogo(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	if _, err := s.queries.GetMachineProviderByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	contentType, data, ok := readLogo(w, r)
	if !ok {
		return
	}
	if err := s.queries.UpsertMachineProviderLogo(r.Context(), database.UpsertMachineProviderLogoParams{ContentType: contentType, ImageData: data, PublicID: publicID}); err != nil {
		writeDatabaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s apiServer) DeleteMachineProviderLogo(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	if _, err := s.queries.GetMachineProviderByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	rows, err := s.queries.DeleteMachineProviderLogo(r.Context(), publicID)
	deleteResource(w, rows, err)
}
