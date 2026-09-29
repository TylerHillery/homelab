package hlims

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConditionalUpdateRejectsStaleResource(t *testing.T) {
	handler := newAPITestHandler(t)
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "OpenCode"})
	id := service["publicId"].(string)
	get := apiRequest(t, handler, http.MethodGet, "/api/v1/services/"+id, nil, "")
	assertStatus(t, get, http.StatusOK)
	version := get.Header().Get("ETag")
	if version == "" {
		t.Fatal("inventory GET did not provide an ETag")
	}
	request := func(name string) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(map[string]any{"name": name, "slug": "opencode"})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/v1/services/"+id, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("If-Match", version)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	first := request("OpenCode updated")
	assertStatus(t, first, http.StatusOK)
	second := request("Stale replacement")
	assertStatus(t, second, http.StatusPreconditionFailed)
	if got := decodeObject(t, apiRequest(t, handler, http.MethodGet, "/api/v1/services/"+id, nil, ""))["name"]; got != "OpenCode updated" {
		t.Fatalf("stale PUT overwrote Service: %v", got)
	}
}
