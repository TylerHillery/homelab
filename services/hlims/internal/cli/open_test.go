package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseCanonicalOpenTarget(t *testing.T) {
	t.Parallel()
	target, err := parseOpenTarget("go/grafana/production/d/overview?host=badger&via=tailnet&refresh=30s#panel")
	if err != nil {
		t.Fatal(err)
	}
	if target.host != "badger" || target.service != "grafana" || target.instance != "production" {
		t.Fatalf("target = %#v", target)
	}
	if target.via != "tailnet" || target.query.Get("via") != "" || target.query.Get("host") != "" || target.query.Get("refresh") != "30s" {
		t.Fatalf("target query = %#v", target)
	}
	if target.fragment != "panel" || len(target.suffix) != 2 {
		t.Fatalf("target suffix/fragment = %#v", target)
	}
}

func TestParsePortOpenTarget(t *testing.T) {
	t.Parallel()
	target, err := parseOpenTarget("https://go/badger/5173/src/status?scheme=https")
	if err != nil {
		t.Fatal(err)
	}
	if target.machine != "badger" || target.port != 5173 || target.scheme != "https" {
		t.Fatalf("target = %#v", target)
	}
}

func TestAppendOpenTargetPreservesDestinationAndCallerQuery(t *testing.T) {
	t.Parallel()
	destination, err := appendOpenTarget("https://badger.example.ts.net/grafana?orgId=1", openTarget{
		suffix:   []string{"d", "overview"},
		query:    url.Values{"refresh": []string{"30s"}},
		fragment: "panel",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://badger.example.ts.net/grafana/d/overview?orgId=1&refresh=30s#panel"
	if destination != want {
		t.Fatalf("destination = %q, want %q", destination, want)
	}
}

func TestOpenTargetPreservesEscapedSuffix(t *testing.T) {
	t.Parallel()
	target, err := parseOpenTarget("grafana/production/d/a%2Fb")
	if err != nil {
		t.Fatal(err)
	}
	destination, err := appendOpenTarget("https://badger.example.ts.net/grafana", target)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://badger.example.ts.net/grafana/d/a%2Fb"
	if destination != want {
		t.Fatalf("destination = %q, want %q", destination, want)
	}
}

func TestCanonicalOpenTargetForwardsSchemeQuery(t *testing.T) {
	t.Parallel()
	target, err := parseOpenTarget("grafana/production?scheme=dark")
	if err != nil {
		t.Fatal(err)
	}
	if target.scheme != "" || target.query.Get("scheme") != "dark" {
		t.Fatalf("target = %#v", target)
	}
}

func TestAppendOpenTargetPreservesResolverFragment(t *testing.T) {
	t.Parallel()
	destination, err := appendOpenTarget("https://badger.example.ts.net/app#default", openTarget{query: url.Values{}})
	if err != nil {
		t.Fatal(err)
	}
	if destination != "https://badger.example.ts.net/app#default" {
		t.Fatalf("destination = %q", destination)
	}
}

func TestParseOpenTargetRejectsNonHTTPURL(t *testing.T) {
	t.Parallel()
	if _, err := parseOpenTarget("ftp://go/grafana/production"); err == nil {
		t.Fatal("parseOpenTarget accepted a non-HTTP URL")
	}
}

func TestOpenCommandResolvesAndLaunches(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/resolve/grafana/production" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("host") != "badger" {
			t.Errorf("host = %q", request.URL.Query().Get("host"))
		}
		if request.URL.Query().Get("via") != "tailnet" {
			t.Errorf("via = %q", request.URL.Query().Get("via"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"url":"https://badger.example.ts.net/grafana","via":"tailnet"}`))
	}))
	t.Cleanup(server.Close)

	var opened string
	command := NewRootCommand(Dependencies{
		Output:     &bytes.Buffer{},
		Error:      &bytes.Buffer{},
		HTTPClient: server.Client(),
		OpenURL: func(_ context.Context, target string) error {
			opened = target
			return nil
		},
	})
	command.SetArgs([]string{
		"--api-url", server.URL + "/api/v1",
		"open", "grafana/production/d/overview?host=badger&via=tailnet&refresh=30s#panel",
	})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "https://badger.example.ts.net/grafana/d/overview?refresh=30s#panel"
	if opened != want {
		t.Fatalf("opened = %q, want %q", opened, want)
	}
}

func TestOpenPrintDoesNotLaunchBrowser(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"url":"http://badger:5173","via":"lan"}`))
	}))
	t.Cleanup(server.Close)

	var output bytes.Buffer
	command := NewRootCommand(Dependencies{
		Output:     &output,
		Error:      &bytes.Buffer{},
		HTTPClient: server.Client(),
		OpenURL: func(context.Context, string) error {
			t.Fatal("browser launcher called with --print")
			return nil
		},
	})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "open", "--print", "badger/5173"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if output.String() != "http://badger:5173\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestManagedOpenPrintPreservesProjectQuery(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/resolve/posthog/example-cloud" {
			t.Errorf("managed resolver path = %q", request.URL.Path)
		}
		if request.URL.Query().Get("host") != "managed" {
			t.Errorf("managed host = %q", request.URL.Query().Get("host"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"url":"https://app.example.test/project/42/home?origin=app","via":"managed"}`))
	}))
	t.Cleanup(server.Close)
	var output bytes.Buffer
	command := NewRootCommand(Dependencies{Output: &output, Error: &bytes.Buffer{}, HTTPClient: server.Client()})
	command.SetArgs([]string{"--api-url", server.URL + "/api/v1", "open", "--print", "posthog/example-cloud/events?host=managed&tab=insights"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "https://app.example.test/project/42/home/events?origin=app&tab=insights\n"; got != want {
		t.Fatalf("managed destination = %q; want %q", got, want)
	}
}
