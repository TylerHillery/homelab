package golink

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
)

type apiServer struct{}

func registerAPIHandlers(mux *http.ServeMux) {
	api.HandlerFromMuxWithBaseURL(apiServer{}, mux, "/.api/v1")
}

func (apiServer) ListLinks(w http.ResponseWriter, _ *http.Request) {
	links, err := db.LoadAll()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to list links")
		return
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Short < links[j].Short })

	items := make([]api.Link, 0, len(links))
	for _, link := range links {
		items = append(items, apiLink(link))
	}
	writeJSON(w, http.StatusOK, struct {
		Items []api.Link `json:"items"`
	}{Items: items})
}

func (apiServer) GetLink(w http.ResponseWriter, _ *http.Request, short api.Short) {
	link, err := db.Load(short)
	if errors.Is(err, fs.ErrNotExist) {
		writeAPIError(w, http.StatusNotFound, "not_found", "link not found")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to load link")
		return
	}
	writeJSON(w, http.StatusOK, apiLink(link))
}

func (apiServer) CreateLink(w http.ResponseWriter, r *http.Request) {
	var body api.CreateLink
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, err := db.Load(body.Short); err == nil {
		writeAPIError(w, http.StatusConflict, "already_exists", "link already exists")
		return
	} else if !errors.Is(err, fs.ErrNotExist) {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to load link")
		return
	}
	createOrUpdateAPILink(w, r, body.Short, body.Url, nil)
}

func (apiServer) UpdateLink(w http.ResponseWriter, r *http.Request, short api.Short) {
	var body api.UpdateLink
	if !decodeJSON(w, r, &body) {
		return
	}
	link, err := db.Load(short)
	if errors.Is(err, fs.ErrNotExist) {
		writeAPIError(w, http.StatusNotFound, "not_found", "link not found")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to load link")
		return
	}
	createOrUpdateAPILink(w, r, short, body.Url, link)
}

func (apiServer) DeleteLink(w http.ResponseWriter, r *http.Request, short api.Short) {
	link, err := db.Load(short)
	if errors.Is(err, fs.ErrNotExist) {
		writeAPIError(w, http.StatusNotFound, "not_found", "link not found")
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to load link")
		return
	}
	current, ok := apiCurrentUser(w, r)
	if !ok {
		return
	}
	if !canEditLink(r.Context(), link, current) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "link is owned by another user")
		return
	}
	if err := db.Delete(short); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to delete link")
		return
	}
	deleteLinkStats(link)
	w.WriteHeader(http.StatusNoContent)
}

func createOrUpdateAPILink(w http.ResponseWriter, r *http.Request, short, long string, existing *Link) {
	if !reShortName.MatchString(short) {
		writeAPIError(w, http.StatusBadRequest, "invalid_short", "short may only contain letters, numbers, dash, and period")
		return
	}
	if _, err := texttemplate.New("").Funcs(expandFuncMap).Parse(long); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_url", "url contains an invalid template")
		return
	}
	current, ok := apiCurrentUser(w, r)
	if !ok {
		return
	}
	if !canEditLink(r.Context(), existing, current) {
		writeAPIError(w, http.StatusForbidden, "forbidden", "link is owned by another user")
		return
	}

	now := time.Now().UTC()
	status := http.StatusOK
	if existing == nil {
		existing = &Link{Short: short, Created: now, Owner: current.login}
		status = http.StatusCreated
	}
	existing.Short = short
	existing.Long = long
	existing.LastEdit = now
	if err := db.Save(existing); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to save link")
		return
	}
	if status == http.StatusCreated {
		totalLinkCount.Inc()
	}
	writeJSON(w, status, apiLink(existing))
}

func apiCurrentUser(w http.ResponseWriter, r *http.Request) (user, bool) {
	current, err := currentUser(r)
	if err != nil {
		writeAPIError(w, http.StatusUnauthorized, "unauthorized", "Tailscale identity is required")
		return user{}, false
	}
	return current, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		writeAPIError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "request body must contain one JSON value")
		return false
	}
	return true
}

func apiLink(link *Link) api.Link {
	return api.Link{
		Short:     link.Short,
		Url:       link.Long,
		Owner:     link.Owner,
		CreatedAt: link.Created,
		UpdatedAt: link.LastEdit,
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
