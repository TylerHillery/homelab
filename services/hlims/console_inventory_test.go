package hlims

import (
	"net/http"
	"strings"
	"testing"
)

func TestInventoryAndOrdersExposeSparePartsAndSelectedPurchaseLines(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Lab"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{
		"machineProviderPublicId": provider["publicId"], "name": "Home",
	})
	maker := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Example"})
	system := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": maker["publicId"], "kind": "system", "name": "Test system",
		"links": []any{map[string]any{"kind": "support", "label": "System support guide", "url": "https://example.com/system"}},
	})
	memory := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": maker["publicId"], "kind": "memory", "name": "8 GiB DDR3 module",
		"memorySpec": map[string]any{"capacityBytes": 8589934592, "memoryType": "DDR3"},
	})
	systemAsset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": system["publicId"], "name": "test-host",
		"placement": map[string]any{"type": "area", "areaPublicId": area["publicId"]},
	})
	machine := createAPIResource(t, handler, "/api/v1/machines", map[string]any{
		"machineProviderPublicId": provider["publicId"], "areaPublicId": area["publicId"],
		"name": "test-host", "kind": "bare_metal", "assetPublicId": systemAsset["publicId"],
	})
	installed := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": memory["publicId"], "name": "Installed module",
		"placement": map[string]any{"type": "asset", "parentAssetPublicId": systemAsset["publicId"], "slot": "DIMM 1"},
	})
	spare := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": memory["publicId"], "name": "Stored module",
		"placement": map[string]any{"type": "unplaced"},
	})
	purchase := createAPIResource(t, handler, "/api/v1/purchases", map[string]any{
		"assetPublicIds":  []any{installed["publicId"], spare["publicId"]},
		"totalPriceCents": 4534, "currency": "USD", "source": "Example store", "purchasedOn": "2026-09-15",
		"orderReference": "example-1",
		"lines":          []any{map[string]any{"description": "Kit with two modules", "quantity": 1, "subtotalCents": 3299}},
	})

	inventory := consoleRequest(handler, http.MethodGet, "/console/inventory/", false)
	assertStatus(t, inventory, http.StatusOK)
	for _, want := range []string{
		"HLIMS / Inventory", `href="/console/inventory/" aria-current="page"`, "8 GiB DDR3 module",
		"Installed module", "Stored module", `href="/console/#machine-` + machine["publicId"].(string) + `"`,
		`href="/console/orders/#order-` + purchase["publicId"].(string) + `"`,
	} {
		if !strings.Contains(inventory.Body.String(), want) {
			t.Errorf("inventory page missing %q", want)
		}
	}
	if strings.Contains(inventory.Body.String(), "What you own. Where it lives.") || strings.Contains(inventory.Body.String(), "Physical assets</span>") {
		t.Fatal("Inventory should lead with the filters and records, not an intro or summary dashboard")
	}
	filtered := consoleRequest(handler, http.MethodGet, "/console/inventory/?filter=uninstalled", false)
	assertStatus(t, filtered, http.StatusOK)
	if !strings.Contains(filtered.Body.String(), "Stored module") || strings.Contains(filtered.Body.String(), "Installed module") {
		t.Fatalf("uninstalled parts filter = %s", filtered.Body.String())
	}
	assertStatus(t, consoleRequest(handler, http.MethodGet, "/console/inventory/?filter=bad", false), http.StatusBadRequest)

	orders := consoleRequest(handler, http.MethodGet, "/console/orders/", false)
	assertStatus(t, orders, http.StatusOK)
	for _, want := range []string{
		"HLIMS / Orders", `href="/console/orders/" aria-current="page"`, "Example store", "Kit with two modules", "Homelab item subtotals",
		"$45.34", "$32.99", "$12.35", "example-1", "Stored module",
		`href="/console/inventory/?asset=` + spare["publicId"].(string) + `#asset-` + spare["publicId"].(string) + `"`,
	} {
		if !strings.Contains(orders.Body.String(), want) {
			t.Errorf("orders page missing %q", want)
		}
	}
	if strings.Contains(orders.Body.String(), "The story behind the hardware.") || strings.Contains(orders.Body.String(), "Full order totals / USD") || strings.Contains(orders.Body.String(), "Orders / Assets") {
		t.Fatal("Orders should only show the homelab subtotal above the individual order cards")
	}
	assetDetail := consoleRequest(handler, http.MethodGet, "/console/inventory/?asset="+spare["publicId"].(string), false)
	assertStatus(t, assetDetail, http.StatusOK)
	if !strings.Contains(assetDetail.Body.String(), `role="dialog" aria-modal="true"`) || !strings.Contains(assetDetail.Body.String(), "Stored module") || !strings.Contains(assetDetail.Body.String(), `href="/console/orders/#order-`+purchase["publicId"].(string)+`"`) {
		t.Fatalf("deep-linked asset detail = %s", assetDetail.Body.String())
	}
	fragment := consoleRequest(handler, http.MethodGet, "/console/inventory/detail?product="+memory["publicId"].(string), false)
	assertStatus(t, fragment, http.StatusOK)
	if strings.Contains(fragment.Body.String(), "<!doctype html>") || !strings.Contains(fragment.Body.String(), "DDR3") || !strings.Contains(fragment.Body.String(), "Installed module") {
		t.Fatalf("product drawer fragment = %s", fragment.Body.String())
	}
	assertStatus(t, consoleRequest(handler, http.MethodGet, "/console/inventory/detail?asset=missing", false), http.StatusNotFound)
	systemDetail := consoleRequest(handler, http.MethodGet, "/console/inventory/detail?asset="+systemAsset["publicId"].(string), false)
	assertStatus(t, systemDetail, http.StatusOK)
	if !strings.Contains(systemDetail.Body.String(), "Contained Assets") || !strings.Contains(systemDetail.Body.String(), "Installed module") || strings.Contains(systemDetail.Body.String(), "Stored module") || !strings.Contains(systemDetail.Body.String(), "System support guide") {
		t.Fatalf("system should show only its installed parts: %s", systemDetail.Body.String())
	}
	componentDetail := consoleRequest(handler, http.MethodGet, "/console/inventory/detail?asset="+installed["publicId"].(string), false)
	assertStatus(t, componentDetail, http.StatusOK)
	if !strings.Contains(componentDetail.Body.String(), "Contained in") || !strings.Contains(componentDetail.Body.String(), `href="/console/inventory/?asset=`+systemAsset["publicId"].(string)+`"`) || !strings.Contains(componentDetail.Body.String(), `href="/console/#machine-`+machine["publicId"].(string)+`"`) {
		t.Fatalf("component should link back to its system and Machine: %s", componentDetail.Body.String())
	}
	rack := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": maker["publicId"], "kind": "rack", "name": "Test Rack",
		"rackSpec": map[string]any{"rackUnits": 4, "mountingStandard": "10-inch"},
	})
	rackAsset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": rack["publicId"], "name": "rack",
		"placement": map[string]any{"type": "area", "areaPublicId": area["publicId"]},
	})
	assertStatus(t, apiRequest(t, handler, http.MethodPut, "/api/v1/assets/"+systemAsset["publicId"].(string), map[string]any{
		"productPublicId": system["publicId"], "name": "test-host",
		"placement": map[string]any{"type": "asset", "parentAssetPublicId": rackAsset["publicId"], "slot": "U1"},
	}, "application/json"), http.StatusOK)
	rackDetail := consoleRequest(handler, http.MethodGet, "/console/inventory/detail?asset="+rackAsset["publicId"].(string), false)
	assertStatus(t, rackDetail, http.StatusOK)
	if !strings.Contains(rackDetail.Body.String(), "test-host") || !strings.Contains(rackDetail.Body.String(), "Installed module") || !strings.Contains(rackDetail.Body.String(), "--component-depth: 2") || strings.Contains(rackDetail.Body.String(), "Stored module") {
		t.Fatalf("rack should recursively show its Machine and installed module, not unrelated spares: %s", rackDetail.Body.String())
	}
	for _, path := range []string{"/console/inventory", "/console/orders"} {
		response := consoleRequest(handler, http.MethodGet, path, false)
		assertStatus(t, response, http.StatusPermanentRedirect)
		if response.Header().Get("Location") != path+"/" {
			t.Fatalf("%s redirects to %q", path, response.Header().Get("Location"))
		}
	}
}

