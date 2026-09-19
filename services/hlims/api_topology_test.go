package hlims

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func TestTopologyEndpointBuildsSortedRecursiveSnapshot(t *testing.T) {
	handler := newAPITestHandler(t)
	_ = createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Zulu Provider"})
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Alpha Provider"})
	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+provider["publicId"].(string)+"/logo", []byte("\x89PNG\r\n\x1a\nPROVIDER"), "image/png"), http.StatusNoContent)
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": provider["publicId"], "name": "Main Area",
	})
	root := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"],
		"areaPublicId":            area["publicId"],
		"name":                    "Root Host",
		"kind":                    "bare_metal",
		"isFavorite":              true,
		"hostname":                "root.example.com",
		"operatingSystem":         "Linux",
		"operatingSystemVersion":  "42",
		"kernel":                  "6.12",
		"architecture":            "x86_64",
		"cpuCount":                16,
		"cpuAllocation":           "dedicated",
		"cpuVendor":               "AMD",
		"memoryBytes":             68719476736,
		"storageBytes":            2000000000000,
		"storageMediaKind":        "ssd",
		"storageInterfaceKind":    "nvme",
	})
	createAPIResource(t, handler, "/api/v1/machine-users", map[string]any{
		"machinePublicId": root["publicId"], "username": "zulu", "notes": "Fallback",
	})
	createAPIResource(t, handler, "/api/v1/machine-users", map[string]any{
		"machinePublicId": root["publicId"], "username": "alpha", "isPreferred": true,
	})
	lan := createAPIResource(t, handler, "/api/v1/networks", map[string]any{"name": "LAN", "kind": "lan"})
	public := createAPIResource(t, handler, "/api/v1/networks", map[string]any{"name": "Public", "kind": "public"})
	createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": lan["publicId"], "machinePublicId": root["publicId"], "address": "192.0.2.10", "dnsName": "root.lan",
	})
	createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": public["publicId"], "machinePublicId": root["publicId"], "address": "203.0.113.10", "isPrimary": true,
	})
	for _, name := range []string{"Zulu VM", "Alpha VM"} {
		createAPIResource(t, handler, "/api/v1/machines", map[string]any{
			"machineProviderPublicId": provider["publicId"],
			"areaPublicId":            area["publicId"],
			"parentMachinePublicId":   root["publicId"],
			"name":                    name,
			"kind":                    "virtual_machine",
			"virtualizationPlatform":  "KVM",
		})
	}
	zuluService := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Zulu Service"})
	alphaService := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Alpha Service", "description": "Dashboard"})
	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/services/"+alphaService["publicId"].(string)+"/logo", []byte("\x89PNG\r\n\x1a\nHLIMS"), "image/png"), http.StatusNoContent)
	for _, item := range []struct {
		service map[string]any
		name    string
		port    int
	}{
		{service: zuluService, name: "Production", port: 8080},
		{service: alphaService, name: "Zulu", port: 443},
		{service: alphaService, name: "Alpha", port: 8443},
	} {
		createAPIResource(t, handler, "/api/v1/instances", map[string]any{
			"servicePublicId": item.service["publicId"],
			"machinePublicId": root["publicId"],
			"name":            item.name,
			"port":            item.port,
		})
	}

	response := apiRequest(t, handler, http.MethodGet, "/api/v1/topology", nil, "")
	assertStatus(t, response, http.StatusOK)
	var topology api.Topology
	if err := json.Unmarshal(response.Body.Bytes(), &topology); err != nil {
		t.Fatal(err)
	}
	var shape map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &shape); err != nil {
		t.Fatal(err)
	}
	if len(shape) != 1 || shape["providers"] == nil {
		t.Fatalf("topology shape = %#v; want providers only", shape)
	}
	if len(topology.Providers) != 2 || topology.Providers[0].Name != "Alpha Provider" || topology.Providers[1].Name != "Zulu Provider" {
		t.Fatalf("providers = %#v", topology.Providers)
	}
	if !topology.Providers[0].HasLogo || topology.Providers[1].HasLogo {
		t.Fatalf("provider logo flags = %#v", topology.Providers)
	}
	machine := topology.Providers[0].Areas[0].Machines[0]
	if !machine.IsFavorite || machine.Hostname == nil || *machine.Hostname != "root.example.com" || machine.CpuCount == nil || *machine.CpuCount != 16 {
		t.Fatalf("machine summary = %#v", machine)
	}
	if len(machine.Children) != 2 || machine.Children[0].Name != "Alpha VM" || machine.Children[1].Name != "Zulu VM" {
		t.Fatalf("children = %#v", machine.Children)
	}
	if len(machine.Users) != 2 || machine.Users[0].Username != "alpha" || !machine.Users[0].IsPreferred || machine.Users[1].Username != "zulu" {
		t.Fatalf("machine users = %#v", machine.Users)
	}
	if len(machine.Addresses) != 2 || machine.Addresses[0].NetworkKind != api.NetworkKindPublic || !machine.Addresses[0].IsPrimary || machine.Addresses[1].NetworkKind != api.NetworkKindLan {
		t.Fatalf("machine addresses = %#v", machine.Addresses)
	}
	if len(machine.Services) != 2 || machine.Services[0].Name != "Alpha Service" || !machine.Services[0].HasLogo {
		t.Fatalf("services = %#v", machine.Services)
	}
	instances := machine.Services[0].Instances
	if len(instances) != 2 || instances[0].Name != "Alpha" || instances[0].ResolverPath != "/root-host/alpha-service/alpha" {
		t.Fatalf("instances = %#v", instances)
	}
	if response.Body.String() == "" || containsAny(response.Body.String(), "imageData", "iVBOR") {
		t.Fatalf("topology exposed logo bytes: %s", response.Body.String())
	}
}

