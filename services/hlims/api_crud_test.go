package hlims

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizedCRUDAndResolver(t *testing.T) {
	db, err := NewSQLiteDB(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	handler := newHandler(db)

	missingArea := apiRequest(t, handler, http.MethodPost, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": "000000000000", "name": "Home",
	}, "application/json")
	assertStatus(t, missingArea, http.StatusNotFound)

	providerResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/machine-providers", map[string]any{
		"name": "  My Provider  ", "slug": " My_PROVIDER ",
	}, "application/json")
	assertStatus(t, providerResponse, http.StatusCreated)
	provider := decodeObject(t, providerResponse)
	providerID := provider["publicId"].(string)
	if provider["name"] != "My Provider" || provider["slug"] != "my-provider" {
		t.Fatalf("normalized provider = %#v", provider)
	}
	if strings.Contains(providerResponse.Body.String(), `"id"`) {
		t.Fatal("response exposed an internal ID")
	}
	if providerResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("API response included a CORS header")
	}

	conflict := apiRequest(t, handler, http.MethodPost, "/api/v1/machine-providers", map[string]any{"name": "my provider"}, "application/json")
	assertStatus(t, conflict, http.StatusConflict)

	updatedProvider := apiRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+providerID, map[string]any{"name": "Renamed Provider"}, "application/json")
	assertStatus(t, updatedProvider, http.StatusOK)
	if got := decodeObject(t, updatedProvider)["slug"]; got != "my-provider" {
		t.Fatalf("slug after omitted PUT slug = %v", got)
	}

	areaResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": providerID, "name": " Main Room ",
	}, "application/json")
	assertStatus(t, areaResponse, http.StatusCreated)
	areaID := decodeObject(t, areaResponse)["publicId"].(string)

	otherProviderResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/machine-providers", map[string]any{"name": "Other"}, "application/json")
	assertStatus(t, otherProviderResponse, http.StatusCreated)
	otherProviderID := decodeObject(t, otherProviderResponse)["publicId"].(string)
	mismatchedMachine := apiRequest(t, handler, http.MethodPost, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": otherProviderID, "areaPublicId": areaID, "name": "Bad", "kind": "bare_metal",
	}, "application/json")
	assertStatus(t, mismatchedMachine, http.StatusBadRequest)

	machineResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "name": " Badger ", "kind": "bare_metal", "hostname": "BADGER.EXAMPLE.COM.",
	}, "application/json")
	assertStatus(t, machineResponse, http.StatusCreated)
	machine := decodeObject(t, machineResponse)
	machineID := machine["publicId"].(string)
	if machine["slug"] != "badger" || machine["hostname"] != "badger.example.com" {
		t.Fatalf("normalized machine = %#v", machine)
	}

	networkResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/networks", map[string]any{
		"areaPublicId": areaID, "name": "Home LAN", "kind": "lan", "cidr": "192.168.1.42/24",
	}, "application/json")
	assertStatus(t, networkResponse, http.StatusCreated)
	network := decodeObject(t, networkResponse)
	networkID := network["publicId"].(string)
	if network["cidr"] != "192.168.1.0/24" {
		t.Fatalf("canonical CIDR = %v", network["cidr"])
	}

	addressResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/addresses", map[string]any{
		"networkPublicId": networkID, "machinePublicId": machineID, "address": "192.168.1.010", "dnsName": "BADGER.LAN.",
	}, "application/json")
	assertStatus(t, addressResponse, http.StatusBadRequest)
	addressResponse = apiRequest(t, handler, http.MethodPost, "/api/v1/addresses", map[string]any{
		"networkPublicId": networkID, "machinePublicId": machineID, "address": "192.168.1.10", "dnsName": "BADGER.LAN.", "isPrimary": true,
	}, "application/json")
	assertStatus(t, addressResponse, http.StatusCreated)
	address := decodeObject(t, addressResponse)
	addressID := address["publicId"].(string)
	if address["dnsName"] != "badger.lan" {
		t.Fatalf("normalized DNS name = %v", address["dnsName"])
	}

	serviceResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/services", map[string]any{"name": "OpenCode"}, "application/json")
	assertStatus(t, serviceResponse, http.StatusCreated)
	serviceID := decodeObject(t, serviceResponse)["publicId"].(string)
	instanceResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/instances", map[string]any{
		"servicePublicId": serviceID, "machinePublicId": machineID, "name": "Production", "port": 4096,
	}, "application/json")
	assertStatus(t, instanceResponse, http.StatusCreated)
	instanceID := decodeObject(t, instanceResponse)["publicId"].(string)

	firstEndpointResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": instanceID, "addressPublicId": addressID, "name": "Direct", "scheme": "HTTP", "port": 4096, "basePath": "app", "isPreferred": true,
	}, "application/json")
	assertStatus(t, firstEndpointResponse, http.StatusCreated)
	firstEndpointID := decodeObject(t, firstEndpointResponse)["publicId"].(string)
	secondEndpointResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/instance-endpoints", map[string]any{
		"instancePublicId": instanceID, "addressPublicId": addressID, "name": "TLS", "scheme": "https", "port": 443,
	}, "application/json")
	assertStatus(t, secondEndpointResponse, http.StatusCreated)
	secondEndpointID := decodeObject(t, secondEndpointResponse)["publicId"].(string)
	secondEndpointResponse = apiRequest(t, handler, http.MethodPut, "/api/v1/instance-endpoints/"+secondEndpointID, map[string]any{
		"instancePublicId": instanceID, "addressPublicId": addressID, "name": "TLS", "scheme": "https", "port": 443, "isPreferred": true,
	}, "application/json")
	assertStatus(t, secondEndpointResponse, http.StatusOK)
	firstEndpoint := apiRequest(t, handler, http.MethodGet, "/api/v1/instance-endpoints/"+firstEndpointID, nil, "")
	assertStatus(t, firstEndpoint, http.StatusOK)
	if decodeObject(t, firstEndpoint)["isPreferred"] != false {
		t.Fatal("creating a new preferred endpoint did not demote the old endpoint")
	}

	resolved := apiRequest(t, handler, http.MethodGet, "/api/v1/resolve/badger/opencode/production", nil, "")
	assertStatus(t, resolved, http.StatusOK)
	if got := decodeObject(t, resolved)["url"]; got != "https://badger.lan/" {
		t.Fatalf("resolved URL = %v", got)
	}
	redirect := apiRequest(t, handler, http.MethodGet, "/badger/opencode/production/dashboard", nil, "")
	assertStatus(t, redirect, http.StatusFound)
	if redirect.Header().Get("Location") != "https://badger.lan/dashboard" {
		t.Fatalf("redirect Location = %q", redirect.Header().Get("Location"))
	}

	manufacturerResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/manufacturers", map[string]any{"name": "  Hewlett Packard  "}, "application/json")
	assertStatus(t, manufacturerResponse, http.StatusCreated)
	manufacturerID := decodeObject(t, manufacturerResponse)["publicId"].(string)
	assertStatus(t, apiRequest(t, handler, http.MethodGet, "/api/v1/manufacturers/"+manufacturerID, nil, ""), http.StatusOK)
	manufacturerResponse = apiRequest(t, handler, http.MethodPut, "/api/v1/manufacturers/"+manufacturerID, map[string]any{"name": "HP", "slug": "H-P"}, "application/json")
	assertStatus(t, manufacturerResponse, http.StatusOK)
	if decodeObject(t, manufacturerResponse)["slug"] != "h-p" {
		t.Fatal("manufacturer PUT did not normalize slug")
	}
	assertStatus(t, apiRequest(t, handler, http.MethodDelete, "/api/v1/manufacturers/"+manufacturerID, nil, ""), http.StatusNoContent)
	assertStatus(t, apiRequest(t, handler, http.MethodGet, "/api/v1/manufacturers/"+manufacturerID, nil, ""), http.StatusNotFound)

	unsupported := apiRequest(t, handler, http.MethodPost, "/api/v1/services", map[string]any{"name": "Wrong type"}, "text/plain")
	assertStatus(t, unsupported, http.StatusUnsupportedMediaType)
}

