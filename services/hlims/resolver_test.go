package hlims

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func TestResolver(t *testing.T) {
	db := newResolverTestDB(t)
	resolver := resolver{queries: db.queries}

	t.Run("preferred instance endpoint", func(t *testing.T) {
		destination, err := resolver.instance(
			context.Background(),
			"badger",
			"opencode",
			"production",
			"",
			"session",
			url.Values{"view": {"full"}, "via": {"tailnet"}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if destination.URL != "https://badger.example.ts.net/opencode/session?view=full" {
			t.Fatalf("URL = %q", destination.URL)
		}
		if destination.Via != "tailnet" {
			t.Fatalf("via = %q; want tailnet", destination.Via)
		}
	})

	t.Run("LAN instance endpoint", func(t *testing.T) {
		destination, err := resolver.instance(
			context.Background(),
			"badger",
			"opencode",
			"production",
			"lan",
			"",
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		if destination.URL != "http://192.168.68.60:4096/" {
			t.Fatalf("URL = %q", destination.URL)
		}
	})

	t.Run("ad hoc machine port", func(t *testing.T) {
		destination, err := resolver.machinePort(
			context.Background(),
			"badger",
			5173,
			"",
			"",
			"src",
			url.Values{"scheme": {"http"}, "x": {"1"}},
		)
		if err != nil {
			t.Fatal(err)
		}
		if destination.URL != "http://badger.example.ts.net:5173/src?x=1" {
			t.Fatalf("URL = %q", destination.URL)
		}
	})
}

func TestEndpointURLHostSelection(t *testing.T) {
	t.Parallel()
	target := endpoint{
		address: "100.64.0.1", dnsName: sql.NullString{String: "firewall.example.ts.net", Valid: true},
		scheme: "https", port: 443,
	}
	for _, test := range []struct {
		mode, want string
	}{
		{"auto", "https://firewall.example.ts.net/"},
		{"dns", "https://firewall.example.ts.net/"},
		{"ip", "https://100.64.0.1/"},
	} {
		target.hostType = test.mode
		got, err := endpointURL(target, "", nil)
		if err != nil || got != test.want {
			t.Fatalf("hostType %s: URL = %q, error = %v; want %q", test.mode, got, err, test.want)
		}
	}
	target.hostType = "dns"
	target.dnsName = sql.NullString{}
	if _, err := endpointURL(target, "", nil); err == nil {
		t.Fatal("DNS endpoint without a name must not fall back to an IP URL")
	}
}

func TestRedirectAndAPIHandlers(t *testing.T) {
	db := newResolverTestDB(t)
	handler := newHandler(db)
	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusPermanentRedirect || root.Header().Get("Location") != "/console/" {
		t.Fatalf("root redirect = %d %q; want 308 /console/", root.Code, root.Header().Get("Location"))
	}

	request := httptest.NewRequest(http.MethodGet, "/badger/opencode/production/dashboard?via=lan&org=1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusFound {
		t.Fatalf("redirect status = %d; want %d", response.Code, http.StatusFound)
	}
	if location := response.Header().Get("Location"); location != "http://192.168.68.60:4096/dashboard?org=1" {
		t.Fatalf("Location = %q", location)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/resolve/badger/5173?via=tailnet", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("API status = %d; want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var destination api.ResolvedDestination
	if err := json.NewDecoder(response.Body).Decode(&destination); err != nil {
		t.Fatal(err)
	}
	if destination.Url != "http://badger.example.ts.net:5173/" || destination.Via != api.ViaTailnet {
		t.Fatalf("destination = %+v", destination)
	}
}

func newResolverTestDB(t *testing.T) *SQLiteDB {
	t.Helper()
	db, err := NewSQLiteDB(filepath.Join(t.TempDir(), "resolver.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	providerID, providerPublicID := testIdentifiers(t)
	areaID, areaPublicID := testIdentifiers(t)
	machineID, machinePublicID := testIdentifiers(t)
	lanID, lanPublicID := testIdentifiers(t)
	tailnetID, tailnetPublicID := testIdentifiers(t)
	lanAddressID, lanAddressPublicID := testIdentifiers(t)
	tailnetAddressID, tailnetAddressPublicID := testIdentifiers(t)
	serviceID, servicePublicID := testIdentifiers(t)
	instanceID, instancePublicID := testIdentifiers(t)
	lanEndpointID, lanEndpointPublicID := testIdentifiers(t)
	tailnetEndpointID, tailnetEndpointPublicID := testIdentifiers(t)

	ctx := context.Background()
	_, err = db.queries.CreateMachineProvider(ctx, database.CreateMachineProviderParams{ID: providerID, PublicID: providerPublicID, Name: "Homelab", Slug: "homelab"})
	mustSucceed(t, err)
	_, err = db.queries.CreateArea(ctx, database.CreateAreaParams{ID: areaID, PublicID: areaPublicID, MachineProviderID: providerID, Name: "Primary Home", Slug: "primary-home"})
	mustSucceed(t, err)
	_, err = db.queries.CreateMachine(ctx, database.CreateMachineParams{ID: machineID, PublicID: machinePublicID, MachineProviderID: providerID, AreaID: areaID, Name: "Badger", Slug: "badger", Kind: "bare_metal", Hostname: sql.NullString{String: "badger", Valid: true}})
	mustSucceed(t, err)
	_, err = db.queries.CreateNetwork(ctx, database.CreateNetworkParams{ID: lanID, PublicID: lanPublicID, AreaID: sql.NullString{String: areaID, Valid: true}, Name: "Home LAN", Slug: "home-lan", Kind: "lan"})
	mustSucceed(t, err)
	_, err = db.queries.CreateNetwork(ctx, database.CreateNetworkParams{ID: tailnetID, PublicID: tailnetPublicID, Name: "Personal Tailnet", Slug: "personal-tailnet", Kind: "tailnet"})
	mustSucceed(t, err)
	_, err = db.queries.CreateAddress(ctx, database.CreateAddressParams{ID: lanAddressID, PublicID: lanAddressPublicID, NetworkID: lanID, MachineID: sql.NullString{String: machineID, Valid: true}, Name: sql.NullString{String: "LAN", Valid: true}, Address: "192.168.68.60", IsPrimary: 1})
	mustSucceed(t, err)
	_, err = db.queries.CreateAddress(ctx, database.CreateAddressParams{ID: tailnetAddressID, PublicID: tailnetAddressPublicID, NetworkID: tailnetID, MachineID: sql.NullString{String: machineID, Valid: true}, Name: sql.NullString{String: "Tailscale", Valid: true}, Address: "100.82.5.73", DnsName: sql.NullString{String: "badger.example.ts.net", Valid: true}, IsPrimary: 1})
	mustSucceed(t, err)
	_, err = db.queries.CreateService(ctx, database.CreateServiceParams{ID: serviceID, PublicID: servicePublicID, Name: "OpenCode", Slug: "opencode"})
	mustSucceed(t, err)
	_, err = db.queries.CreateInstance(ctx, database.CreateInstanceParams{ID: instanceID, PublicID: instancePublicID, ServiceID: serviceID, MachineID: machineID, Name: "Production", Slug: "production", Port: 4096})
	mustSucceed(t, err)
	_, err = db.queries.CreateInstanceEndpoint(ctx, database.CreateInstanceEndpointParams{ID: lanEndpointID, PublicID: lanEndpointPublicID, InstanceID: instanceID, AddressID: lanAddressID, Name: "LAN", Scheme: "http", Port: 4096, HostType: "auto"})
	mustSucceed(t, err)
	_, err = db.queries.CreateInstanceEndpoint(ctx, database.CreateInstanceEndpointParams{ID: tailnetEndpointID, PublicID: tailnetEndpointPublicID, InstanceID: instanceID, AddressID: tailnetAddressID, Name: "Tailscale Serve", Scheme: "https", Port: 443, BasePath: "/opencode", HostType: "auto", IsPreferred: 1})
	mustSucceed(t, err)

	return db
}

func mustSucceed(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
