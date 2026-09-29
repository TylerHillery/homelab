package hlims

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"

	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type demoRecord struct {
	id       string
	publicID string
}

func seedDemoInventory(ctx context.Context, store *SQLiteDB) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	queries := store.queries.WithTx(tx)
	existing, err := queries.ListMachineProviders(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return tx.Commit()
	}

	homelab, err := createDemoProvider(ctx, queries, "Homelab", "homelab", color.RGBA{R: 43, G: 108, B: 176, A: 255}, 7)
	if err != nil {
		return err
	}
	cloud, err := createDemoProvider(ctx, queries, "Cloud Lab", "cloud-lab", color.RGBA{R: 117, G: 77, B: 168, A: 255}, 8)
	if err != nil {
		return err
	}
	primary, err := createDemoArea(ctx, queries, homelab, "Home", "home", "HOME")
	if err != nil {
		return err
	}
	region, err := createDemoArea(ctx, queries, cloud, "Falkenstein / FSN1", "falkenstein-fsn1", "FSN1")
	if err != nil {
		return err
	}

	badger, err := createDemoMachine(ctx, queries, demoMachine{
		provider: homelab, area: primary, name: "Badger", slug: "badger", kind: "bare_metal",
		favorite: true,
		hostname: "badger.lan", os: "Ubuntu Server", osVersion: "24.04.3 LTS", kernel: "6.8.0", architecture: "x86_64",
		cpuCount: 4, cpuAllocation: "dedicated", cpuVendor: "Intel", memoryBytes: 8 * 1024 * 1024 * 1024,
		storageBytes: 250_000_000_000, storageMedia: "ssd", storageInterface: "sata",
	})
	if err != nil {
		return err
	}
	atlas, err := createDemoMachine(ctx, queries, demoMachine{
		provider: homelab, area: primary, parent: badger, name: "Atlas", slug: "atlas", kind: "virtual_machine",
		favorite:       true,
		virtualization: "KVM", hostname: "atlas.internal", os: "Debian", osVersion: "13", architecture: "x86_64",
		cpuCount: 2, cpuAllocation: "shared", memoryBytes: 4 * 1024 * 1024 * 1024, storageBytes: 80_000_000_000,
		storageMedia: "ssd", storageInterface: "virtio",
	})
	if err != nil {
		return err
	}
	devbox, err := createDemoMachine(ctx, queries, demoMachine{
		provider: homelab, area: primary, parent: atlas, name: "Atlas Devbox", slug: "atlas-devbox", kind: "virtual_machine",
		virtualization: "systemd-nspawn", hostname: "devbox.internal", os: "Debian", osVersion: "13", architecture: "x86_64",
		cpuCount: 1, cpuAllocation: "shared", memoryBytes: 2 * 1024 * 1024 * 1024, storageBytes: 24_000_000_000,
		storageMedia: "ssd", storageInterface: "virtual",
	})
	if err != nil {
		return err
	}
	homeAssistant, err := createDemoMachine(ctx, queries, demoMachine{
		provider: homelab, area: primary, parent: badger, name: "Home Automation", slug: "home-automation", kind: "virtual_machine",
		virtualization: "KVM", hostname: "homeassistant.lan", os: "Home Assistant OS", osVersion: "15.2", architecture: "x86_64",
		cpuCount: 2, cpuAllocation: "shared", memoryBytes: 3 * 1024 * 1024 * 1024, storageBytes: 32_000_000_000,
		storageMedia: "ssd", storageInterface: "virtio",
	})
	if err != nil {
		return err
	}
	brewer, err := createDemoMachine(ctx, queries, demoMachine{
		provider: homelab, area: primary, name: "Brewer", slug: "brewer", kind: "bare_metal",
		hostname: "brewer.lan", os: "Ubuntu Server", osVersion: "24.04.2 LTS", kernel: "6.8.0", architecture: "x86_64",
		cpuCount: 4, cpuAllocation: "dedicated", cpuVendor: "Intel", memoryBytes: 8 * 1024 * 1024 * 1024,
		storageBytes: 250_000_000_000, storageMedia: "ssd", storageInterface: "sata",
	})
	if err != nil {
		return err
	}
	buck, err := createDemoMachine(ctx, queries, demoMachine{
		provider: homelab, area: primary, name: "Buck", slug: "buck", kind: "bare_metal",
		hostname: "buck.lan", os: "Ubuntu Server", osVersion: "22.04.5 LTS", kernel: "5.15.0", architecture: "x86_64",
		cpuCount: 4, cpuAllocation: "dedicated", cpuVendor: "Intel", memoryBytes: 8 * 1024 * 1024 * 1024,
		storageBytes: 250_000_000_000, storageMedia: "ssd", storageInterface: "sata",
	})
	if err != nil {
		return err
	}
	for _, machine := range []demoRecord{badger, brewer, buck} {
		if err := createDemoMachineUser(ctx, queries, machine, "tyler", true); err != nil {
			return err
		}
	}
	for _, username := range []string{"deploy", "root"} {
		if err := createDemoMachineUser(ctx, queries, badger, username, false); err != nil {
			return err
		}
	}
	edge, err := createDemoMachine(ctx, queries, demoMachine{
		provider: cloud, area: region, name: "Northstar", slug: "northstar", kind: "virtual_machine",
		favorite: true,
		hostname: "northstar.example.net", os: "Ubuntu", osVersion: "24.04 LTS", architecture: "x86_64",
		cpuCount: 2, cpuAllocation: "shared", memoryBytes: 4 * 1024 * 1024 * 1024, storageBytes: 80_000_000_000,
		storageMedia: "ssd", storageInterface: "virtual", monthlyCost: 799, currency: "USD",
	})
	if err != nil {
		return err
	}
	grafana, err := createDemoService(ctx, queries, "Grafana", "grafana", "Infrastructure dashboards", color.RGBA{R: 238, G: 92, B: 38, A: 255}, 1)
	if err != nil {
		return err
	}
	prometheus, err := createDemoService(ctx, queries, "Prometheus", "prometheus", "Metrics collection and alerting", color.RGBA{R: 194, G: 43, B: 35, A: 255}, 2)
	if err != nil {
		return err
	}
	opencode, err := createDemoService(ctx, queries, "OpenCode", "opencode", "Remote development workspaces", color.RGBA{R: 30, G: 118, B: 84, A: 255}, 3)
	if err != nil {
		return err
	}
	gitea, err := createDemoService(ctx, queries, "Gitea", "gitea", "Private source control", color.RGBA{R: 93, G: 135, B: 76, A: 255}, 4)
	if err != nil {
		return err
	}
	haService, err := createDemoService(ctx, queries, "Home Assistant", "home-assistant", "Home automation control plane", color.RGBA{R: 42, G: 169, B: 224, A: 255}, 5)
	if err != nil {
		return err
	}
	uptime, err := createDemoService(ctx, queries, "Uptime Kuma", "uptime-kuma", "Service availability monitoring", color.RGBA{R: 70, G: 184, B: 131, A: 255}, 6)
	if err != nil {
		return err
	}

	homeLAN, err := createDemoNetwork(ctx, queries, "Home LAN", "home-lan", "lan", primary)
	if err != nil {
		return err
	}
	tailnet, err := createDemoNetwork(ctx, queries, "Personal Tailnet", "personal-tailnet", "tailnet", demoRecord{})
	if err != nil {
		return err
	}
	public, err := createDemoNetwork(ctx, queries, "Public Internet", "public-internet", "public", demoRecord{})
	if err != nil {
		return err
	}

	badgerAddress, err := createDemoAddress(ctx, queries, homeLAN, badger, "LAN", "192.168.50.20", "badger.lan")
	if err != nil {
		return err
	}
	atlasAddress, err := createDemoAddress(ctx, queries, homeLAN, atlas, "LAN", "192.168.50.31", "atlas.internal")
	if err != nil {
		return err
	}
	devboxAddress, err := createDemoAddress(ctx, queries, tailnet, devbox, "Tailnet", "100.64.0.42", "atlas-devbox.demo.ts.net")
	if err != nil {
		return err
	}
	homeAssistantAddress, err := createDemoAddress(ctx, queries, homeLAN, homeAssistant, "LAN", "192.168.50.40", "homeassistant.lan")
	if err != nil {
		return err
	}
	brewerAddress, err := createDemoAddress(ctx, queries, homeLAN, brewer, "LAN", "192.168.50.21", "brewer.lan")
	if err != nil {
		return err
	}
	edgeAddress, err := createDemoAddress(ctx, queries, public, edge, "Public", "203.0.113.17", "northstar.example.net")
	if err != nil {
		return err
	}

	instances := []struct {
		service, machine, address demoRecord
		name, slug                string
		port                      int64
		scheme                    string
	}{
		{grafana, badger, badgerAddress, "Production", "production", 3000, "http"},
		{grafana, badger, badgerAddress, "Sandbox", "sandbox", 3010, "http"},
		{prometheus, badger, badgerAddress, "Primary", "primary", 9090, "http"},
		{gitea, badger, badgerAddress, "Production", "production", 3001, "http"},
		{opencode, atlas, atlasAddress, "Workspace", "workspace", 4096, "http"},
		{grafana, atlas, atlasAddress, "Development", "development", 3000, "http"},
		{gitea, devbox, devboxAddress, "Actions Runner", "actions-runner", 8080, "http"},
		{haService, homeAssistant, homeAssistantAddress, "Home", "home", 8123, "http"},
		{uptime, brewer, brewerAddress, "Production", "production", 3001, "http"},
		{grafana, edge, edgeAddress, "Staging", "staging", 443, "https"},
	}
	for _, item := range instances {
		if err := createDemoInstance(ctx, queries, item.service, item.machine, item.address, item.name, item.slug, item.port, item.scheme); err != nil {
			return err
		}
	}
	postgres, err := createDemoService(ctx, queries, "PostgreSQL", "postgresql", "Database instances in separate deployments", color.RGBA{R: 51, G: 103, B: 145, A: 255}, 2)
	if err != nil {
		return err
	}
	app, err := createDemoService(ctx, queries, "Example API", "example-api", "Application using shared infrastructure", color.RGBA{R: 107, G: 80, B: 175, A: 255}, 4)
	if err != nil {
		return err
	}
	agent, err := createDemoService(ctx, queries, "Metrics Agent", "metrics-agent", "Host-level telemetry process", color.RGBA{R: 82, G: 157, B: 109, A: 255}, 5)
	if err != nil {
		return err
	}
	watcher, err := createDemoService(ctx, queries, "Log Watcher", "log-watcher", "Another host-level systemd unit", color.RGBA{R: 70, G: 105, B: 133, A: 255}, 6)
	if err != nil {
		return err
	}
	docs, err := createDemoService(ctx, queries, "Example Docs", "example-docs", "Static docs served by the example API", color.RGBA{R: 34, G: 126, B: 149, A: 255}, 3)
	if err != nil {
		return err
	}
	shared, err := createDemoDeployment(ctx, queries, badger, "Shared Infrastructure", "shared-infrastructure", "/srv/stacks/shared", "shared", []string{"compose.yaml"})
	if err != nil {
		return err
	}
	web, err := createDemoDeployment(ctx, queries, badger, "Web Application", "web-application", "/srv/apps/example", "example", []string{"compose.yaml", "compose.local.yaml"})
	if err != nil {
		return err
	}
	sharedDB, err := createDemoBareInstance(ctx, queries, postgres, badger, shared, "Shared DB", "shared-db", "db")
	if err != nil {
		return err
	}
	if _, err := createDemoBareInstance(ctx, queries, postgres, badger, web, "App DB", "app-db", "db"); err != nil {
		return err
	}
	for _, local := range []struct{ name, slug string }{{"Local CLI A", "local-cli-a"}, {"Local CLI B", "local-cli-b"}} {
		if _, err := createDemoBareInstance(ctx, queries, postgres, badger, demoRecord{}, local.name, local.slug, ""); err != nil {
			return err
		}
	}
	appAPI, err := createDemoBareInstance(ctx, queries, app, badger, web, "Production", "production", "api")
	if err != nil {
		return err
	}
	if _, err := createDemoSystemdInstance(ctx, queries, agent, brewer, "Primary", "primary", "metrics-agent.service"); err != nil {
		return err
	}
	if _, err := createDemoSystemdInstance(ctx, queries, watcher, brewer, "Primary", "primary", "log-watcher.service"); err != nil {
		return err
	}
	dependency, err := newDemoRecord()
	if err != nil {
		return err
	}
	if _, err := queries.CreateInstanceDependency(ctx, database.CreateInstanceDependencyParams{
		ID: dependency.id, PublicID: dependency.publicID,
		ConsumerInstanceID: appAPI.id, ProviderInstanceID: sharedDB.id, Kind: "uses",
	}); err != nil {
		return fmt.Errorf("create shared database usage: %w", err)
	}
	staticDocs, err := newDemoRecord()
	if err != nil {
		return err
	}
	if _, err := queries.CreateInstance(ctx, database.CreateInstanceParams{
		ID: staticDocs.id, PublicID: staticDocs.publicID, ServiceID: docs.id,
		MachineID: nullableDemoID(badger), DeploymentID: nullableDemoID(web),
		DeploymentMember: nullableDemoString("site/dist"), DeploymentRole: "static_content",
		HostingKind: "machine", Name: "Site", Slug: "site",
	}); err != nil {
		return fmt.Errorf("create static documentation instance: %w", err)
	}
	publication, err := newDemoRecord()
	if err != nil {
		return err
	}
	if _, err := queries.CreateInstanceDependency(ctx, database.CreateInstanceDependencyParams{
		ID: publication.id, PublicID: publication.publicID,
		ConsumerInstanceID: staticDocs.id, ProviderInstanceID: appAPI.id, Kind: "served_by",
	}); err != nil {
		return fmt.Errorf("create documentation publication: %w", err)
	}

	return tx.Commit()
}