func TestProductAggregateAPI(t *testing.T) {
	handler := newAPITestHandler(t)
	manufacturer := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Advanced Micro Devices"})
	manufacturerID := manufacturer["publicId"].(string)

	created := apiRequest(t, handler, http.MethodPost, "/api/v1/products", map[string]any{
		"manufacturerPublicId": manufacturerID,
		"kind":                 "processor",
		"name":                 " Ryzen 9 7950X ",
		"partNumber":           " 100-100000514WOF ",
		"notes":                " Desktop CPU ",
		"processorSpec": map[string]any{
			"coreCount": 16, "threadCount": 32, "baseClockMhz": 4500, "virtualization": " AMD-V ",
		},
	}, "application/json")
	assertStatus(t, created, http.StatusCreated)
	product := decodeObject(t, created)
	productID := product["publicId"].(string)
	if product["name"] != "Ryzen 9 7950X" || product["partNumber"] != "100-100000514WOF" || product["notes"] != "Desktop CPU" {
		t.Fatalf("normalized product = %#v", product)
	}
	processorSpec := product["processorSpec"].(map[string]any)
	if processorSpec["coreCount"] != float64(16) || processorSpec["threadCount"] != float64(32) || processorSpec["baseClockMhz"] != float64(4500) || processorSpec["virtualization"] != "AMD-V" {
		t.Fatalf("processor spec = %#v", processorSpec)
	}

	invalidUpdate := apiRequest(t, handler, http.MethodPut, "/api/v1/products/"+productID, map[string]any{
		"manufacturerPublicId": manufacturerID,
		"kind":                 "memory",
		"name":                 "Invalid replacement",
		"memorySpec":           map[string]any{"capacityBytes": 34359738368, "memoryType": "DDR5"},
		"driveSpec":            map[string]any{"capacityBytes": 1000000000, "mediaKind": "ssd"},
	}, "application/json")
	assertStatus(t, invalidUpdate, http.StatusBadRequest)
	unchanged := apiRequest(t, handler, http.MethodGet, "/api/v1/products/"+productID, nil, "")
	assertStatus(t, unchanged, http.StatusOK)
	unchangedProduct := decodeObject(t, unchanged)
	if unchangedProduct["kind"] != "processor" || unchangedProduct["name"] != "Ryzen 9 7950X" {
		t.Fatalf("product changed after rejected update = %#v", unchangedProduct)
	}
	if got := unchangedProduct["processorSpec"].(map[string]any)["threadCount"]; got != float64(32) {
		t.Fatalf("processor spec changed after rejected update: threadCount = %v", got)
	}

	updated := apiRequest(t, handler, http.MethodPut, "/api/v1/products/"+productID, map[string]any{
		"manufacturerPublicId": manufacturerID,
		"kind":                 "drive",
		"name":                 "NVMe Drive",
		"driveSpec": map[string]any{
			"capacityBytes": 2000000000000, "mediaKind": "ssd", "interfaceKind": "nvme",
		},
	}, "application/json")
	assertStatus(t, updated, http.StatusOK)
	updatedProduct := decodeObject(t, updated)
	driveSpec := updatedProduct["driveSpec"].(map[string]any)
	if updatedProduct["kind"] != "drive" || driveSpec["capacityBytes"] != float64(2000000000000) || driveSpec["mediaKind"] != "ssd" || driveSpec["interfaceKind"] != "nvme" {
		t.Fatalf("updated product = %#v", updatedProduct)
	}
	if _, exists := updatedProduct["processorSpec"]; exists {
		t.Fatalf("updated product retained processorSpec = %#v", updatedProduct)
	}

	listed := apiRequest(t, handler, http.MethodGet, "/api/v1/products", nil, "")
	assertStatus(t, listed, http.StatusOK)
	items := decodeObject(t, listed)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["driveSpec"].(map[string]any)["interfaceKind"] != "nvme" {
		t.Fatalf("listed products = %#v", items)
	}
}

