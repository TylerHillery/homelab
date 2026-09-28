package hlims

import (
	"net/http"
	"testing"
)

func TestRouterWiFiSpecsAndWiredPortProfiles(t *testing.T) {
	handler := newAPITestHandler(t)
	maker := createAPIResource(t, handler, "/api/v1/manufacturers", map[string]any{"name": "Example Network"})
	body := map[string]any{
		"manufacturerPublicId": maker["publicId"], "kind": "router", "name": "Tri-band router",
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
	}
	product := createAPIResource(t, handler, "/api/v1/products", body)
	if wifi := product["wifiSpec"].(map[string]any); wifi["generation"] != float64(7) || len(wifi["bands"].([]any)) != 3 {
		t.Fatalf("Wi-Fi specifications = %#v", wifi)
	}
	if ports := product["portProfiles"].([]any); len(ports) != 2 || ports[0].(map[string]any)["speedMbps"] != float64(5000) {
		t.Fatalf("wired port groups = %#v", ports)
	}
	listed := apiRequest(t, handler, http.MethodGet, "/api/v1/products", nil, "")
	assertStatus(t, listed, http.StatusOK)
	if len(decodeObject(t, listed)["items"].([]any)[0].(map[string]any)["wifiSpec"].(map[string]any)["bands"].([]any)) != 3 {
		t.Fatal("product list lost Wi-Fi band specifications")
	}
	body["kind"] = "switch"
	assertStatus(t, apiRequest(t, handler, http.MethodPost, "/api/v1/products", body, "application/json"), http.StatusBadRequest)
	body["kind"] = "router"
	body["wifiSpec"].(map[string]any)["bands"] = []any{
		map[string]any{"bandGHz": "6", "maxLinkMbps": 5765},
		map[string]any{"bandGHz": "6", "maxLinkMbps": 5000},
	}
	assertStatus(t, apiRequest(t, handler, http.MethodPut, "/api/v1/products/"+product["publicId"].(string), body, "application/json"), http.StatusBadRequest)
	read := apiRequest(t, handler, http.MethodGet, "/api/v1/products/"+product["publicId"].(string), nil, "")
	assertStatus(t, read, http.StatusOK)
	if len(decodeObject(t, read)["wifiSpec"].(map[string]any)["bands"].([]any)) != 3 {
		t.Fatal("invalid update changed existing Wi-Fi specifications")
	}
}