func TestInventoryDrawerSeparatesSharedRouterSpecsFromEachOwnedUnit(t *testing.T) {
	handler := newAPITestHandler(t)
	provider := createAPIResource(t, handler, "/api/v1/machine-providers", map[string]any{"name": "Home"})
	area := createAPIResource(t, handler, "/api/v1/areas", map[string]any{"machineProviderPublicId": provider["publicId"], "name": "Site"})
	maker := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Example Networking"})
	model := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": maker["publicId"], "kind": "router", "name": "Example Wi-Fi 7 mesh unit",
		"wifiSpec": map[string]any{
			"generation": 7, "ieeeStandard": "802.11be", "class": "BE11000", "maxChannelWidthMHz": 320,
			"bands": []any{
				map[string]any{"bandGHz": "2.4", "maxLinkMbps": 688},
				map[string]any{"bandGHz": "5", "maxLinkMbps": 4324},
				map[string]any{"bandGHz": "6", "maxLinkMbps": 5765},
			},
		},
		"portProfiles": []any{
			map[string]any{"name": "WAN/LAN", "portCount": 2, "connector": "RJ45", "speedMbps": 5000},
			map[string]any{"name": "WAN/LAN", "portCount": 1, "connector": "RJ45", "speedMbps": 2500},
		},
	})
	unitNames := []string{"main", "office", "living-room"}
	units := make([]map[string]any, 0, len(unitNames))
	for _, name := range unitNames {
		units = append(units, createAPIResource(t, handler, "/api/v1/assets", map[string]any{
			"productPublicId": model["publicId"], "name": name,
			"notes":     "Unit at " + name,
			"placement": map[string]any{"type": "area", "areaPublicId": area["publicId"]},
		}))
	}
	purchase := createAPIResource(t, handler, "/api/v1/purchases", map[string]any{
		"assetPublicIds":  []any{units[0]["publicId"], units[1]["publicId"], units[2]["publicId"]},
		"totalPriceCents": 63299, "currency": "USD", "source": "Example retailer",
		"lines": []any{map[string]any{
			"description": "Three-pack", "quantity": 1, "subtotalCents": 59999,
			"includeInHomelabTotal": false,
		}},
	})
	fragment := consoleRequest(handler, http.MethodGet, "/console/inventory/detail?asset="+units[1]["publicId"].(string), false)
	assertStatus(t, fragment, http.StatusOK)
	for _, want := range []string{
		"Unit at office", "Example Wi-Fi 7 mesh unit", "802.11be", "BE11000", "320 MHz",
		"2.4 GHz", "688 Mbps", "5 GHz", "4324 Mbps", "6 GHz", "5765 Mbps",
		"2× RJ45 · 5 Gbps", "1× RJ45 · 2.5 Gbps",
		`href="/console/orders/#order-` + purchase["publicId"].(string) + `"`,
		`href="/console/inventory/?asset=` + units[0]["publicId"].(string) + `"`,
	} {
		if !strings.Contains(fragment.Body.String(), want) {
			t.Errorf("unit drawer missing %q", want)
		}
	}
	if strings.Contains(fragment.Body.String(), "Unit at living-room</p>") {
		t.Fatal("office unit detail should not inherit a different Asset's role notes")
	}
	modelFragment := consoleRequest(handler, http.MethodGet, "/console/inventory/detail?product="+model["publicId"].(string), false)
	assertStatus(t, modelFragment, http.StatusOK)
	if strings.Contains(modelFragment.Body.String(), "This unit") || !strings.Contains(modelFragment.Body.String(), "3 owned") {
		t.Fatalf("shared model drawer = %s", modelFragment.Body.String())
	}
	orders := consoleRequest(handler, http.MethodGet, "/console/orders/", false)
	assertStatus(t, orders, http.StatusOK)
	if !strings.Contains(orders.Body.String(), "Homelab item subtotals") || !strings.Contains(orders.Body.String(), "Shared home · excluded from homelab total") || !strings.Contains(orders.Body.String(), "$0.00") || !strings.Contains(orders.Body.String(), "$599.99") {
		t.Fatalf("mixed-use purchase should remain inventoried without inflating homelab spending: %s", orders.Body.String())
	}
}
