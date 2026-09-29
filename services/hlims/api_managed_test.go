package hlims

import (
	"net/http"
	"strings"
	"testing"
)

func TestManagedInstanceDirectURLsWithoutMachineOrAddress(t *testing.T) {
	handler := newAPITestHandler(t)
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Sentry"})
	serviceID := service["publicId"].(string)
	invalid := apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": serviceID, "hostingKind": "managed", "name": "my-org",
	}, "application/json")
	assertStatus(t, invalid, http.StatusBadRequest)
	instance := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": serviceID, "hostingKind": "managed", "managedProvider": " Sentry Cloud ", "name": "my-org",
	})
	instanceID := instance["publicId"].(string)
	if instance["managedProvider"] != "Sentry Cloud" || instance["hostingKind"] != "managed" || instance["machinePublicId"] != nil || instance["port"] != nil {
		t.Fatalf("managed instance has invented machine or port: %#v", instance)
	}
	for _, url := range []string{"http://sentry.example/projects/", "https://user:pass@sentry.example/", "https://sentry.example/#fragment"} {
		response := apiRequest(t, handler, http.MethodPost, "/api/v1/instance-endpoints", map[string]any{
			"instancePublicId": instanceID, "name": "Invalid", "directUrl": url,
		}, "application/json")
		assertStatus(t, response, http.StatusBadRequest)
	}
	primary := createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": instanceID, "name": "Projects", "directUrl": "https://org.example.test/projects/", "isPreferred": true,
	})
	projectURL := "https://org.example.test/projects/example-project/?project=1234"
	project := createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": instanceID, "name": "Example project", "directUrl": projectURL,
	})
	if project["addressPublicId"] != nil || project["port"] != nil || project["scheme"] != nil {
		t.Fatalf("managed endpoint has invented address or port: %#v", project)
	}
	resolved := apiRequest(t, handler, http.MethodGet, "/api/v1/resolve/sentry/my-org", nil, "")
	assertStatus(t, resolved, http.StatusOK)
	if got := decodeObject(t, resolved); got["url"] != "https://org.example.test/projects/" || got["via"] != "managed" {
		t.Fatalf("resolved managed destination = %#v", got)
	}
	assertStatus(t, apiRequest(t, handler, http.MethodGet, "/api/v1/resolve/sentry/my-org?via=tailnet", nil, ""), http.StatusBadRequest)
	selected := apiRequest(t, handler, http.MethodGet, "/sentry/my-org", nil, "")
	assertStatus(t, selected, http.StatusFound)
	if selected.Header().Get("Location") != "https://org.example.test/projects/" {
		t.Fatalf("managed redirect = %q", selected.Header().Get("Location"))
	}
	updated := apiRequest(t, handler, http.MethodPut, "/api/v1/instance-endpoints/"+project["publicId"].(string), map[string]any{
		"instancePublicId": instanceID, "name": "Example project", "directUrl": projectURL, "isPreferred": true,
	}, "application/json")
	assertStatus(t, updated, http.StatusOK)
	if decodeObject(t, apiRequest(t, handler, http.MethodGet, "/api/v1/instance-endpoints/"+primary["publicId"].(string), nil, ""))["isPreferred"] != false {
		t.Fatal("managed preferred endpoint did not demote the old one")
	}
	selected = apiRequest(t, handler, http.MethodGet, "/sentry/my-org?tab=issues", nil, "")
	assertStatus(t, selected, http.StatusFound)
	if got := selected.Header().Get("Location"); got != projectURL+"&tab=issues" {
		t.Fatalf("managed URL query was lost: %q", got)
	}
	instances := apiRequest(t, handler, http.MethodGet, "/api/v1/instances", nil, "")
	assertStatus(t, instances, http.StatusOK)
	if !strings.Contains(instances.Body.String(), instanceID) {
		t.Fatal("managed instance missing from list")
	}
	assertStatus(t, apiRequest(t, handler, http.MethodGet, "/api/v1/topology", nil, ""), http.StatusOK)
	page := consoleRequest(handler, http.MethodGet, "/console/services/", false)
	assertStatus(t, page, http.StatusOK)
	for _, expected := range []string{"HLIMS / Services", "Sentry Cloud", "Managed", "Example project", "/sentry/my-org", "project=1234", `href="/console/services/" aria-current="page"`, `class="service-catalog-overview"`, `<details class="service-catalog-details"`, `class="service-catalog-popover"`} {
		if !strings.Contains(page.Body.String(), expected) {
			t.Errorf("services page missing %q", expected)
		}
	}
	if strings.Contains(page.Body.String(), "Machine: <a") {
		t.Fatal("managed service appeared to belong to a machine")
	}
}
