package golink

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
)

func TestLinksAPI(t *testing.T) {
	previousDB := db
	previousDev := *dev
	t.Cleanup(func() {
		db = previousDB
		*dev = previousDev
	})

	var err error
	db, err = NewSQLiteDB(filepath.Join(t.TempDir(), "hlims.db"))
	if err != nil {
		t.Fatal(err)
	}
	*dev = "127.0.0.1:8080"
	handler := serveHandler()

	createBody := []byte(`{"short":"oc1","url":"https://mini-pc-1.example.ts.net"}`)
	response := apiRequest(t, handler, http.MethodPost, "/.api/v1/links", createBody)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body.String())
	}
	var created api.Link
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Short != "oc1" || created.Url != "https://mini-pc-1.example.ts.net" {
		t.Fatalf("created link = %+v", created)
	}

	response = apiRequest(t, handler, http.MethodGet, "/.api/v1/links/oc1", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", response.Code, response.Body.String())
	}

	updateBody := []byte(`{"url":"https://mini-pc-1.example.ts.net/opencode"}`)
	response = apiRequest(t, handler, http.MethodPut, "/.api/v1/links/oc1", updateBody)
	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", response.Code, response.Body.String())
	}

	response = apiRequest(t, handler, http.MethodGet, "/.api/v1/links", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", response.Code, response.Body.String())
	}
	var listed struct {
		Items []api.Link `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 || listed.Items[0].Url != "https://mini-pc-1.example.ts.net/opencode" {
		t.Fatalf("listed links = %+v", listed.Items)
	}

	response = apiRequest(t, handler, http.MethodDelete, "/.api/v1/links/oc1", nil)
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", response.Code, response.Body.String())
	}

	response = apiRequest(t, handler, http.MethodGet, "/.api/v1/links/oc1", nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("deleted link status = %d, body = %s", response.Code, response.Body.String())
	}
}

func apiRequest(t *testing.T, handler http.Handler, method, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