func TestMachineUserCRUDValidationAndPreference(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Local"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "Home"})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"], "name": "Badger", "kind": "bare_metal",
	})

	invalid := apiRequest(t, handler, http.MethodPost, "/api/v1/machine-users", map[string]any{
		"machinePublicId": machine["publicId"], "username": "-unsafe",
	}, "application/json")
	assertStatus(t, invalid, http.StatusBadRequest)
	missingMachine := apiRequest(t, handler, http.MethodPost, "/api/v1/machine-users", map[string]any{
		"machinePublicId": "000000000000", "username": "tyler",
	}, "application/json")
	assertStatus(t, missingMachine, http.StatusNotFound)

	first := createAPIResource(t, handler, "/api/v1/machine-users", map[string]any{
		"machinePublicId": machine["publicId"], "username": " tyler ", "isPreferred": true, "notes": "Operator",
	})
	if first["username"] != "tyler" || first["isPreferred"] != true || first["notes"] != "Operator" {
		t.Fatalf("created machine user = %#v", first)
	}
	duplicate := apiRequest(t, handler, http.MethodPost, "/api/v1/machine-users", map[string]any{
		"machinePublicId": machine["publicId"], "username": "tyler",
	}, "application/json")
	assertStatus(t, duplicate, http.StatusConflict)

	second := createAPIResource(t, handler, "/api/v1/machine-users", map[string]any{
		"machinePublicId": machine["publicId"], "username": "deploy_user", "isPreferred": true,
	})
	firstResponse := apiRequest(t, handler, http.MethodGet, "/api/v1/machine-users/"+first["publicId"].(string), nil, "")
	assertStatus(t, firstResponse, http.StatusOK)
	if decodeObject(t, firstResponse)["isPreferred"] != false {
		t.Fatal("new preferred machine user did not demote the old user")
	}

	updated := apiRequest(t, handler, http.MethodPut, "/api/v1/machine-users/"+first["publicId"].(string), map[string]any{
		"machinePublicId": machine["publicId"], "username": "admin.1", "isPreferred": true,
	}, "application/json")
	assertStatus(t, updated, http.StatusOK)
	if decodeObject(t, updated)["username"] != "admin.1" {
		t.Fatalf("updated machine user = %s", updated.Body.String())
	}
	secondResponse := apiRequest(t, handler, http.MethodGet, "/api/v1/machine-users/"+second["publicId"].(string), nil, "")
	if decodeObject(t, secondResponse)["isPreferred"] != false {
		t.Fatal("updated preferred machine user did not demote the old user")
	}

	listed := apiRequest(t, handler, http.MethodGet, "/api/v1/machine-users", nil, "")
	assertStatus(t, listed, http.StatusOK)
	if items := decodeObject(t, listed)["items"].([]any); len(items) != 2 {
		t.Fatalf("listed machine users = %#v", items)
	}
	assertStatus(t, apiRequest(t, handler, http.MethodDelete, "/api/v1/machine-users/"+second["publicId"].(string), nil, ""), http.StatusNoContent)
	assertStatus(t, apiRequest(t, handler, http.MethodGet, "/api/v1/machine-users/"+second["publicId"].(string), nil, ""), http.StatusNotFound)
}

