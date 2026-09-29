package hlims

import (
	"net/http"
	"strings"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
)

func TestMachineWorkerNeedsNeitherPortNorURL(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Example"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "Home"})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "name": "worker-host", "kind": "virtual_machine",
	})
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Example replicator"})
	withoutPort := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": service["publicId"], "machinePublicId": machine["publicId"], "name": "worker",
	})
	if withoutPort["port"] != nil || withoutPort["machinePublicId"] != machine["publicId"] {
		t.Fatalf("worker without a listener = %#v", withoutPort)
	}
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": service["publicId"], "machinePublicId": machine["publicId"], "name": "bad-port", "port": 0,
	}, "application/json"), http.StatusBadRequest)
	topology := apiRequest(t, handler, http.MethodGet, "/api/v1/topology", nil, "")
	assertStatus(t, topology, http.StatusOK)
	if !strings.Contains(topology.Body.String(), withoutPort["publicId"].(string)) {
		t.Fatal("worker without URL missing from topology")
	}
	var item api.Topology
	if err := decodeJSON("application/json", topology.Body, &item); err != nil {
		t.Fatal(err)
	}
	worker := item.Providers[0].Areas[0].Machines[0].Services[0].Instances[0]
	if worker.Port != nil || worker.ResolverPath != nil || len(worker.Endpoints) != 0 {
		t.Fatalf("worker without listener has a fake port or URL: %#v", worker)
	}
	machines := consoleRequest(handler, http.MethodGet, "/console/machines/", false)
	assertStatus(t, machines, http.StatusOK)
	if !strings.Contains(machines.Body.String(), "Example replicator") || !strings.Contains(machines.Body.String(), "Tracked · no URL") || strings.Contains(machines.Body.String(), "<code>:0</code>") {
		t.Fatalf("machine card does not show the worker correctly: %s", machines.Body.String())
	}
	services := consoleRequest(handler, http.MethodGet, "/console/services/", false)
	assertStatus(t, services, http.StatusOK)
	if !strings.Contains(services.Body.String(), "Tracked · no URL recorded") || !strings.Contains(services.Body.String(), "Internal service · no URL recorded") {
		t.Fatalf("service catalog omits an internal worker: %s", services.Body.String())
	}
	assertStatus(t, apiRequest(t, handler, http.MethodGet, "/api/v1/resolve/example-replicator/worker", nil, ""), http.StatusNotFound)
}
