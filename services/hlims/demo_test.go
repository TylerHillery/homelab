package hlims

import (
	"bytes"
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
)

func TestSeedDemoInventoryBuildsTopologyAndIsIdempotent(t *testing.T) {
	store, err := NewSQLiteDB(filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := seedDemoInventory(ctx, store); err != nil {
		t.Fatal(err)
	}
	if err := seedDemoInventory(ctx, store); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	topology, err := (apiServer{db: store.db, queries: store.queries}).topology(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(topology.Providers) != 2 {
		t.Fatalf("providers = %d; want 2", len(topology.Providers))
	}
	if !topology.Providers[0].HasLogo || !topology.Providers[1].HasLogo {
		t.Fatalf("demo providers do not have logos: %#v", topology.Providers)
	}
	firstLogo, err := store.queries.GetMachineProviderLogo(ctx, topology.Providers[0].PublicId)
	if err != nil {
		t.Fatal(err)
	}
	secondLogo, err := store.queries.GetMachineProviderLogo(ctx, topology.Providers[1].PublicId)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(firstLogo.ImageData, secondLogo.ImageData) {
		t.Fatal("demo provider logos are identical")
	}
	if got := consoleMachineCount(topology.Providers); got != 7 {
		t.Fatalf("machines = %d; want 7", got)
	}
	if got := consoleInstanceCount(topology.Providers); got != 18 {
		t.Fatalf("instances = %d; want 18", got)
	}
	homelab := topology.Providers[1]
	if len(homelab.Areas) != 1 || homelab.Areas[0].Name != "Home" {
		t.Fatalf("Homelab areas = %#v", homelab.Areas)
	}
	badger := homelab.Areas[0].Machines[0]
	if badger.Name != "Badger" || !badger.IsFavorite || len(badger.Services) != 6 || len(badger.Children) != 2 {
		t.Fatalf("Badger topology = %#v", badger)
	}
	foundGrafana := false
	for _, service := range badger.Services {
		if service.Name == "Grafana" {
			foundGrafana = true
			if len(service.Instances) != 2 || service.Instances[0].Name != "Production" || service.Instances[0].Port == nil || *service.Instances[0].Port != 3000 || service.Instances[1].Name != "Sandbox" || service.Instances[1].Port == nil || *service.Instances[1].Port != 3010 {
				t.Fatalf("Badger Grafana instances = %#v", service.Instances)
			}
			break
		}
	}
	if !foundGrafana {
		t.Fatal("Badger does not run Grafana")
	}
	atlas := badger.Children[0]
	if atlas.Name != "Atlas" || !atlas.IsFavorite || len(atlas.Services) != 2 || len(atlas.Children) != 1 {
		t.Fatalf("Atlas topology = %#v", atlas)
	}
	northstar := topology.Providers[0].Areas[0].Machines[0]
	if northstar.Name != "Northstar" || !northstar.IsFavorite {
		t.Fatalf("Northstar topology = %#v", northstar)
	}
	usersByMachine := make(map[string][]api.TopologyMachineUser)
	for _, provider := range topology.Providers {
		for _, area := range provider.Areas {
			collectDemoMachineUsers(area.Machines, usersByMachine)
		}
	}
	if len(usersByMachine) != 3 {
		t.Fatalf("machines with demo users = %#v; want Badger, Brewer, and Buck only", usersByMachine)
	}
	for _, name := range []string{"Brewer", "Buck"} {
		users := usersByMachine[name]
		if len(users) != 1 || users[0].Username != "tyler" || !users[0].IsPreferred {
			t.Fatalf("%s users = %#v; want preferred tyler", name, users)
		}
	}
	badgerUsers := usersByMachine["Badger"]
	if len(badgerUsers) != 3 || badgerUsers[0].Username != "tyler" || !badgerUsers[0].IsPreferred || badgerUsers[1].Username != "deploy" || badgerUsers[1].IsPreferred || badgerUsers[2].Username != "root" || badgerUsers[2].IsPreferred {
		t.Fatalf("Badger users = %#v; want preferred tyler, deploy, and root", badgerUsers)
	}
}

func collectDemoMachineUsers(machines []api.TopologyMachine, result map[string][]api.TopologyMachineUser) {
	for _, machine := range machines {
		if len(machine.Users) > 0 {
			result[machine.Name] = machine.Users
		}
		collectDemoMachineUsers(machine.Children, result)
	}
}

func TestInMemoryDemoSupportsConcurrentRequests(t *testing.T) {
	store, err := NewSQLiteDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if store.db.Stats().MaxOpenConnections != 1 {
		t.Fatalf("maximum connections = %d; want 1", store.db.Stats().MaxOpenConnections)
	}
	if err := seedDemoInventory(context.Background(), store); err != nil {
		t.Fatal(err)
	}

	server := apiServer{db: store.db, queries: store.queries}
	errors := make(chan error, 12)
	var requests sync.WaitGroup
	for range 12 {
		requests.Add(1)
		go func() {
			defer requests.Done()
			_, err := server.topology(context.Background())
			errors <- err
		}()
	}
	requests.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent topology request: %v", err)
		}
	}
}