func TestAssetPlacementAPI(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Local"})
	providerID := provider["publicId"].(string)
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": providerID, "name": "Rack Room",
	})
	areaID := area["publicId"].(string)
	manufacturer := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Framework"})
	manufacturerID := manufacturer["publicId"].(string)
	systemProduct := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": manufacturerID, "kind": "system", "name": "Chassis",
	})
	systemProductID := systemProduct["publicId"].(string)
	memoryProduct := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": manufacturerID, "kind": "memory", "name": "DIMM",
		"memorySpec": map[string]any{"capacityBytes": 17179869184, "memoryType": "DDR5", "speedMts": 5600},
	})
	memoryProductID := memoryProduct["publicId"].(string)

	rack := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": systemProductID, "name": "Rack", "placement": map[string]any{"type": "area", "areaPublicId": areaID},
	})
	rackID := rack["publicId"].(string)
	if rack["effectiveAreaPublicId"] != areaID || rack["placement"].(map[string]any)["areaPublicId"] != areaID {
		t.Fatalf("area-placed asset = %#v", rack)
	}

	dimm := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": memoryProductID, "name": "DIMM 1", "placement": map[string]any{"type": "asset", "parentAssetPublicId": rackID},
	})
	dimmID := dimm["publicId"].(string)
	if dimm["effectiveAreaPublicId"] != areaID || dimm["placement"].(map[string]any)["parentAssetPublicId"] != rackID {
		t.Fatalf("nested asset = %#v", dimm)
	}

	unplaced := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": systemProductID, "name": "Spare Chassis", "placement": map[string]any{"type": "unplaced"},
	})
	if unplaced["placement"].(map[string]any)["type"] != "unplaced" {
		t.Fatalf("unplaced asset = %#v", unplaced)
	}
	if _, exists := unplaced["effectiveAreaPublicId"]; exists {
		t.Fatalf("unplaced asset has effective area = %#v", unplaced)
	}

	invalidParent := apiRequest(t, handler, http.MethodPost, "/api/v1/assets", map[string]any{
		"productPublicId": memoryProductID, "placement": map[string]any{"type": "asset", "parentAssetPublicId": dimmID},
	}, "application/json")
	assertStatus(t, invalidParent, http.StatusBadRequest)

	chassis := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": systemProductID, "name": "Nested Chassis", "placement": map[string]any{"type": "asset", "parentAssetPublicId": rackID},
	})
	cycle := apiRequest(t, handler, http.MethodPut, "/api/v1/assets/"+rackID, map[string]any{
		"productPublicId": systemProductID, "name": "Rack", "placement": map[string]any{"type": "asset", "parentAssetPublicId": chassis["publicId"]},
	}, "application/json")
	assertStatus(t, cycle, http.StatusConflict)
	rackAfterCycle := apiRequest(t, handler, http.MethodGet, "/api/v1/assets/"+rackID, nil, "")
	assertStatus(t, rackAfterCycle, http.StatusOK)
	if got := decodeObject(t, rackAfterCycle)["placement"].(map[string]any)["type"]; got != "area" {
		t.Fatalf("rack placement after rejected cycle = %v", got)
	}
}

