package hlims

import (
	"net/http"
	"strings"
	"testing"
)

func TestDeploymentGroupsScopeServiceInstancesAndKeepIndependentInstances(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Demo"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "Home"})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "name": "demo-host", "kind": "bare_metal",
	})
	otherMachine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "name": "other-host", "kind": "bare_metal",
	})
	postgres := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "PostgreSQL"})
	apiService := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Example API"})
	docsService := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Example Docs"})
	compose := func(name, project string) map[string]any {
		return createAPIResource(t, handler, "/api/v1/deployments", map[string]any{
			"machinePublicId": machine["publicId"], "name": name,
			"workingDirectory": "/srv/" + project, "composeProject": project, "composeFiles": []string{"compose.yaml", "override.yaml"},
		})
	}
	shared := compose("Shared Infrastructure", "shared")
	web := compose("Web Application", "web")
	if response := apiRequest(t, handler, http.MethodGet, "/api/v1/deployments/"+shared["publicId"].(string), nil, ""); response.Header().Get("ETag") == "" {
		t.Fatal("deployment GET lacks revision ETag for CLI patch")
	}
	create := func(service map[string]any, name string, deployment map[string]any, member string) map[string]any {
		body := map[string]any{"servicePublicId": service["publicId"], "machinePublicId": machine["publicId"], "name": name}
		if deployment != nil {
			body["deploymentPublicId"] = deployment["publicId"]
			body["deploymentMember"] = member
		}
		return createAPIResource(t, handler, "/api/v1/instances", body)
	}
	sharedDB := create(postgres, "Shared DB", shared, "db")
	create(postgres, "App DB", web, "db")
	create(postgres, "Local CLI A", nil, "")
	create(postgres, "Local CLI B", nil, "")
	unit := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": apiService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Host Unit", "systemdUnit": "api.service", "systemdScope": "user", "systemdUser": "deploy",
	})
	createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": postgres["publicId"], "machinePublicId": machine["publicId"],
		"name": "Host Database", "systemdUnit": "postgresql.service", "systemdScope": "user", "systemdUser": "deploy",
	})
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": postgres["publicId"], "machinePublicId": machine["publicId"],
		"name": "Invalid missing user", "systemdUnit": "other.service", "systemdScope": "user",
	}, "application/json"), http.StatusBadRequest)
	webAPI := create(apiService, "Web API", web, "api")
	docs := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": docsService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Static Docs", "deploymentPublicId": web["publicId"], "deploymentMember": "site/dist", "deploymentRole": "static_content",
	})
	createAPIResource(t, handler, "/api/v1/instance-dependencies", map[string]any{
		"consumerInstancePublicId": docs["publicId"], "providerInstancePublicId": webAPI["publicId"], "kind": "served_by",
	})
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": docsService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Invalid standalone docs", "deploymentRole": "static_content",
	}, "application/json"), http.StatusBadRequest)
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": docsService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Invalid systemd docs", "systemdUnit": "docs.service", "systemdScope": "system", "deploymentRole": "static_content",
	}, "application/json"), http.StatusBadRequest)
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": apiService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Invalid both managers", "deploymentPublicId": web["publicId"], "deploymentMember": "worker", "systemdUnit": "worker.service", "systemdScope": "system",
	}, "application/json"), http.StatusBadRequest)
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": docsService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Invalid static listener", "deploymentPublicId": web["publicId"], "deploymentMember": "other/dist", "deploymentRole": "static_content", "port": 443,
	}, "application/json"), http.StatusBadRequest)
	createAPIResource(t, handler, "/api/v1/instance-dependencies", map[string]any{
		"consumerInstancePublicId": webAPI["publicId"], "providerInstancePublicId": sharedDB["publicId"],
	})
	if response := apiRequest(t, handler, http.MethodGet, "/api/v1/instance-dependencies", nil, ""); !strings.Contains(response.Body.String(), sharedDB["publicId"].(string)) {
		t.Fatal("cross-deployment usage relationship missing from API")
	}
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instance-dependencies", map[string]any{
		"consumerInstancePublicId": sharedDB["publicId"], "providerInstancePublicId": sharedDB["publicId"],
	}, "application/json"), http.StatusBadRequest)
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": postgres["publicId"], "machinePublicId": otherMachine["publicId"],
		"name": "Invalid Host", "deploymentPublicId": shared["publicId"], "deploymentMember": "db2",
	}, "application/json"), http.StatusBadRequest)
	otherDeployment := createAPIResource(t, handler, "/api/v1/deployments", map[string]any{
		"machinePublicId": otherMachine["publicId"], "name": "Other Stack",
		"workingDirectory": "/srv/other", "composeProject": "other", "composeFiles": []string{"compose.yaml"},
	})
	createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": postgres["publicId"], "machinePublicId": otherMachine["publicId"],
		"name": "Other DB", "deploymentPublicId": otherDeployment["publicId"], "deploymentMember": "db",
	})
	child := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"],
		"parentMachinePublicId": machine["publicId"], "name": "child-host", "kind": "virtual_machine",
	})
	createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": postgres["publicId"], "machinePublicId": child["publicId"],
		"name": "Child DB", "systemdUnit": "postgresql.service", "systemdScope": "system",
	})
	assertStatus(t, apiRequest(t, handler, http.MethodDelete, "/api/v1/deployments/"+shared["publicId"].(string), nil, ""), http.StatusConflict)

	servicesPage := consoleRequest(handler, http.MethodGet, "/console/services/", false)
	assertStatus(t, servicesPage, http.StatusOK)
	text := servicesPage.Body.String()
	start := strings.Index(text, `id="service-`+postgres["publicId"].(string)+`"`)
	if start == -1 {
		t.Fatal("PostgreSQL Service card missing")
	}
	card := text[start:]
	if end := strings.Index(card, "</article>"); end != -1 {
		card = card[:end]
	}
	for _, expected := range []string{"Shared Infrastructure", "Web Application", "Independent instances", "Shared DB", "App DB", "Local CLI A", "Local CLI B"} {
		if !strings.Contains(card, expected) {
			t.Errorf("PostgreSQL card missing %q", expected)
		}
	}
	for _, group := range []struct{ name, member, other string }{
		{"Shared Infrastructure", "Shared DB", "App DB"},
		{"Web Application", "App DB", "Shared DB"},
	} {
		segment := card[strings.Index(card, group.name):]
		if next := strings.Index(segment, `class="service-group-heading"`); next != -1 {
			segment = segment[:next]
		}
		if !strings.Contains(segment, group.member) || strings.Contains(segment, group.other) {
			t.Errorf("%s included an instance from another deployment", group.name)
		}
	}
	machinesPage := consoleRequest(handler, http.MethodGet, "/console/machines/", false)
	assertStatus(t, machinesPage, http.StatusOK)
	machineText := machinesPage.Body.String()
	for _, expected := range []string{"deployment-docker.svg", "deployment-systemd.svg", "override.yaml", "api.service", "Uses PostgreSQL / Shared DB", "Static files · site/dist", "Served by Example API / Web API", "systemd units", "user · deploy", "Independent services", "Local CLI A", "Local CLI B"} {
		if !strings.Contains(machineText, expected) {
			t.Errorf("Machine card missing %q", expected)
		}
	}
	if count := strings.Count(machineText, `class="machine-systemd-manager"`); count != 1 {
		t.Fatalf("systemd units on this host formed %d manager groups; want one", count)
	}
	if strings.Contains(machineText, "Host Units") {
		t.Fatal("systemd unit was rendered as an artificial Deployment")
	}
	sharedStart := strings.Index(machineText, `id="deployment-`+shared["publicId"].(string)+`"`)
	webStart := strings.Index(machineText, `id="deployment-`+web["publicId"].(string)+`"`)
	if sharedStart < 0 || webStart <= sharedStart || !strings.Contains(machineText[sharedStart:webStart], "Shared DB") || strings.Contains(machineText[sharedStart:webStart], "App DB") {
		t.Fatal("Shared Infrastructure Machine group included another deployment's PostgreSQL Instance")
	}
	otherStart := strings.Index(machineText, `id="machine-`+otherMachine["publicId"].(string)+`"`)
	otherGroup := strings.Index(machineText, `id="deployment-`+otherDeployment["publicId"].(string)+`"`)
	if otherStart < 0 || otherGroup <= otherStart || !strings.Contains(machineText[otherGroup:], "Other DB") || strings.Contains(machineText[sharedStart:webStart], "Other DB") {
		t.Fatal("deployment instances leaked between Machines")
	}
	if strings.Contains(machineText, "Child DB") {
		t.Fatal("collapsed child Machine's systemd unit appeared on its parent")
	}
	childPage := consoleRequest(handler, http.MethodGet, "/console/machines/"+machine["publicId"].(string)+"/children", true)
	assertStatus(t, childPage, http.StatusOK)
	for _, expected := range []string{"systemd units", "postgresql.service", "Child DB"} {
		if !strings.Contains(childPage.Body.String(), expected) {
			t.Errorf("nested Machine deployment missing %q", expected)
		}
	}
	assertStatus(t, consoleRequest(handler, http.MethodGet, "/console/deployments/", false), http.StatusNotFound)
	if unit["deploymentPublicId"] != nil || unit["systemdUnit"] != "api.service" {
		t.Fatal("systemd unit was incorrectly stored as a Deployment membership")
	}
}
