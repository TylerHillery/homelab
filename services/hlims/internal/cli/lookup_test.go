package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetServiceBySlugKeepsBrandedDisplayName(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/services":
			if r.URL.Query().Get("slug") != "opencode" {
				t.Errorf("slug lookup query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"items":[{"publicId":"service00001","name":"OpenCode","slug":"opencode"}]}`))
		case "/api/v1/services/service00001":
			_, _ = w.Write([]byte(`{"publicId":"service00001","name":"OpenCode","slug":"opencode"}`))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	var out bytes.Buffer
	cmd := NewRootCommand(Dependencies{Output: &out, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	cmd.SetArgs([]string{"--api-url", server.URL + "/api/v1", "services", "get", "opencode"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"name": "OpenCode"`) || !strings.Contains(out.String(), `"slug": "opencode"`) {
		t.Fatalf("branded Service lookup = %s", out.String())
	}
}

func TestInstanceListFiltersReachAPI(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances" || r.URL.Query().Get("service") != "grafana" || r.URL.Query().Get("machine") != "badger" || r.URL.Query().Get("hostingKind") != "machine" {
			t.Errorf("filtered instances request = %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)
	cmd := NewRootCommand(Dependencies{Output: &bytes.Buffer{}, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	cmd.SetArgs([]string{"--api-url", server.URL + "/api/v1", "instances", "list", "--service", "grafana", "--machine", "badger", "--hosting-kind", "machine"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