func TestExpandedMachineAPI(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Local"})
	providerID := provider["publicId"].(string)
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": providerID, "name": "Primary",
	})
	areaID := area["publicId"].(string)
	otherArea := createAPIResource(t, handler, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": providerID, "name": "Secondary",
	})
	otherAreaID := otherArea["publicId"].(string)
	manufacturer := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Supermicro"})
	manufacturerID := manufacturer["publicId"].(string)
	systemProduct := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": manufacturerID, "kind": "system", "name": "Server",
	})
	systemProductID := systemProduct["publicId"].(string)
	memoryProduct := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": manufacturerID, "kind": "memory", "name": "Memory",
		"memorySpec": map[string]any{"capacityBytes": 8589934592, "memoryType": "DDR4"},
	})
	systemAsset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": systemProductID, "placement": map[string]any{"type": "area", "areaPublicId": areaID},
	})
	systemAssetID := systemAsset["publicId"].(string)
	nonSystemAsset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": memoryProduct["publicId"], "placement": map[string]any{"type": "area", "areaPublicId": areaID},
	})
	unplacedAsset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": systemProductID, "placement": map[string]any{"type": "unplaced"},
	})

	hostBody := map[string]any{
		"machineProviderPublicId":   providerID,
		"areaPublicId":              areaID,
		"assetPublicId":             systemAssetID,
		"name":                      " Atlas Host ",
		"kind":                      "bare_metal",
		"hostname":                  "ATLAS.EXAMPLE.COM.",
		"osMachineId":               "AABBCCDD",
		"operatingSystem":           " Debian ",
		"operatingSystemVersion":    "13",
		"kernel":                    "6.12.1",
		"architecture":              "X86_64",
		"cpuCount":                  16,
		"cpuAllocation":             "dedicated",
		"cpuVendor":                 " AMD ",
		"memoryBytes":               int64(68719476736),
		"storageBytes":              int64(2000000000000),
		"storageMediaKind":          "ssd",
		"storageInterfaceKind":      "nvme",
		"estimatedMonthlyCostCents": int64(2500),
		"costCurrency":              "usd",
		"notes":                     " Main host ",
		"isFavorite":                true,
	}
	hostResponse := apiRequest(t, handler, http.MethodPost, "/api/v1/machines", hostBody, "application/json")
	assertStatus(t, hostResponse, http.StatusCreated)
	host := decodeObject(t, hostResponse)
	hostID := host["publicId"].(string)
	wantHost := map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "assetPublicId": systemAssetID,
		"name": "Atlas Host", "slug": "atlas-host", "kind": "bare_metal", "hostname": "atlas.example.com",
		"osMachineId": "aabbccdd", "operatingSystem": "Debian", "operatingSystemVersion": "13",
		"kernel": "6.12.1", "architecture": "x86_64", "cpuCount": float64(16), "cpuAllocation": "dedicated",
		"cpuVendor": "AMD", "memoryBytes": float64(68719476736), "storageBytes": float64(2000000000000),
		"storageMediaKind": "ssd", "storageInterfaceKind": "nvme", "estimatedMonthlyCostCents": float64(2500),
		"costCurrency": "USD", "notes": "Main host", "isFavorite": true,
	}
	for field, want := range wantHost {
		if got := host[field]; got != want {
			t.Errorf("created machine %s = %v; want %v", field, got, want)
		}
	}
	hostGet := apiRequest(t, handler, http.MethodGet, "/api/v1/machines/"+hostID, nil, "")
	assertStatus(t, hostGet, http.StatusOK)
	if got := decodeObject(t, hostGet)["storageInterfaceKind"]; got != "nvme" {
		t.Fatalf("machine GET storageInterfaceKind = %v", got)
	}
	listResponse := apiRequest(t, handler, http.MethodGet, "/api/v1/machines", nil, "")
	assertStatus(t, listResponse, http.StatusOK)
	listed := decodeObject(t, listResponse)["items"].([]any)[0].(map[string]any)
	if listed["isFavorite"] != true {
		t.Fatalf("machine list isFavorite = %v; want true", listed["isFavorite"])
	}
	delete(hostBody, "isFavorite")
	hostUpdate := apiRequest(t, handler, http.MethodPut, "/api/v1/machines/"+hostID, hostBody, "application/json")
	assertStatus(t, hostUpdate, http.StatusOK)
	if got := decodeObject(t, hostUpdate)["isFavorite"]; got != false {
		t.Fatalf("machine omitted PUT isFavorite = %v; want false", got)
	}

	for name, assetID := range map[string]string{
		"non-system product": nonSystemAsset["publicId"].(string),
		"unplaced system":    unplacedAsset["publicId"].(string),
	} {
		t.Run("invalid backing asset "+name, func(t *testing.T) {
			response := apiRequest(t, handler, http.MethodPost, "/api/v1/machines", map[string]any{
				"machineProviderPublicId": providerID, "areaPublicId": areaID, "assetPublicId": assetID,
				"name": "Invalid " + name, "kind": "bare_metal",
			}, "application/json")
			assertStatus(t, response, http.StatusBadRequest)
		})
	}

	otherHost := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": otherAreaID, "name": "Other Host", "kind": "bare_metal",
	})
	wrongLocationParent := apiRequest(t, handler, http.MethodPost, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "parentMachinePublicId": otherHost["publicId"],
		"name": "Wrong Location VM", "kind": "virtual_machine", "virtualizationPlatform": "kvm",
	}, "application/json")
	assertStatus(t, wrongLocationParent, http.StatusBadRequest)

	vm := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "parentMachinePublicId": hostID,
		"name": "Guest One", "kind": "virtual_machine", "virtualizationPlatform": " KVM ", "cpuAllocation": "shared",
	})
	vmID := vm["publicId"].(string)
	if vm["parentMachinePublicId"] != hostID || vm["virtualizationPlatform"] != "KVM" || vm["cpuAllocation"] != "shared" {
		t.Fatalf("virtual machine = %#v", vm)
	}
	vmWithAsset := apiRequest(t, handler, http.MethodPost, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "assetPublicId": systemAssetID,
		"name": "Asset VM", "kind": "virtual_machine",
	}, "application/json")
	assertStatus(t, vmWithAsset, http.StatusBadRequest)

	nestedVM := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "parentMachinePublicId": vmID,
		"name": "Nested Guest", "kind": "virtual_machine", "virtualizationPlatform": "qemu",
	})
	cycle := apiRequest(t, handler, http.MethodPut, "/api/v1/machines/"+vmID, map[string]any{
		"machineProviderPublicId": providerID, "areaPublicId": areaID, "parentMachinePublicId": nestedVM["publicId"],
		"name": "Guest One", "kind": "virtual_machine", "virtualizationPlatform": "KVM", "cpuAllocation": "shared",
	}, "application/json")
	assertStatus(t, cycle, http.StatusConflict)
}

