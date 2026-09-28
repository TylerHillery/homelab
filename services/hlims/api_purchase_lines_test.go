package hlims

import (
	"net/http"
	"testing"
)

func TestPurchaseTracksSelectedLinesWithoutAssigningWholeOrderCostToAsset(t *testing.T) {
	handler := newAPITestHandler(t)
	manufacturer := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Example NIC Vendor"})
	product := createAPIResource(t, handler, "/api/v1/products", map[string]any{
		"manufacturerPublicId": manufacturer["publicId"], "kind": "network_adapter", "name": "Two-port NIC",
	})
	asset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
		"productPublicId": product["publicId"], "placement": map[string]any{"type": "unplaced"},
	})
	payload := map[string]any{
		"assetPublicIds": []any{asset["publicId"]}, "totalPriceCents": 4534,
		"currency": "USD", "source": "Example store", "orderReference": "private-order-1",
		"lines": []any{map[string]any{"productPublicId": product["publicId"], "description": "Two-port NIC", "quantity": 1, "subtotalCents": 3299}},
	}
	purchase := createAPIResource(t, handler, "/api/v1/purchases", payload)
	if purchase["trackedSubtotalCents"] != float64(3299) || purchase["totalPriceCents"] != float64(4534) || purchase["orderReference"] != "private-order-1" {
		t.Fatalf("partial order = %#v", purchase)
	}
	if lines := purchase["lines"].([]any); len(lines) != 1 || lines[0].(map[string]any)["quantity"] != float64(1) {
		t.Fatalf("selected lines = %#v", lines)
	}
	list := apiRequest(t, handler, http.MethodGet, "/api/v1/purchases", nil, "")
	assertStatus(t, list, http.StatusOK)
	if got := decodeObject(t, list)["items"].([]any)[0].(map[string]any)["trackedSubtotalCents"]; got != float64(3299) {
		t.Fatalf("listed tracked subtotal = %#v", got)
	}
	badQuantity := map[string]any{
		"assetPublicIds": []any{asset["publicId"]}, "totalPriceCents": 4534, "currency": "USD",
		"lines": []any{map[string]any{"description": "NIC", "quantity": 0}},
	}
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/purchases", badQuantity, "application/json"), http.StatusBadRequest)
	badTotal := map[string]any{
		"assetPublicIds": []any{asset["publicId"]}, "totalPriceCents": 3000, "currency": "USD",
		"lines": []any{map[string]any{"description": "NIC", "quantity": 1, "subtotalCents": 3299}},
	}
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/purchases", badTotal, "application/json"), http.StatusBadRequest)
}

func TestSharedHomePurchaseLineDoesNotCountTowardHomelabSubtotal(t *testing.T) {
	handler := newAPITestHandler(t)
	maker := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Example"})
	createAsset := func(kind, name string) (map[string]any, map[string]any) {
		product := createAPIResource(t, handler, "/api/v1/products", map[string]any{
			"manufacturerPublicId": maker["publicId"], "kind": kind, "name": name,
		})
		asset := createAPIResource(t, handler, "/api/v1/assets", map[string]any{
			"productPublicId": product["publicId"], "name": name + " unit", "placement": map[string]any{"type": "unplaced"},
		})
		return product, asset
	}
	nic, nicAsset := createAsset("network_adapter", "Lab NIC")
	router, routerAsset := createAsset("router", "Household router")
	purchase := createAPIResource(t, handler, "/api/v1/purchases", map[string]any{
		"assetPublicIds":  []any{nicAsset["publicId"], routerAsset["publicId"]},
		"totalPriceCents": 11000, "currency": "USD", "source": "Example store",
		"lines": []any{
			map[string]any{"productPublicId": nic["publicId"], "description": "Lab NIC", "quantity": 1, "subtotalCents": 3000},
			map[string]any{"productPublicId": router["publicId"], "description": "Household router", "quantity": 1, "subtotalCents": 7000, "includeInHomelabTotal": false},
		},
	})
	if purchase["trackedSubtotalCents"] != float64(10000) || purchase["homelabSubtotalCents"] != float64(3000) || len(purchase["assetPublicIds"].([]any)) != 2 {
		t.Fatalf("mixed-purpose order = %#v", purchase)
	}
	lines := purchase["lines"].([]any)
	if lines[0].(map[string]any)["includeInHomelabTotal"] != true || lines[1].(map[string]any)["includeInHomelabTotal"] != false {
		t.Fatalf("purchase line classification = %#v", lines)
	}
	update := map[string]any{
		"assetPublicIds": purchase["assetPublicIds"], "totalPriceCents": 11000, "currency": "USD",
		"source": "Example store", "lines": []any{
			map[string]any{"productPublicId": nic["publicId"], "description": "Lab NIC", "quantity": 1, "subtotalCents": 3000},
			map[string]any{"productPublicId": router["publicId"], "description": "Household router", "quantity": 1, "subtotalCents": 7000, "includeInHomelabTotal": true},
		},
	}
	response := apiRequest(t, handler, http.MethodPut, "/api/v1/purchases/"+purchase["publicId"].(string), update, "application/json")
	assertStatus(t, response, http.StatusOK)
	if subtotal := decodeObject(t, response)["homelabSubtotalCents"]; subtotal != float64(10000) {
		t.Fatalf("reclassified homelab subtotal = %#v", subtotal)
	}
}
