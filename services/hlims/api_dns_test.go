package hlims

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
)

func TestDNSAndIngressInventorySharesOnePublicAddress(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Digital Ocean"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "San Francisco, United States (SFO3)", "slug": "sfo3", "providerCode": "sfo3"})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "name": "App Droplet", "kind": "virtual_machine",
	})
	network := createAPIResource(t, handler, "/api/v1/networks", map[string]any{"name": "Public Internet", "kind": "public"})
	address := createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": network["publicId"], "machinePublicId": machine["publicId"], "address": "203.0.113.10",
	})
	zone := createAPIResource(t, handler, "/api/v1/dns-zones", map[string]any{"name": "example.com"})
	getRecord := func(name string) map[string]any {
		return createAPIResource(t, handler, "/api/v1/dns-records", map[string]any{
			"zonePublicId": zone["publicId"], "name": name, "kind": "A", "addressPublicId": address["publicId"],
		})
	}
	apex, docs, www := getRecord("@"), getRecord("dbtdocs"), getRecord("www")
	if apex["fqdn"] != "example.com" || docs["fqdn"] != "dbtdocs.example.com" {
		t.Fatalf("DNS records = %#v, %#v", apex, docs)
	}
	service := func(name string) map[string]any {
		return createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": name})
	}
	instance := func(name string, service map[string]any, port int) map[string]any {
		return createAPIResource(t, handler, "/api/v1/instances", map[string]any{
			"name": name, "servicePublicId": service["publicId"], "machinePublicId": machine["publicId"], "port": port,
		})
	}
	ingress := instance("Ingress", service("Caddy"), 443)
	app := instance("Production", service("FastAPI"), 8000)
	docSite := instance("Production", service("dbt Docs"), 443)
	for _, item := range []struct {
		name, kind, target string
		instance           map[string]any
		record             map[string]any
	}{
		{"Website", "proxy", "backend:8000", app, apex},
		{"Documentation", "static", "/srv/www/dbtdocs", docSite, docs},
		{"WWW redirect", "redirect", "https://example.com{uri}", ingress, www},
	} {
		endpoint := createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
			"instancePublicId": item.instance["publicId"], "addressPublicId": address["publicId"],
			"dnsRecordPublicId": item.record["publicId"], "name": item.name,
			"scheme": "https", "port": 443, "isPreferred": item.kind != "redirect" || item.name == "WWW redirect",
		})
		createAPIResource(t, handler, "/api/v1/ingress-routes", map[string]any{
			"endpointPublicId": endpoint["publicId"], "ingressInstancePublicId": ingress["publicId"],
			"kind": item.kind, "target": item.target,
		})
	}
	response := apiRequest(t, handler, http.MethodGet, "/api/v1/topology", nil, "")
	assertStatus(t, response, http.StatusOK)
	var topology api.Topology
	if err := json.Unmarshal(response.Body.Bytes(), &topology); err != nil {
		t.Fatal(err)
	}
	services := topology.Providers[0].Areas[0].Machines[0].Services
	if len(services) != 3 {
		t.Fatalf("Machine Topology should show Caddy and the two applications: %#v", services)
	}
	urls := make(map[string]api.TopologyInstanceEndpoint)
	for _, service := range services {
		if service.Name == "Caddy" && len(service.Instances[0].Endpoints) != 1 {
			t.Fatalf("Caddy redirect should remain attached to its ingress instance: %#v", service)
		}
		for _, instance := range service.Instances {
			for _, endpoint := range instance.Endpoints {
				urls[endpoint.Url] = endpoint
			}
		}
	}
	if len(urls) != 3 || urls["https://example.com/"].Ingress == nil || urls["https://example.com/"].Ingress.Kind != "proxy" || urls["https://dbtdocs.example.com/"].Ingress == nil || urls["https://dbtdocs.example.com/"].Ingress.Kind != "static" || urls["https://www.example.com/"].Ingress == nil || urls["https://www.example.com/"].Ingress.Kind != "redirect" {
		t.Fatalf("public URLs and ingress routes = %#v", urls)
	}
	page := consoleRequest(handler, http.MethodGet, "/console/machines/", false)
	assertStatus(t, page, http.StatusOK)
	for _, want := range []string{"example.com", "dbtdocs.example.com", "www.example.com", "Caddy", "FastAPI", "dbt Docs"} {
		if !strings.Contains(page.Body.String(), want) {
			t.Errorf("console missing %q", want)
		}
	}
	if strings.Contains(page.Body.String(), "DNS / Owned domains") || strings.Contains(page.Body.String(), "domain-inventory") {
		t.Fatal("Machine Topology should not include the DNS inventory page")
	}
	if !strings.Contains(page.Body.String(), "<h5>Caddy</h5>") {
		t.Fatal("Caddy's clickable redirect should appear on its service card")
	}
}