func TestServiceLogoAPI(t *testing.T) {
	handler := newAPITestHandler(t)
	service := createAPIResource(t, handler, "/api/v1/services", map[string]any{"name": "Grafana"})
	serviceID := service["publicId"].(string)
	if service["hasLogo"] != false {
		t.Fatalf("new service hasLogo = %v", service["hasLogo"])
	}

	png := []byte("\x89PNG\r\n\x1a\nHLIMS")
	stored := rawAPIRequest(t, handler, http.MethodPut, "/api/v1/services/"+serviceID+"/logo", png, "image/png")
	assertStatus(t, stored, http.StatusNoContent)

	serviceResponse := apiRequest(t, handler, http.MethodGet, "/api/v1/services/"+serviceID, nil, "")
	assertStatus(t, serviceResponse, http.StatusOK)
	if decodeObject(t, serviceResponse)["hasLogo"] != true {
		t.Fatal("service did not report its stored logo")
	}
	listResponse := apiRequest(t, handler, http.MethodGet, "/api/v1/services", nil, "")
	assertStatus(t, listResponse, http.StatusOK)
	listed := decodeObject(t, listResponse)["items"].([]any)[0].(map[string]any)
	if listed["hasLogo"] != true {
		t.Fatal("service list did not report its stored logo")
	}

	fetched := rawAPIRequest(t, handler, http.MethodGet, "/api/v1/services/"+serviceID+"/logo", nil, "")
	assertStatus(t, fetched, http.StatusOK)
	if !bytes.Equal(fetched.Body.Bytes(), png) || fetched.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("fetched logo content type = %q; body = %x", fetched.Header().Get("Content-Type"), fetched.Body.Bytes())
	}
	if fetched.Header().Get("ETag") == "" || fetched.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("fetched logo headers = %#v", fetched.Header())
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/services/"+serviceID+"/logo", nil)
	request.Header.Set("If-None-Match", fetched.Header().Get("ETag"))
	notModified := httptest.NewRecorder()
	handler.ServeHTTP(notModified, request)
	assertStatus(t, notModified, http.StatusNotModified)

	mismatch := rawAPIRequest(t, handler, http.MethodPut, "/api/v1/services/"+serviceID+"/logo", png, "image/jpeg")
	assertStatus(t, mismatch, http.StatusBadRequest)
	tooLarge := rawAPIRequest(t, handler, http.MethodPut, "/api/v1/services/"+serviceID+"/logo", make([]byte, maxLogoSize+1), "image/png")
	assertStatus(t, tooLarge, http.StatusRequestEntityTooLarge)

	deleted := rawAPIRequest(t, handler, http.MethodDelete, "/api/v1/services/"+serviceID+"/logo", nil, "")
	assertStatus(t, deleted, http.StatusNoContent)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodGet, "/api/v1/services/"+serviceID+"/logo", nil, ""), http.StatusNotFound)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodDelete, "/api/v1/services/"+serviceID+"/logo", nil, ""), http.StatusNotFound)
	serviceResponse = apiRequest(t, handler, http.MethodGet, "/api/v1/services/"+serviceID, nil, "")
	if decodeObject(t, serviceResponse)["hasLogo"] != false {
		t.Fatal("service still reported a deleted logo")
	}
}