func createDemoDeployment(ctx context.Context, queries *database.Queries, machine demoRecord, name, slug, directory, project string, files []string) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	params := database.CreateDeploymentParams{
		ID: record.id, PublicID: record.publicID, MachineID: machine.id,
		Name: name, Slug: slug, WorkingDirectory: directory, ComposeProject: project,
	}
	if files != nil {
		encoded, err := json.Marshal(files)
		if err != nil {
			return record, err
		}
		params.ComposeFiles = string(encoded)
	}
	_, err = queries.CreateDeployment(ctx, params)
	return demoCreateResult(record, "deployment "+name, err)
}

func createDemoSystemdInstance(ctx context.Context, queries *database.Queries, service, machine demoRecord, name, slug, unit string) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateInstance(ctx, database.CreateInstanceParams{
		ID: record.id, PublicID: record.publicID, ServiceID: service.id, MachineID: nullableDemoID(machine),
		DeploymentRole: "service", SystemdUnit: nullableDemoString(unit), SystemdScope: nullableDemoString("system"),
		HostingKind: "machine", Name: name, Slug: slug,
	})
	return demoCreateResult(record, "systemd instance "+name, err)
}

func createDemoBareInstance(ctx context.Context, queries *database.Queries, service, machine, deployment demoRecord, name, slug, member string) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateInstance(ctx, database.CreateInstanceParams{
		ID: record.id, PublicID: record.publicID, ServiceID: service.id,
		MachineID: nullableDemoID(machine), DeploymentID: nullableDemoID(deployment),
		DeploymentMember: nullableDemoString(member), DeploymentRole: "service", HostingKind: "machine", Name: name, Slug: slug,
	})
	return demoCreateResult(record, "instance "+name, err)
}

