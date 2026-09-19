package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRecordsUsesGeneratedAPIClient(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/machines" {
			t.Errorf("request = %s %s, want GET /api/v1/machines", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"items":[{"publicId":"badmach8k2q5n","name":"Badger","slug":"badger","kind":"bare_metal"}]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL+"/api/v1", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	records, err := client.ListRecords(context.Background(), Machines)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].Name != "Badger" || records[0].Detail != "bare_metal" {
		t.Fatalf("record = %#v", records[0])
	}
}

func TestTopologyFetchesAndDecodesGeneratedModel(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/topology" {
			t.Errorf("request = %s %s, want GET /api/v1/topology", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"providers":[{"publicId":"provider0001","name":"Provider","slug":"provider","areas":[]}]}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL+"/api/v1", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	topology, err := client.Topology(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(topology.Providers) != 1 || topology.Providers[0].Name != "Provider" {
		t.Fatalf("topology = %#v", topology)
	}
}

func TestMachineUserResourceDispatch(t *testing.T) {
	t.Parallel()
	requests := make(chan string, 5)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Method + " " + request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		if request.Method == http.MethodDelete {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		_, _ = writer.Write([]byte(`{"publicId":"user00000001","machinePublicId":"machine00001","username":"tyler","isPreferred":true}`))
	}))
	t.Cleanup(server.Close)
	client, err := New(server.URL+"/api/v1", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"machinePublicId":"machine00001","username":"tyler"}`)
	if _, err := client.List(context.Background(), MachineUsers); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Get(context.Background(), MachineUsers, "user00000001"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Create(context.Background(), MachineUsers, body); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Update(context.Background(), MachineUsers, "user00000001", body); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Delete(context.Background(), MachineUsers, "user00000001"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /api/v1/machine-users", "GET /api/v1/machine-users/user00000001", "POST /api/v1/machine-users",
		"PUT /api/v1/machine-users/user00000001", "DELETE /api/v1/machine-users/user00000001",
	}
	for _, expected := range want {
		if got := <-requests; got != expected {
			t.Fatalf("request = %q; want %q", got, expected)
		}
	}
}

func TestAPIErrorIncludesStructuredResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte(`{"code":"not_found","message":"machine not found"}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL+"/api/v1", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Get(context.Background(), Machines, "missingmach1")
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *Error", err)
	}
	if apiErr.StatusCode != http.StatusNotFound || apiErr.Code != "not_found" {
		t.Fatalf("API error = %#v", apiErr)
	}
}

func TestNewRejectsNonHTTPURL(t *testing.T) {
	t.Parallel()
	if _, err := New("framework13-wsl", nil); err == nil {
		t.Fatal("New accepted an API URL without an HTTP scheme")
	}
}

func TestOpenAPISpecUsesServerRoot(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/openapi.yaml" {
			t.Errorf("path = %q, want /openapi.yaml", request.URL.Path)
		}
		_, _ = writer.Write([]byte("openapi: 3.0.3\n"))
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL+"/api/v1", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.OpenAPISpec(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(response.Body) != "openapi: 3.0.3\n" {
		t.Fatalf("body = %q", response.Body)
	}
}

func TestDestinationURLRejectsNilResponse(t *testing.T) {
	t.Parallel()
	if _, err := DestinationURL(nil); err == nil {
		t.Fatal("DestinationURL accepted a nil response")
	}
}
