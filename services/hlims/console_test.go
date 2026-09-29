package hlims

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
)

func TestConsoleRendersTopologyAndHTMXFragments(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Homelab"})
	providerID := provider["publicId"].(string)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+providerID+"/logo", []byte("\x89PNG\r\n\x1a\nHLIMS"), "image/png"), http.StatusNoContent)
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": providerID, "name": "Primary Home",
	})
	root := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": area["publicId"], "name": "Badger",
		"kind": "bare_metal", "hostname": "badger.lan", "operatingSystem": "Ubuntu", "cpuCount": 4, "cpuThreadCount": 8,
		"memoryBytes": 8589934592, "storageBytes": 250000000000, "isFavorite": true,
	})
	rootID := root["publicId"].(string)
	createAPIResource(t, handler, "/api/v1/machine-users", map[string]any{
		"machinePublicId": rootID, "username": "admin", "isPreferred": true,
	})
	createAPIResource(t, handler, "/api/v1/machine-users", map[string]any{
		"machinePublicId": rootID, "username": "guest", "isPreferred": false,
	})
	if root["isFavorite"] != true {
		t.Fatalf("created root favorite = %#v", root["isFavorite"])
	}
	child := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": area["publicId"],
		"parentMachinePublicId": rootID, "name": "Automation VM", "kind": "virtual_machine",
		"virtualizationPlatform": "KVM", "isFavorite": true,
	})
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Grafana"})
	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/services/"+service["publicId"].(string)+"/logo", []byte("\x89PNG\r\n\x1a\nHLIMS"), "image/png"), http.StatusNoContent)
	instance := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": service["publicId"], "machinePublicId": rootID, "name": "Production", "port": 3000,
	})
	lan := createAPIResource(t, handler, "/api/v1/networks", map[string]any{
		"areaPublicId": area["publicId"], "name": "Home LAN", "kind": "lan",
	})
	tailnet := createAPIResource(t, handler, "/api/v1/networks", map[string]any{"name": "Tailnet", "kind": "tailnet"})
	lanAddress := createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": lan["publicId"], "machinePublicId": rootID, "address": "192.168.1.10",
	})
	tailnetAddress := createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": tailnet["publicId"], "machinePublicId": rootID, "address": "100.64.0.10", "dnsName": "badger.example.ts.net",
	})
	for _, endpoint := range []struct {
		name      string
		addressID any
		preferred bool
		hostType  string
	}{
		{name: "LAN", addressID: lanAddress["publicId"], preferred: true, hostType: "ip"},
		{name: "Tailnet DNS", addressID: tailnetAddress["publicId"], hostType: "dns"},
		{name: "Tailnet IP", addressID: tailnetAddress["publicId"], hostType: "ip"},
	} {
		createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
			"instancePublicId": instance["publicId"], "addressPublicId": endpoint.addressID,
			"name": endpoint.name, "scheme": "http", "port": 3000, "isPreferred": endpoint.preferred, "hostType": endpoint.hostType,
		})
	}

	page := consoleRequest(handler, http.MethodGet, "/console/machines/", false)
	assertStatus(t, page, http.StatusOK)
	body := page.Body.String()
	if !strings.Contains(body, "favorite-group") {
		t.Fatalf("console did not render favorites: %s", body)
	}
	servicesPage := consoleRequest(handler, http.MethodGet, "/console/services/", false)
	assertStatus(t, servicesPage, http.StatusOK)
	for _, expected := range []string{"Grafana", "Machine-hosted", "Badger", "/console/machines/#machine-" + rootID, "http://192.168.1.10:3000/", `href="/console/services/" aria-current="page"`} {
		if !strings.Contains(servicesPage.Body.String(), expected) {
			t.Errorf("service-first page does not contain %q", expected)
		}
	}
	for _, expected := range []string{
		"<!doctype html>", "HLIMS / Machine Topology", "<b>H</b>ome <b>L</b>ab <b>I</b>nformation <b>M</b>anagement <b>S</b>ystem", "Homelab", "Primary Home", "Badger", "Grafana",
		`href="/grafana/production?host=badger" target="_blank" rel="noopener noreferrer"`, `hx-trigger="toggle once"`, "/console/static/htmx-4.0.0.min.js",
		"/console/static/theme.js", "/console/static/route-menu.js", "/console/static/favicon.svg", "data-theme-toggle",
		"expansion=all", "expansion=none", `class="provider-group"`,
		"data-tooltip=\"Expand all\"", "/api/v1/machine-providers/" + providerID + "/logo",
		"/api/v1/services/" + service["publicId"].(string) + "/logo",
		`class="favorite-group"`, `class="favorite-machines"`, "2 machines",
		`aria-label="Remove Badger from favorites"`, `aria-label="Remove Automation VM from favorites"`,
		`hx-put="/console/machines/` + rootID + `/favorite?value=false"`,
		`aria-label="Choose access path for Production"`,
		`/console/static/reachability.js`,
		`/console/static/copy-actions.js`, `data-copy-value="ssh admin@badger.lan"`,
		`data-ssh-user="admin" data-ssh-host="badger.lan"`, `aria-label="Choose SSH user for Badger"`,
		`data-ssh-user-option="admin"`, `data-ssh-user-option="guest"`, `aria-label="Choose SSH host for Badger"`,
		`data-ssh-host-option="badger.example.ts.net"`, `data-ssh-host-option="192.168.1.10"`, `data-copy-value="ssh admin@badger.lan"`,
		`data-reachability-kind="instance" data-reachability-id="` + instance["publicId"].(string) + `" data-reachability-url="/grafana/production?host=badger"`,
		`data-reachability-url="http://192.168.1.10:3000/"`,
		`data-reachability-url="http://badger.example.ts.net:3000/"`,
		`href="/api-docs/" target="_blank" rel="noopener"`,
		`<details class="machine-addresses">`, `summary class="machine-addresses-label" aria-label="Network addresses for Badger"`, `class="machine-os machine-fact`, "192.168.1.10", "100.64.0.10", "badger.example.ts.net",
		`data-copy-value="192.168.1.10" aria-label="Copy IP address 192.168.1.10"`,
		`data-copy-value="badger.example.ts.net" aria-label="Copy DNS name badger.example.ts.net"`,
		"Cores", "Threads", "Memory", "Storage", "8 GiB", "250 GB", `src="/console/static/os-linux.svg" alt=""`, `src="/console/static/os-ubuntu.svg" alt=""`, `Linux</span><span class="machine-os-separator"`, `class="service-logo has-image"`,
		`href="http://badger.example.ts.net:3000/" target="_blank" rel="noopener noreferrer"`,
		`href="http://100.64.0.10:3000/" target="_blank" rel="noopener noreferrer"`,
		`href="http://192.168.1.10:3000/" target="_blank" rel="noopener noreferrer"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("console page does not contain %q", expected)
		}
	}
	themeScript := `<script src="/console/static/theme.js"></script>`
	stylesheetTag := `<link rel="stylesheet" href="/console/static/console.css">`
	if scriptAt, styleAt := strings.Index(body, themeScript), strings.Index(body, stylesheetTag); scriptAt < 0 || styleAt < 0 || scriptAt > styleAt {
		t.Fatal("saved theme must be applied before the stylesheet can paint")
	}
	if !strings.Contains(page.Header().Get("Content-Security-Policy"), "connect-src http: https:") || page.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("console security headers = %#v", page.Header())
	}
	if strings.Contains(body, "machine-inspector") || strings.Contains(body, "inspect-action") {
		t.Fatal("console still rendered the removed machine inspector")
	}
	if strings.Contains(body, "All access paths") || strings.Contains(body, "via lan") || strings.Contains(body, "via tailnet") {
		t.Fatal("console rendered duplicate or ambiguous route menus")
	}
	if !strings.Contains(body, `<details class="machine-addresses">`) || strings.Contains(body, `<details class="machine-addresses" open`) {
		t.Fatal("network addresses should use a closed, non-shifting details menu")
	}
	if strings.Contains(body, `data-reachability-kind="machine"`) {
		t.Fatal("console rendered a Machine reachability indicator that browsers cannot verify")
	}
	if strings.Contains(body, "provider-rail") {
		t.Fatal("console still rendered the removed provider sidebar")
	}
	if strings.Contains(body, "workspace-stats") {
		t.Fatal("console still rendered global topology counters")
	}
	if strings.Contains(body, "workspace-header") || strings.Contains(body, ">All systems<") {
		t.Fatal("console still rendered the removed topology introduction")
	}
	if toolbarAt, workspaceAt := strings.Index(body, `class="topology-toolbar"`), strings.Index(body, `<main id="workspace"`); toolbarAt < 0 || workspaceAt < 0 || toolbarAt > workspaceAt {
		t.Fatal("topology controls are not rendered in the masthead")
	}

	unfavorite := consoleRequest(handler, http.MethodPut, "/console/machines/"+rootID+"/favorite?value=false", true)
	assertStatus(t, unfavorite, http.StatusOK)
	if !strings.Contains(unfavorite.Body.String(), `aria-label="Add Badger to favorites"`) || !strings.Contains(unfavorite.Body.String(), `aria-label="Remove Automation VM from favorites"`) {
		t.Fatalf("favorite toggle did not synchronize topology = %s", unfavorite.Body.String())
	}
	storedRoot := apiRequest(t, handler, http.MethodGet, "/api/v1/machines/"+rootID, nil, "")
	assertStatus(t, storedRoot, http.StatusOK)
	if decodeObject(t, storedRoot)["isFavorite"] != false {
		t.Fatal("favorite toggle did not persist")
	}
	assertStatus(t, consoleRequest(handler, http.MethodPut, "/console/machines/"+rootID+"/favorite?value=maybe", true), http.StatusBadRequest)
	assertStatus(t, consoleRequest(handler, http.MethodPut, "/console/machines/missing00000/favorite?value=true", true), http.StatusNotFound)

	providerFragment := consoleRequest(handler, http.MethodGet, "/console/machine-providers/"+providerID, true)
	assertStatus(t, providerFragment, http.StatusOK)
	if strings.Contains(providerFragment.Body.String(), "<!doctype html>") || !strings.Contains(providerFragment.Body.String(), `class="provider-group"`) {
		t.Fatalf("provider fragment = %s", providerFragment.Body.String())
	}
	expanded := consoleRequest(handler, http.MethodGet, "/console/machines/topology?expansion=all", true)
	assertStatus(t, expanded, http.StatusOK)
	if !strings.Contains(expanded.Body.String(), `class="provider-group" open`) || !strings.Contains(expanded.Body.String(), `class="favorite-group" open`) || !strings.Contains(expanded.Body.String(), child["name"].(string)) || !strings.Contains(expanded.Body.String(), "parent-context") || strings.Contains(expanded.Body.String(), "Loading environments") {
		t.Fatalf("expanded topology fragment = %s", expanded.Body.String())
	}
	collapsed := consoleRequest(handler, http.MethodGet, "/console/machines/topology?expansion=none", true)
	assertStatus(t, collapsed, http.StatusOK)
	if strings.Contains(collapsed.Body.String(), `class="provider-group" open`) {
		t.Fatalf("collapsed topology fragment retained open providers = %s", collapsed.Body.String())
	}
	if strings.Contains(collapsed.Body.String(), `class="favorite-group" open`) {
		t.Fatalf("collapsed topology fragment retained open favorites = %s", collapsed.Body.String())
	}

	children := consoleRequest(handler, http.MethodGet, "/console/machines/"+rootID+"/children", true)
	assertStatus(t, children, http.StatusOK)
	if !strings.Contains(children.Body.String(), child["name"].(string)) || !strings.Contains(children.Body.String(), "Hosted on") || !strings.Contains(children.Body.String(), "Badger") {
		t.Fatalf("children fragment = %s", children.Body.String())
	}
	if strings.Contains(children.Body.String(), `class="machine-facts"`) {
		t.Fatal("machine without recorded capacity should not render empty stat tiles")
	}
}

func TestConsoleMemorySizePreservesGuestCapacity(t *testing.T) {
	guestMemory := int64(16390975488)
	if got := consoleMemorySize(&guestMemory); got != "15.3 GiB" {
		t.Fatalf("guest usable memory = %q; want 15.3 GiB", got)
	}
}

func TestConsoleOSIconsMatchRecordedOS(t *testing.T) {
	for _, tc := range []struct {
		name string
		icon string
	}{
		{name: "Ubuntu 24.04.3 LTS", icon: "os-ubuntu.svg"},
		{name: "FreeBSD 15.1", icon: "os-freebsd.svg"},
		{name: "Windows", icon: "os-windows.svg"},
		{name: "illumos", icon: "os-illumos.svg"},
		{name: "OPNsense", icon: "os-opnsense.svg"},
		{name: "Linux", icon: "os-linux.svg"},
		{name: "UnknownOS"},
		{name: "UbuntuLike"},
	} {
		machine := api.TopologyMachine{OperatingSystem: &tc.name}
		want := ""
		if tc.icon != "" {
			want = "/console/static/" + tc.icon
		}
		if got := consoleOSIcon(machine); got != want {
			t.Errorf("OS %q icon = %q; want %q", tc.name, got, want)
		}
	}
	if got := consoleOSIcon(api.TopologyMachine{}); got != "" {
		t.Errorf("missing OS icon = %q; want no icon", got)
	}
}

func TestConsoleOSBaseOnlyForKnownDistributions(t *testing.T) {
	for _, tc := range []struct {
		name, base, icon string
	}{
		{name: "Ubuntu 24.04 LTS", base: "Linux", icon: "os-linux.svg"},
		{name: "OPNsense", base: "FreeBSD", icon: "os-freebsd.svg"},
		{name: "FreeBSD"},
		{name: "Linux"},
		{name: "Windows"},
		{name: "UbuntuLike"},
	} {
		got := consoleOSBase(api.TopologyMachine{OperatingSystem: &tc.name})
		if tc.base == "" {
			if got != nil {
				t.Errorf("OS %q base = %#v; want none", tc.name, got)
			}
			continue
		}
		if got == nil || got.Name != tc.base || got.Icon != "/console/static/"+tc.icon {
			t.Errorf("OS %q base = %#v; want %s / %s", tc.name, got, tc.base, tc.icon)
		}
	}
}

func TestConsoleSSHControlPrefersUserAndDeduplicatesHosts(t *testing.T) {
	hostname := "badger"
	tailnetDNS := "badger.example.ts.net"
	machine := api.TopologyMachine{
		Hostname: &hostname,
		Users: []api.TopologyMachineUser{
			{Username: "guest"},
			{Username: "admin", IsPreferred: true},
		},
		Addresses: []api.TopologyMachineAddress{
			{Address: "100.64.0.10", DnsName: &tailnetDNS, NetworkKind: api.NetworkKindTailnet},
			{Address: "192.168.1.10", DnsName: &hostname, NetworkKind: api.NetworkKindLan},
		},
	}
	control := consoleSSHControlFor(machine)
	wantUsers := []consoleSSHUser{
		{Username: "admin", IsPreferred: true},
		{Username: "guest"},
	}
	wantTargets := []consoleSSHTarget{
		{Host: "badger", Label: "Hostname"},
		{Host: "badger.example.ts.net", Label: "TAILNET DNS"},
		{Host: "100.64.0.10", Label: "TAILNET IP"},
		{Host: "192.168.1.10", Label: "LAN IP"},
	}
	if control == nil || len(control.Users) != len(wantUsers) || len(control.Targets) != len(wantTargets) {
		t.Fatalf("SSH control = %#v; want users %#v and targets %#v", control, wantUsers, wantTargets)
	}
	for index := range wantUsers {
		if control.Users[index] != wantUsers[index] {
			t.Fatalf("SSH user %d = %#v; want %#v", index, control.Users[index], wantUsers[index])
		}
	}
	for index := range wantTargets {
		if control.Targets[index] != wantTargets[index] {
			t.Fatalf("SSH target %d = %#v; want %#v", index, control.Targets[index], wantTargets[index])
		}
	}
	if control := consoleSSHControlFor(api.TopologyMachine{Hostname: &hostname}); control != nil {
		t.Fatalf("SSH control without users = %#v; want nil", control)
	}
}

func TestConsoleRoutesAreIsolatedFromResolver(t *testing.T) {
	handler := newAPITestHandler(t)

	assertStatus(t, consoleRequest(handler, http.MethodGet, "/console", false), http.StatusNotFound)
	assertStatus(t, consoleRequest(handler, http.MethodGet, "/console/", false), http.StatusNotFound)

	missing := consoleRequest(handler, http.MethodGet, "/console/not-a-page", false)
	assertStatus(t, missing, http.StatusNotFound)
	if strings.Contains(missing.Body.String(), "destination not found") {
		t.Fatal("unknown console route fell through to resolver")
	}
	apiDocsRedirect := consoleRequest(handler, http.MethodGet, "/api-docs", false)
	assertStatus(t, apiDocsRedirect, http.StatusPermanentRedirect)
	if apiDocsRedirect.Header().Get("Location") != "/api-docs/" {
		t.Fatalf("API docs redirect = %q", apiDocsRedirect.Header().Get("Location"))
	}
	apiDocs := consoleRequest(handler, http.MethodGet, "/api-docs/", false)
	assertStatus(t, apiDocs, http.StatusOK)
	for _, expected := range []string{"HLIMS API / V1", "/openapi.yaml", "swagger-ui-5.32.15.css", "swagger-ui-bundle-5.32.15.js", "openapi-viewer.js"} {
		if !strings.Contains(apiDocs.Body.String(), expected) {
			t.Errorf("API docs page does not contain %q", expected)
		}
	}
	specification := consoleRequest(handler, http.MethodGet, "/openapi.yaml", false)
	assertStatus(t, specification, http.StatusOK)
	for _, expected := range []string{
		"tags:\n  - {name: Topology",
		"tags: [Machines]\n      operationId: listMachines",
		"tags: [Services]\n      operationId: listServices",
		"tags: [Resolver]\n      operationId: resolveInstance",
	} {
		if !strings.Contains(specification.Body.String(), expected) {
			t.Errorf("OpenAPI specification does not contain resource grouping %q", expected)
		}
	}
	assertStatus(t, consoleRequest(handler, http.MethodGet, "/api-docs/missing", false), http.StatusNotFound)
	removedInspector := consoleRequest(handler, http.MethodGet, "/console/machines/not-a-machine", false)
	assertStatus(t, removedInspector, http.StatusNotFound)

	stylesheet := consoleRequest(handler, http.MethodGet, "/console/static/console.css", false)
	assertStatus(t, stylesheet, http.StatusOK)
	for _, expected := range []string{"--signal:", "--tooltip-bg:", "--tooltip-ink:", ".reachability-tooltip", ".is-unreachable"} {
		if !strings.Contains(stylesheet.Body.String(), expected) {
			t.Errorf("console stylesheet does not contain %q", expected)
		}
	}
	htmx := consoleRequest(handler, http.MethodGet, "/console/static/htmx-4.0.0.min.js", false)
	assertStatus(t, htmx, http.StatusOK)
	if htmx.Body.Len() < 1000 {
		t.Fatalf("vendored htmx response is unexpectedly small: %d bytes", htmx.Body.Len())
	}
	swaggerUI := consoleRequest(handler, http.MethodGet, "/console/static/swagger-ui-bundle-5.32.15.js", false)
	assertStatus(t, swaggerUI, http.StatusOK)
	if swaggerUI.Body.Len() < 1_000_000 {
		t.Fatalf("vendored Swagger UI response is unexpectedly small: %d bytes", swaggerUI.Body.Len())
	}
	viewer := consoleRequest(handler, http.MethodGet, "/console/static/openapi-viewer.js", false)
	assertStatus(t, viewer, http.StatusOK)
	if !strings.Contains(viewer.Body.String(), `url: "/openapi.yaml"`) || !strings.Contains(viewer.Body.String(), "supportedSubmitMethods: []") {
		t.Fatal("OpenAPI viewer is not configured with the local read-only specification")
	}
	reachability := consoleRequest(handler, http.MethodGet, "/console/static/reachability.js", false)
	assertStatus(t, reachability, http.StatusOK)
	for _, expected := range []string{`mode: "no-cors"`, "AbortController", "htmx:after:process", "browser could not reach this route"} {
		if !strings.Contains(reachability.Body.String(), expected) {
			t.Errorf("browser reachability controller does not contain %q", expected)
		}
	}
	copyActions := consoleRequest(handler, http.MethodGet, "/console/static/copy-actions.js", false)
	assertStatus(t, copyActions, http.StatusOK)
	for _, expected := range []string{"navigator.clipboard", "document.execCommand", `closest("[data-copy-value]")`, `closest("[data-ssh-user-option]")`, `closest("[data-ssh-host-option]")`, "updateSSHCommand", "is-copied", "is-copy-error"} {
		if !strings.Contains(copyActions.Body.String(), expected) {
			t.Errorf("copy actions controller does not contain %q", expected)
		}
	}
	theme := consoleRequest(handler, http.MethodGet, "/console/static/theme.js", false)
	assertStatus(t, theme, http.StatusOK)
	if !strings.Contains(theme.Body.String(), "prefers-color-scheme") || !strings.Contains(theme.Body.String(), "hlims-theme") {
		t.Fatal("console theme controller was not served")
	}
	favicon := consoleRequest(handler, http.MethodGet, "/console/static/favicon.svg", false)
	assertStatus(t, favicon, http.StatusOK)
	if !strings.Contains(favicon.Body.String(), `fill="#a599b5"`) {
		t.Fatal("console favicon does not contain the square mark")
	}
}

func consoleRequest(handler http.Handler, method, target string, htmx bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	if htmx {
		request.Header.Set("HX-Request", "true")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
