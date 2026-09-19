package hlims

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

const maxLogoSize = 1 << 20

var allowedLogoTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

func (s apiServer) GetServiceLogo(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	row, err := s.queries.GetServiceLogo(r.Context(), publicID)
	if err != nil {
		writeDatabaseError(w, err)
		return
	}
	writeLogo(w, r, row.ContentType, row.ImageData)
}

func writeLogo(w http.ResponseWriter, r *http.Request, contentType string, data []byte) {
	digest := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(digest[:]) + `"`
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s apiServer) UpdateServiceLogo(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	if _, err := s.queries.GetServiceByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	contentType, data, ok := readLogo(w, r)
	if !ok {
		return
	}
	if err := s.queries.UpsertServiceLogo(r.Context(), database.UpsertServiceLogoParams{ContentType: contentType, ImageData: data, PublicID: publicID}); err != nil {
		writeDatabaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readLogo(w http.ResponseWriter, r *http.Request) (string, []byte, bool) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "logo must be PNG, JPEG, or WebP")
		return "", nil, false
	}
	if _, allowed := allowedLogoTypes[contentType]; !allowed {
		writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "logo must be PNG, JPEG, or WebP")
		return "", nil, false
	}
	if r.ContentLength > maxLogoSize {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "logo_too_large", "logo must not exceed 1 MiB")
		return "", nil, false
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxLogoSize+1))
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, "logo_too_large", "logo must not exceed 1 MiB")
		} else {
			writeAPIError(w, http.StatusBadRequest, "invalid_input", "could not read logo")
		}
		return "", nil, false
	}
	if len(data) == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "logo must not be empty")
		return "", nil, false
	}
	if len(data) > maxLogoSize {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "logo_too_large", "logo must not exceed 1 MiB")
		return "", nil, false
	}
	if detected := http.DetectContentType(data); detected != contentType {
		writeAPIError(w, http.StatusBadRequest, "invalid_input", "logo bytes do not match Content-Type")
		return "", nil, false
	}
	return contentType, data, true
}

func (s apiServer) DeleteServiceLogo(w http.ResponseWriter, r *http.Request, publicID api.PublicId) {
	if _, err := s.queries.GetServiceByPublicID(r.Context(), publicID); err != nil {
		writeDatabaseError(w, err)
		return
	}
	rows, err := s.queries.DeleteServiceLogo(r.Context(), publicID)
	deleteResource(w, rows, err)
}