func newDemoRecord() (demoRecord, error) {
	id, publicID, err := newIDs()
	return demoRecord{id: id, publicID: publicID}, err
}

func createDemoProvider(ctx context.Context, queries *database.Queries, name, slug string, tint color.RGBA, pattern int) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateMachineProvider(ctx, database.CreateMachineProviderParams{ID: record.id, PublicID: record.publicID, Name: name, Slug: slug})
	if err != nil {
		return record, fmt.Errorf("create provider %s: %w", name, err)
	}
	logo, err := demoLogo(tint, pattern)
	if err != nil {
		return record, err
	}
	err = queries.UpsertMachineProviderLogo(ctx, database.UpsertMachineProviderLogoParams{ContentType: "image/png", ImageData: logo, PublicID: record.publicID})
	return demoCreateResult(record, "provider "+name, err)
}

func createDemoArea(ctx context.Context, queries *database.Queries, provider demoRecord, name, slug, code string) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateArea(ctx, database.CreateAreaParams{ID: record.id, PublicID: record.publicID, MachineProviderID: provider.id, Name: name, Slug: slug, ProviderCode: sql.NullString{String: code, Valid: true}})
	return demoCreateResult(record, "area "+name, err)
}

type demoMachine struct {
	provider, area, parent                 demoRecord
	name, slug, kind                       string
	virtualization, hostname               string
	os, osVersion, kernel, architecture    string
	cpuAllocation, cpuVendor               string
	storageMedia, storageInterface         string
	currency                               string
	cpuCount                               int64
	memoryBytes, storageBytes, monthlyCost int64
	favorite                               bool
}