func TestMachineProviderLogoAPI(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Hetzner"})
	providerID := provider["publicId"].(string)
	if provider["hasLogo"] != false {
		t.Fatalf("new provider hasLogo = %v", provider["hasLogo"])
	}

	png := []byte("\x89PNG\r\n\x1a\nPROVIDER")
	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+providerID+"/logo", png, "image/png"), http.StatusNoContent)

	getProvider := apiRequest(t, handler, http.MethodGet, "/api/v1/machine-providers/"+providerID, nil, "")
	assertStatus(t, getProvider, http.StatusOK)
	if decodeObject(t, getProvider)["hasLogo"] != true {
		t.Fatal("provider did not report its stored logo")
	}
	listProviders := apiRequest(t, handler, http.MethodGet, "/api/v1/machine-providers", nil, "")
	assertStatus(t, listProviders, http.StatusOK)
	if decodeObject(t, listProviders)["items"].([]any)[0].(map[string]any)["hasLogo"] != true {
		t.Fatal("provider list did not report its stored logo")
	}
	updatedProvider := apiRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+providerID, map[string]any{"name": "Hetzner Cloud"}, "application/json")
	assertStatus(t, updatedProvider, http.StatusOK)
	if updated := decodeObject(t, updatedProvider); updated["hasLogo"] != true || updated["name"] != "Hetzner Cloud" {
		t.Fatalf("updated provider = %#v", updated)
	}

	fetched := rawAPIRequest(t, handler, http.MethodGet, "/api/v1/machine-providers/"+providerID+"/logo", nil, "")
	assertStatus(t, fetched, http.StatusOK)
	if !bytes.Equal(fetched.Body.Bytes(), png) || fetched.Header().Get("Content-Type") != "image/png" || fetched.Header().Get("ETag") == "" {
		t.Fatalf("fetched provider logo headers = %#v; body = %x", fetched.Header(), fetched.Body.Bytes())
	}

	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+providerID+"/logo", png, "image/jpeg"), http.StatusBadRequest)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodPut, "/api/v1/machine-providers/"+providerID+"/logo", make([]byte, maxLogoSize+1), "image/png"), http.StatusRequestEntityTooLarge)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodDelete, "/api/v1/machine-providers/"+providerID+"/logo", nil, ""), http.StatusNoContent)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodGet, "/api/v1/machine-providers/"+providerID+"/logo", nil, ""), http.StatusNotFound)
	assertStatus(t, rawAPIRequest(t, handler, http.MethodDelete, "/api/v1/machine-providers/"+providerID+"/logo", nil, ""), http.StatusNotFound)
}

func newAPITestHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := NewSQLiteDB(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return newHandler(db)
}

func createAPIResource(t *testing.T, handler http.Handler, target string, body map[string]any) map[string]any {
	t.Helper()
	response := apiRequest(t, handler, http.MethodPost, target, body, "application/json")
	assertStatus(t, response, http.StatusCreated)
	return decodeObject(t, response)
}

func apiRequest(t *testing.T, handler http.Handler, method, target string, body any, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(encoded))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func rawAPIRequest(t *testing.T, handler http.Handler, method, target string, body []byte, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, bytes.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d; want %d; body = %s", response.Code, want, response.Body.String())
	}
}

func decodeObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
