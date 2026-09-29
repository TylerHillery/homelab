package hlims

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// conditionalInventoryWrites serializes PUTs and compares optional If-Match
// headers against the current resource representation. CLI merge updates send
// this header so that another write cannot silently overwrite a newer record.
func conditionalInventoryWrites(next http.Handler) http.Handler {
	var putMu sync.Mutex
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !inventoryItemPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPut {
			putMu.Lock()
			defer putMu.Unlock()
			if expected := r.Header.Get("If-Match"); expected != "" {
				current := httptest.NewRecorder()
				probe := r.Clone(r.Context())
				probe.Method, probe.Body = http.MethodGet, nil
				next.ServeHTTP(current, probe)
				if current.Code != http.StatusOK {
					writeInventoryResponse(w, current)
					return
				}
				if expected != inventoryETag(current.Body.Bytes()) {
					writeAPIError(w, http.StatusPreconditionFailed, "precondition_failed", "inventory record changed; fetch it and retry")
					return
				}
			}
		}
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		response := httptest.NewRecorder()
		next.ServeHTTP(response, r)
		if response.Code == http.StatusOK {
			response.Header().Set("ETag", inventoryETag(response.Body.Bytes()))
		}
		writeInventoryResponse(w, response)
	})
}

func writeInventoryResponse(w http.ResponseWriter, response *httptest.ResponseRecorder) {
	for name, values := range response.Header() {
		w.Header()[name] = append([]string(nil), values...)
	}
	w.WriteHeader(response.Code)
	_, _ = w.Write(response.Body.Bytes())
}

func inventoryETag(body []byte) string {
	digest := sha256.Sum256(body)
	return `"` + hex.EncodeToString(digest[:]) + `"`
}

func inventoryItemPath(path string) bool {
	if !strings.HasPrefix(path, "/api/v1/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	if len(parts) != 2 || len(parts[1]) != 12 {
		return false
	}
	switch parts[0] {
	case "machine-providers", "areas", "manufacturers", "products", "assets", "purchases", "machines", "machine-users", "networks", "addresses", "deployments", "services", "instances", "instance-dependencies", "instance-endpoints":
		return true
	default:
		return false
	}
}