func TestLoopbackInstanceEndpoint(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Local"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "Framework"})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "name": "WSL", "kind": "virtual_machine",
	})
	network := createAPIResource(t, handler, "/api/v1/networks", map[string]any{"name": "WSL loopback", "kind": "loopback"})
	address := createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": network["publicId"], "machinePublicId": machine["publicId"], "address": "127.0.0.1",
	})
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Local UI"})
	instance := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": service["publicId"], "machinePublicId": machine["publicId"], "name": "Staging", "port": 8081,
	})
	createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": instance["publicId"], "addressPublicId": address["publicId"],
		"name": "localhost", "scheme": "http", "port": 8081, "hostType": "ip", "isPreferred": true,
	})
	tailnet := createAPIResource(t, handler, "/api/v1/networks", map[string]any{"name": "Tailnet", "kind": "tailnet"})
	tailAddress := createAPIResource(t, handler, "/api/v1/addresses", map[string]any{
		"networkPublicId": tailnet["publicId"], "machinePublicId": machine["publicId"],
		"address": "100.64.0.10", "dnsName": "wsl.example.ts.net",
	})
	serveService := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Tailscale Serve"})
	serve := createAPIResource(t, handler, "/api/v1/instances", map[string]any{
		"servicePublicId": serveService["publicId"], "machinePublicId": machine["publicId"],
		"name": "Framework WSL", "port": 443,
	})
	served := createAPIResource(t, handler, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": instance["publicId"], "addressPublicId": tailAddress["publicId"],
		"name": "Tailnet HTTPS", "scheme": "https", "port": 443, "hostType": "dns", "isPreferred": true,
	})
	createAPIResource(t, handler, "/api/v1/ingress-routes", map[string]any{
		"endpointPublicId": served["publicId"], "ingressInstancePublicId": serve["publicId"],
		"kind": "proxy", "target": "http://127.0.0.1:8081",
	})
	response := apiRequest(t, handler, http.MethodGet, "/api/v1/topology", nil, "")
	assertStatus(t, response, http.StatusOK)
	var topology api.Topology
	if err := json.Unmarshal(response.Body.Bytes(), &topology); err != nil {
		t.Fatal(err)
	}
	endpoints := topology.Providers[0].Areas[0].Machines[0].Services[0].Instances[0].Endpoints
	services := topology.Providers[0].Areas[0].Machines[0].Services
	if len(services) != 2 || services[1].Name != "Tailscale Serve" || len(services[1].Instances) != 1 || services[1].Instances[0].ResolverPath != nil {
		t.Fatalf("ingress-only workload should be inventoried without a dead link: %#v", services)
	}
	if len(endpoints) != 2 || endpoints[0].Url != "https://wsl.example.ts.net/" || !endpoints[0].IsPreferred || endpoints[0].Ingress == nil || endpoints[0].Ingress.IngressServiceName != "Tailscale Serve" || endpoints[1].Url != "http://127.0.0.1:8081/" || endpoints[1].NetworkKind != "loopback" {
		t.Fatalf("Tailscale Serve and loopback endpoints = %#v", endpoints)
	}
	page := consoleRequest(handler, http.MethodGet, "/console/machines/", false)
	if !strings.Contains(page.Body.String(), "<h5>Tailscale Serve</h5>") || !strings.Contains(page.Body.String(), "Tracked · no URL") || !strings.Contains(page.Body.String(), "Tailscale Serve · proxy") {
		t.Fatal("Tailscale Serve should appear as a tracked ingress workload without a direct URL")
	}
}