func TestBuildTopologyRejectsBrokenMachineHierarchy(t *testing.T) {
	providerRows := []database.ListMachineProvidersRow{{PublicID: "provider0001", Name: "Provider", Slug: "provider"}}
	areaRows := []database.ListAreasRow{{PublicID: "area00000001", MachineProviderPublicID: "provider0001", Name: "Area", Slug: "area"}}
	base := database.ListMachineDetailsRow{
		PublicID: "machine00001", MachineProviderPublicID: "provider0001", AreaPublicID: "area00000001",
		Name: "Machine", Slug: "machine", Kind: "bare_metal",
	}

	t.Run("missing parent", func(t *testing.T) {
		row := base
		row.ParentMachinePublicID = sql.NullString{String: "missing00001", Valid: true}
		if _, err := buildTopology(providerRows, areaRows, []database.ListMachineDetailsRow{row}, nil, nil, nil, nil); err == nil {
			t.Fatal("buildTopology accepted a missing parent")
		}
	})
	t.Run("cycle", func(t *testing.T) {
		first := base
		first.ParentMachinePublicID = sql.NullString{String: "machine00002", Valid: true}
		second := base
		second.PublicID, second.Name, second.Slug = "machine00002", "Machine 2", "machine-2"
		second.ParentMachinePublicID = sql.NullString{String: "machine00001", Valid: true}
		if _, err := buildTopology(providerRows, areaRows, []database.ListMachineDetailsRow{first, second}, nil, nil, nil, nil); err == nil {
			t.Fatal("buildTopology accepted a cycle")
		}
	})
	t.Run("machine user missing machine", func(t *testing.T) {
		users := []database.ListMachineUsersRow{{PublicID: "user00000001", MachinePublicID: "missing00001", Username: "tyler"}}
		if _, err := buildTopology(providerRows, areaRows, []database.ListMachineDetailsRow{base}, users, nil, nil, nil); err == nil {
			t.Fatal("buildTopology accepted a machine user with a missing machine")
		}
	})
	t.Run("invalid machine user", func(t *testing.T) {
		users := []database.ListMachineUsersRow{{PublicID: "user00000001", MachinePublicID: base.PublicID, Username: "-root"}}
		if _, err := buildTopology(providerRows, areaRows, []database.ListMachineDetailsRow{base}, users, nil, nil, nil); err == nil {
			t.Fatal("buildTopology accepted an invalid machine username")
		}
	})
	t.Run("malformed machine address", func(t *testing.T) {
		addresses := []database.ListTopologyMachineAddressesRow{{
			PublicID: "address00001", MachineID: sql.NullString{String: "internal", Valid: true}, MachinePublicID: base.PublicID,
			NetworkPublicID: "network00001", NetworkKind: "invalid", Address: "192.0.2.1",
		}}
		if _, err := buildTopology(providerRows, areaRows, []database.ListMachineDetailsRow{base}, nil, addresses, nil, nil); err == nil {
			t.Fatal("buildTopology accepted an invalid address network kind")
		}
	})
}

func TestTopologyAvailableVia(t *testing.T) {
	t.Parallel()

	if routes := topologyAvailableVia(false, false); routes == nil || len(routes) != 0 {
		t.Fatalf("empty routes = %#v; want non-nil empty slice", routes)
	}
	routes := topologyAvailableVia(true, true)
	if len(routes) != 2 || routes[0] != api.ViaLan || routes[1] != api.ViaTailnet {
		t.Fatalf("routes = %#v; want LAN then Tailnet", routes)
	}
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}
