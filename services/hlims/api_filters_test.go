package hlims

import (
	"net/http"
	"testing"
)

func TestInventoryListFiltersUseSlugsAndRelationships(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Example"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "Home"})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "kind": "virtual_machine", "name": "worker-host",
	})
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "OpenCode"})
	createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Other Service"})
	worker := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": service["publicId"], "machinePublicId": machine["publicId"], "name": "production",
	})
	managed := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": service["publicId"], "hostingKind": "managed", "managedProvider": "Example Cloud", "name": "production",
	})
	endpoint := createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": managed["publicId"], "name": "Cloud", "directUrl": "https://app.example.test/",
	})
	for _, tc := range []struct {
		path string
		id   any
	}{
		{path: "/api/v1/services?slug=opencode", id: service["publicId"]},
		{path: "/api/v1/machines?slug=worker-host", id: machine["publicId"]},
		{path: "/api/v1/instances?service=opencode&machine=worker-host&slug=production&hostingKind=machine", id: worker["publicId"]},
		{path: "/api/v1/instances?service=opencode&hostingKind=managed", id: managed["publicId"]},
		{path: "/api/v1/instance-endpoints?instancePublicId=" + managed["publicId"].(string), id: endpoint["publicId"]},
	} {
		response := apiRequest(t, handler, http.MethodGet, tc.path, nil, "")
		assertStatus(t, response, http.StatusOK)
		items := decodeObject(t, response)["items"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["publicId"] != tc.id {
			t.Errorf("%s items = %#v; want one matching record", tc.path, items)
		}
	}
	response := apiRequest(t, handler, http.MethodGet, "/api/v1/instances?service=missing", nil, "")
	assertStatus(t, response, http.StatusOK)
	if items := decodeObject(t, response)["items"].([]any); len(items) != 0 {
		t.Fatalf("unknown Service filter returned %#v", items)
	}
}