func createDemoMachine(ctx context.Context, queries *database.Queries, value demoMachine) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	params := database.CreateMachineParams{
		ID: record.id, PublicID: record.publicID, MachineProviderID: value.provider.id, AreaID: value.area.id,
		Name: value.name, Slug: value.slug, Kind: value.kind, IsFavorite: value.favorite,
		ParentMachineID: nullableDemoID(value.parent), VirtualizationPlatform: nullableDemoString(value.virtualization),
		Hostname: nullableDemoString(value.hostname), OperatingSystem: nullableDemoString(value.os),
		OperatingSystemVersion: nullableDemoString(value.osVersion), Kernel: nullableDemoString(value.kernel),
		Architecture: nullableDemoString(value.architecture), CpuCount: nullableDemoInt(value.cpuCount),
		CpuAllocation: nullableDemoString(value.cpuAllocation), CpuVendor: nullableDemoString(value.cpuVendor),
		MemoryBytes: nullableDemoInt(value.memoryBytes), StorageBytes: nullableDemoInt(value.storageBytes),
		StorageMediaKind: nullableDemoString(value.storageMedia), StorageInterfaceKind: nullableDemoString(value.storageInterface),
		EstimatedMonthlyCostCents: nullableDemoInt(value.monthlyCost), CostCurrency: nullableDemoString(value.currency),
	}
	_, err = queries.CreateMachine(ctx, params)
	return demoCreateResult(record, "machine "+value.name, err)
}

