package cli

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceLogoSetBySlugVerifiesUpload(t *testing.T) {
	t.Parallel()
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	logo := imageData.Bytes()
	var uploaded bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"publicId":"service00001","slug":"opencode"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/services/service00001/logo":
			if r.Header.Get("Content-Type") != "image/png" {
				t.Errorf("logo Content-Type = %q", r.Header.Get("Content-Type"))
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || !bytes.Equal(body, logo) {
				t.Errorf("logo bytes do not match: %v", err)
			}
			uploaded = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services/service00001/logo":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(logo)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services/service00001":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"publicId":"service00001","name":"OpenCode","slug":"opencode","hasLogo":true}`))
		default:
			t.Errorf("unexpected logo request: %s %s", r.Method, r.URL)
		}
	}))
	t.Cleanup(server.Close)
	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Input: bytes.NewReader(logo), Output: &output, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "services", "logo", "set", "opencode", "--file", "-"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !uploaded || !strings.Contains(output.String(), `"hasLogo": true`) {
		t.Fatalf("logo was not uploaded and verified: %s", output.String())
	}
}

func TestLogoRejectsUnsupportedDataBeforeRequest(t *testing.T) {
	t.Parallel()
	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requested = true }))
	t.Cleanup(server.Close)
	command := NewRootCommand(Dependencies{Input: strings.NewReader("not an image"), Output: &bytes.Buffer{}, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "services", "logo", "set", "opencode", "--file", "-"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported logo image type") || requested {
		t.Fatalf("unverified artwork reached API: error=%v requested=%v", err, requested)
	}
}
