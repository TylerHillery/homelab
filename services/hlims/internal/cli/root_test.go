package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMachineListCommand(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/machines" {
			t.Errorf("path = %q, want /api/v1/machines", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)

	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Output: &output, Error: &output, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "machines", "list"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "{\n  \"items\": []\n}\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestCreateRejectsInvalidJSONBeforeRequest(t *testing.T) {
	t.Parallel()
	var requested bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requested = true
	}))
	t.Cleanup(server.Close)

	command := NewRootCommand(Dependencies{
		Input:      strings.NewReader("not-json"),
		Output:     &bytes.Buffer{},
		Error:      &bytes.Buffer{},
		HTTPClient: server.Client(),
	})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "services", "create"})
	err := command.Execute()
	if err == nil || err.Error() != "request body is not valid JSON" {
		t.Fatalf("error = %v", err)
	}
	if requested {
		t.Fatal("invalid JSON reached the API server")
	}
}

func TestDeleteRequiresConfirmation(t *testing.T) {
	t.Parallel()
	command := NewRootCommand(Dependencies{Output: &bytes.Buffer{}, Error: &bytes.Buffer{}})
	command.SetArgs([]string{"machines", "delete", "badmach8k2q5n"})
	err := command.Execute()
	if err == nil || err.Error() != "refusing to delete without --yes" {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateRejectsOversizedInput(t *testing.T) {
	t.Parallel()
	command := NewRootCommand(Dependencies{
		Input:  strings.NewReader(strings.Repeat(" ", maxInputSize+1)),
		Output: &bytes.Buffer{},
		Error:  &bytes.Buffer{},
	})
	command.SetArgs([]string{"services", "create"})
	err := command.Execute()
	if err == nil || err.Error() != "JSON request exceeds 16 MiB" {
		t.Fatalf("error = %v", err)
	}
}

func TestCompactOutput(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)

	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Output: &output, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "--compact", "machines", "list"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "{\"items\":[]}\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestDataCommandRejectsNonJSONSuccess(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("login required"))
	}))
	t.Cleanup(server.Close)

	command := NewRootCommand(Dependencies{Output: &bytes.Buffer{}, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "machines", "list"})
	err := command.Execute()
	if err == nil || err.Error() != "HLIMS API returned a non-JSON success response" {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateHelpExamplesIncludePublicID(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Output: &output, Error: &output})
	command.SetArgs([]string{"services", "update", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "hlims services update PUBLIC_ID --file record.json") {
		t.Fatalf("help output omitted public ID:\n%s", output.String())
	}
}

func TestMachineUserCreateExample(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Output: &output, Error: &output})
	command.SetArgs([]string{"machine-users", "create", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `{"machinePublicId":"badmach8k2q5n","username":"tyler","isPreferred":true}`) {
		t.Fatalf("machine user help omitted create example:\n%s", output.String())
	}
}