func createDemoService(ctx context.Context, queries *database.Queries, name, slug, description string, tint color.RGBA, pattern int) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateService(ctx, database.CreateServiceParams{ID: record.id, PublicID: record.publicID, Name: name, Slug: slug, Description: nullableDemoString(description)})
	if err != nil {
		return record, fmt.Errorf("create service %s: %w", name, err)
	}
	logo, err := demoLogo(tint, pattern)
	if err != nil {
		return record, err
	}
	err = queries.UpsertServiceLogo(ctx, database.UpsertServiceLogoParams{ContentType: "image/png", ImageData: logo, PublicID: record.publicID})
	return demoCreateResult(record, "logo for "+name, err)
}

func createDemoNetwork(ctx context.Context, queries *database.Queries, name, slug, kind string, area demoRecord) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateNetwork(ctx, database.CreateNetworkParams{ID: record.id, PublicID: record.publicID, AreaID: nullableDemoID(area), Name: name, Slug: slug, Kind: kind})
	return demoCreateResult(record, "network "+name, err)
}

func createDemoAddress(ctx context.Context, queries *database.Queries, network, machine demoRecord, name, address, dnsName string) (demoRecord, error) {
	record, err := newDemoRecord()
	if err != nil {
		return record, err
	}
	_, err = queries.CreateAddress(ctx, database.CreateAddressParams{
		ID: record.id, PublicID: record.publicID, NetworkID: network.id, MachineID: nullableDemoID(machine),
		Name: nullableDemoString(name), Address: address, DnsName: nullableDemoString(dnsName), IsPrimary: 1,
	})
	return demoCreateResult(record, "address for "+machine.publicID, err)
}

