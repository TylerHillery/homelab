package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
)

func TestPatchUsesSlugAndPreservesFieldsWithRevision(t *testing.T) {
	t.Parallel()
	var updated map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services":
			_, _ = w.Write([]byte(`{"items":[{"publicId":"service00001","slug":"opencode"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services/service00001":
			w.Header().Set("ETag", `"current"`)
			_, _ = w.Write([]byte(`{"publicId":"service00001","name":"OpenCode","slug":"opencode","hasLogo":true,"description":"old"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/services/service00001":
			if r.Header.Get("If-Match") != `"current"` {
				t.Errorf("missing revision precondition: %q", r.Header.Get("If-Match"))
			}
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Error(err)
			}
			_, _ = w.Write([]byte(`{"publicId":"service00001","name":"OpenCode","slug":"opencode","description":"updated","hasLogo":true}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
	}))
	t.Cleanup(server.Close)
	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Input: strings.NewReader(`{"description":"updated"}`), Output: &output, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "services", "patch", "opencode"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if updated["name"] != "OpenCode" || updated["slug"] != "opencode" || updated["description"] != "updated" || updated["publicId"] != nil || updated["hasLogo"] != nil {
		t.Fatalf("patch lost data or sent read-only fields: %#v", updated)
	}
	if !strings.Contains(output.String(), `"name": "OpenCode"`) {
		t.Fatalf("patch output = %s", output.String())
	}
}

func TestMergeFieldsPreservesNestedDataAndLargeIntegers(t *testing.T) {
	t.Parallel()
	current, err := decodeObject([]byte(`{"publicId":"item00000001","links":[{"url":"https://example.test"}],"memoryBytes":9007199254740993,"placement":{"type":"asset","parentAssetPublicId":"parent000001","slot":"A"}}`))
	if err != nil {
		t.Fatal(err)
	}
	patch, err := decodeObject([]byte(`{"placement":{"slot":"B"},"notes":"updated"}`))
	if err != nil {
		t.Fatal(err)
	}
	mergeFields(current, patch)
	body, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"memoryBytes":9007199254740993`, `"parentAssetPublicId":"parent000001"`, `"slot":"B"`, `"links":[{"url":"https://example.test"}]`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("merged record lost %q: %s", want, body)
		}
	}
}

func TestManagedEndpointPatchStripsComputedHostMode(t *testing.T) {
	t.Parallel()
	endpoint := map[string]any{
		"publicId": "endpoint0001", "instancePublicId": "instance0001", "directUrl": "https://app.example.test/",
		"name": "Cloud", "hostType": "auto", "basePath": "", "isPreferred": true,
	}
	stripReadOnlyFields(endpoint, apiclient.InstanceEndpoints)
	if endpoint["hostType"] != nil || endpoint["publicId"] != nil || endpoint["directUrl"] != "https://app.example.test/" {
		t.Fatalf("managed endpoint patch body = %#v", endpoint)
	}
}