func createDemoMachineUser(ctx context.Context, queries *database.Queries, machine demoRecord, username string, preferred bool) error {
	record, err := newDemoRecord()
	if err != nil {
		return err
	}
	preferredValue := int64(0)
	if preferred {
		preferredValue = 1
	}
	_, err = queries.CreateMachineUser(ctx, database.CreateMachineUserParams{
		ID: record.id, PublicID: record.publicID, MachineID: machine.id, Username: username, IsPreferred: preferredValue,
	})
	if err != nil {
		return fmt.Errorf("create machine user %s for %s: %w", username, machine.publicID, err)
	}
	return nil
}

func createDemoInstance(ctx context.Context, queries *database.Queries, service, machine, address demoRecord, name, slug string, port int64, scheme string) error {
	instance, err := newDemoRecord()
	if err != nil {
		return err
	}
	_, err = queries.CreateInstance(ctx, database.CreateInstanceParams{ID: instance.id, PublicID: instance.publicID, ServiceID: service.id, MachineID: sql.NullString{String: machine.id, Valid: true}, DeploymentRole: "service", HostingKind: "machine", Name: name, Slug: slug, Port: sql.NullInt64{Int64: port, Valid: true}})
	if err != nil {
		return fmt.Errorf("create instance %s: %w", name, err)
	}
	endpoint, err := newDemoRecord()
	if err != nil {
		return err
	}
	_, err = queries.CreateInstanceEndpoint(ctx, database.CreateInstanceEndpointParams{
		ID: endpoint.id, PublicID: endpoint.publicID, InstanceID: instance.id, AddressID: sql.NullString{String: address.id, Valid: true},
		Name: "Preferred", Scheme: sql.NullString{String: scheme, Valid: true}, Port: sql.NullInt64{Int64: port, Valid: true}, HostType: "auto", IsPreferred: 1,
	})
	if err != nil {
		return fmt.Errorf("create endpoint for %s: %w", name, err)
	}
	return nil
}

func demoCreateResult(record demoRecord, label string, err error) (demoRecord, error) {
	if err != nil {
		return record, fmt.Errorf("create %s: %w", label, err)
	}
	return record, nil
}

func nullableDemoString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullableDemoID(value demoRecord) sql.NullString {
	return nullableDemoString(value.id)
}

func nullableDemoInt(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: value != 0}
}

func demoLogo(tint color.RGBA, pattern int) ([]byte, error) {
	logo := image.NewRGBA(image.Rect(0, 0, 64, 64))
	dark := color.RGBA{R: 21, G: 32, B: 29, A: 255}
	light := color.RGBA{R: 250, G: 249, B: 243, A: 255}
	for y := range 64 {
		for x := range 64 {
			pixel := tint
			if x < 7 || y < 7 || x >= 57 || y >= 57 {
				pixel = dark
			} else if (x+y+pattern*7)%19 < 4 || (x*pattern+y)%29 < 3 {
				pixel = light
			}
			logo.SetRGBA(x, y, pixel)
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, logo); err != nil {
		return nil, fmt.Errorf("encode demo logo: %w", err)
	}
	return encoded.Bytes(), nil
}
