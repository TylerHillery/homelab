package hlims

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func TestInventorySchema(t *testing.T) {
	store, err := NewSQLiteDB(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	providerID, providerPublicID := testIdentifiers(t)
	areaID, areaPublicID := testIdentifiers(t)
	hpID, hpPublicID := testIdentifiers(t)
	hynixID, hynixPublicID := testIdentifiers(t)
	systemProductID, systemProductPublicID := testIdentifiers(t)
	memoryProductID, memoryProductPublicID := testIdentifiers(t)
	systemAssetID, systemAssetPublicID := testIdentifiers(t)
	memoryAssetID, memoryAssetPublicID := testIdentifiers(t)
	machineID, machinePublicID := testIdentifiers(t)
	serviceID, servicePublicID := testIdentifiers(t)
	instanceID, instancePublicID := testIdentifiers(t)
	endpointID, endpointPublicID := testIdentifiers(t)
	networkID, networkPublicID := testIdentifiers(t)
	addressID, addressPublicID := testIdentifiers(t)

	execInventorySQL(t, store, `
		insert into machine_providers (id, public_id, name, slug)
		values (?, ?, 'DigitalOcean', 'digitalocean')
	`, providerID, providerPublicID)
	execInventorySQL(t, store, `
		insert into areas (id, public_id, machine_provider_id, name, slug, provider_code)
		values (?, ?, ?, 'San Francisco 3', 'san-francisco-3', 'sfo3')
	`, areaID, areaPublicID, providerID)
	execInventorySQL(t, store, `
		insert into manufacturers (id, public_id, name, slug)
		values (?, ?, 'HP', 'hp'), (?, ?, 'Hynix', 'hynix')
	`, hpID, hpPublicID, hynixID, hynixPublicID)
	execInventorySQL(t, store, `
		insert into products (id, public_id, manufacturer_id, kind, name, part_number)
		values (?, ?, ?, 'system', 'ProDesk 600 G1 DM', 'K7M40US#ABA')
	`, systemProductID, systemProductPublicID, hpID)
	execInventorySQL(t, store, `
		insert into products (id, public_id, manufacturer_id, kind, name, part_number)
		values (?, ?, ?, 'memory', '4 GiB DDR3-1600 SODIMM', 'HMT451S6AFR8A-PB')
	`, memoryProductID, memoryProductPublicID, hynixID)
	execInventorySQL(t, store, `
		insert into memory_specs (product_id, capacity_bytes, memory_type, form_factor, speed_mts)
		values (?, 4294967296, 'DDR3', 'SODIMM', 1600)
	`, memoryProductID)
	execInventorySQL(t, store, `
		insert into assets (
			id, public_id, product_id, area_id, name, serial_number, system_uuid
		) values (?, ?, ?, ?, 'Badger chassis', '2UA5351313', ?)
	`, systemAssetID, systemAssetPublicID, systemProductID, areaID, systemAssetID)
	execInventorySQL(t, store, `
		insert into assets (
			id, public_id, product_id, parent_asset_id, serial_number
		) values (?, ?, ?, ?, '01512315')
	`, memoryAssetID, memoryAssetPublicID, memoryProductID, systemAssetID)
	execInventorySQL(t, store, `
		insert into machines (
			id, public_id, machine_provider_id, area_id, name, slug, kind,
			cpu_count, cpu_allocation, memory_bytes, storage_bytes, storage_media_kind,
			estimated_monthly_cost_cents, cost_currency
		) values (
			?, ?, ?, ?, 'pypacktrends-prod', 'pypacktrends-prod', 'virtual_machine',
			1, 'shared', 2147483648, 50000000000, 'ssd', 1200, 'USD'
		)
	`, machineID, machinePublicID, providerID, areaID)
	var isFavorite bool
	if err := store.db.QueryRow(`select is_favorite from machines where id = ?`, machineID).Scan(&isFavorite); err != nil {
		t.Fatal(err)
	}
	if isFavorite {
		t.Fatal("machine is_favorite did not default to false")
	}
	assertInventorySQLFails(t, store, "set a null machine favorite", `
		update machines set is_favorite = null where id = ?
	`, machineID)
	if _, err := store.queries.CreateService(context.Background(), database.CreateServiceParams{
		ID:          serviceID,
		PublicID:    servicePublicID,
		Name:        "OpenCode",
		Slug:        "opencode",
		Description: sql.NullString{String: "AI coding agent", Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.queries.CreateInstance(context.Background(), database.CreateInstanceParams{
		ID:        instanceID,
		PublicID:  instancePublicID,
		ServiceID: serviceID,
		MachineID: machineID,
		Name:      "Production",
		Slug:      "production",
		Port:      4096,
	}); err != nil {
		t.Fatal(err)
	}
	execInventorySQL(t, store, `
		insert into networks (id, public_id, name, slug, kind, cidr)
		values (?, ?, 'Personal Tailnet', 'personal-tailnet', 'tailnet', '100.64.0.0/10')
	`, networkID, networkPublicID)
	execInventorySQL(t, store, `
		insert into addresses (
			id, public_id, network_id, machine_id, name, address, dns_name, is_primary
		) values (
			?, ?, ?, ?, 'Tailscale', '100.81.203.30',
			'pypacktrends-prod.example.ts.net', 1
		)
	`, addressID, addressPublicID, networkID, machineID)
	if _, err := store.queries.CreateInstanceEndpoint(context.Background(), database.CreateInstanceEndpointParams{
		ID:          endpointID,
		PublicID:    endpointPublicID,
		InstanceID:  instanceID,
		AddressID:   addressID,
		Name:        "Tailnet Serve",
		Scheme:      string(InstanceSchemeHTTPS),
		Port:        443,
		BasePath:    "/opencode",
		IsPreferred: 1,
	}); err != nil {
		t.Fatal(err)
	}

	var installedMemory int64
	if err := store.db.QueryRow(`
		select sum(memory_specs.capacity_bytes)
		from assets
		join memory_specs on memory_specs.product_id = assets.product_id
		where assets.parent_asset_id = ?
	`, systemAssetID).Scan(&installedMemory); err != nil {
		t.Fatal(err)
	}
	if installedMemory != 4294967296 {
		t.Fatalf("installed memory = %d; want 4294967296", installedMemory)
	}
	memoryAsset, err := store.queries.GetAssetByPublicID(context.Background(), memoryAssetPublicID)
	if err != nil {
		t.Fatal(err)
	}
	if !memoryAsset.EffectiveAreaPublicID.Valid || memoryAsset.EffectiveAreaPublicID.String != areaPublicID {
		t.Fatalf("effective area = %v; want %q", memoryAsset.EffectiveAreaPublicID, areaPublicID)
	}

	assertInventorySQLFails(t, store, "attached mismatched product specifications", `
		insert into processor_specs (product_id, core_count, thread_count)
		values (?, 1, 1)
	`, memoryProductID)
	assertInventorySQLFails(t, store, "changed a product kind that has specifications", `
		update products set kind = 'drive' where id = ?
	`, memoryProductID)

	invalidID, invalidPublicID := testIdentifiers(t)
	assertInventorySQLFails(t, store, "placed an asset in both an area and a parent", `
		insert into assets (id, public_id, product_id, parent_asset_id, area_id)
		values (?, ?, ?, ?, ?)
	`, invalidID, invalidPublicID, memoryProductID, systemAssetID, areaID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "used a non-system parent asset", `
		insert into assets (id, public_id, product_id, parent_asset_id)
		values (?, ?, ?, ?)
	`, invalidID, invalidPublicID, memoryProductID, memoryAssetID)

	secondSystemAssetID, secondSystemAssetPublicID := testIdentifiers(t)
	thirdSystemAssetID, thirdSystemAssetPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `
		insert into assets (id, public_id, product_id)
		values (?, ?, ?)
	`, secondSystemAssetID, secondSystemAssetPublicID, systemProductID)
	execInventorySQL(t, store, `
		insert into assets (id, public_id, product_id, parent_asset_id)
		values (?, ?, ?, ?)
	`, thirdSystemAssetID, thirdSystemAssetPublicID, systemProductID, secondSystemAssetID)
	assertInventorySQLFails(t, store, "created a multi-hop asset cycle", `
		update assets set parent_asset_id = ? where id = ?
	`, thirdSystemAssetID, secondSystemAssetID)

	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "backed a virtual machine with an asset", `
		insert into machines (
			id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind
		) values (?, ?, ?, ?, ?, 'Invalid VM asset', 'invalid-vm-asset', 'virtual_machine')
	`, invalidID, invalidPublicID, systemAssetID, providerID, areaID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "backed a machine with a non-system asset", `
		insert into machines (
			id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind
		) values (?, ?, ?, ?, ?, 'Invalid machine asset', 'invalid-machine-asset', 'bare_metal')
	`, invalidID, invalidPublicID, memoryAssetID, providerID, areaID)

	bareMachineID, bareMachinePublicID := testIdentifiers(t)
	execInventorySQL(t, store, `
		insert into machines (
			id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind,
			cpu_allocation
		) values (?, ?, ?, ?, ?, 'Badger', 'badger', 'bare_metal', 'dedicated')
	`, bareMachineID, bareMachinePublicID, systemAssetID, providerID, areaID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "reused a machine backing asset", `
		insert into machines (
			id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind
		) values (?, ?, ?, ?, ?, 'Duplicate asset', 'duplicate-asset', 'bare_metal')
	`, invalidID, invalidPublicID, systemAssetID, providerID, areaID)
	assertInventorySQLFails(t, store, "changed the product kind of a backing asset", `
		update products set kind = 'processor' where id = ?
	`, systemProductID)

	childMachineID, childMachinePublicID := testIdentifiers(t)
	grandchildMachineID, grandchildMachinePublicID := testIdentifiers(t)
	execInventorySQL(t, store, `
		insert into machines (
			id, public_id, parent_machine_id, machine_provider_id, area_id, name, slug, kind
		) values (?, ?, ?, ?, ?, 'Nested VM', 'nested-vm', 'virtual_machine')
	`, childMachineID, childMachinePublicID, machineID, providerID, areaID)
	execInventorySQL(t, store, `
		insert into machines (
			id, public_id, parent_machine_id, machine_provider_id, area_id, name, slug, kind
		) values (?, ?, ?, ?, ?, 'Nested VM 2', 'nested-vm-2', 'virtual_machine')
	`, grandchildMachineID, grandchildMachinePublicID, childMachineID, providerID, areaID)
	assertInventorySQLFails(t, store, "created a multi-hop machine cycle", `
		update machines set parent_machine_id = ? where id = ?
	`, grandchildMachineID, machineID)

	otherProviderID, otherProviderPublicID := testIdentifiers(t)
	otherAreaID, otherAreaPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `
		insert into machine_providers (id, public_id, name, slug)
		values (?, ?, 'Homelab', 'homelab')
	`, otherProviderID, otherProviderPublicID)
	execInventorySQL(t, store, `
		insert into areas (id, public_id, machine_provider_id, name, slug)
		values (?, ?, ?, 'Primary Home', 'primary-home')
	`, otherAreaID, otherAreaPublicID, otherProviderID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "used a local parent in another provider and area", `
		insert into machines (
			id, public_id, parent_machine_id, machine_provider_id, area_id, name, slug, kind
		) values (?, ?, ?, ?, ?, 'Remote child', 'remote-child', 'virtual_machine')
	`, invalidID, invalidPublicID, machineID, otherProviderID, otherAreaID)
	assertInventorySQLFails(t, store, "moved a parent away from its local children", `
		update machines set machine_provider_id = ?, area_id = ? where id = ?
	`, otherProviderID, otherAreaID, machineID)

	assertInventorySQLFails(t, store, "used a lowercase cost currency", `
		update machines set cost_currency = 'usd' where id = ?
	`, machineID)
	assertInventorySQLFails(t, store, "removed currency but retained cost", `
		update machines set cost_currency = null where id = ?
	`, machineID)
	if _, err := store.db.Exec(`
		update machines set parent_machine_id = id where id = ?
	`, machineID); err == nil {
		t.Fatal("made a machine its own parent")
	}
	if _, err := store.db.Exec(`
		update machines set cpu_allocation = 'burstable' where id = ?
	`, machineID); err == nil {
		t.Fatal("set an unsupported CPU allocation kind")
	}
	if _, err := store.db.Exec(`
		update machines set storage_interface_kind = 'unsupported' where id = ?
	`, machineID); err == nil {
		t.Fatal("set an unsupported storage interface kind")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into instances (
			id, public_id, service_id, machine_id, name, slug, port
		) values (?, ?, ?, ?, 'Invalid port', 'invalid-port', 70000)
	`, invalidID, invalidPublicID, serviceID, machineID); err == nil {
		t.Fatal("inserted an instance with an invalid port")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into products (id, public_id, manufacturer_id, kind, name)
		values (?, ?, ?, 'invalid', 'Invalid')
	`, invalidID, invalidPublicID, hpID); err == nil {
		t.Fatal("inserted an invalid product kind")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into assets (id, public_id, product_id)
		values (?, ?, ?)
	`, invalidID, invalidPublicID, invalidID); err == nil {
		t.Fatal("inserted an asset with a missing product")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into areas (id, public_id, name, slug)
		values (?, ?, 'Missing provider', 'missing-provider')
	`, invalidID, invalidPublicID); err == nil {
		t.Fatal("inserted an area without a machine provider")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into machines (id, public_id, area_id, name, slug, kind)
		values (?, ?, ?, 'Missing provider', 'missing-provider', 'bare_metal')
	`, invalidID, invalidPublicID, areaID); err == nil {
		t.Fatal("inserted a machine without a machine provider")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into machines (id, public_id, machine_provider_id, name, slug, kind)
		values (?, ?, ?, 'Missing area', 'missing-area', 'bare_metal')
	`, invalidID, invalidPublicID, providerID); err == nil {
		t.Fatal("inserted a machine without an area")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into machines (id, public_id, machine_provider_id, area_id, name, slug, kind)
		values (?, ?, ?, ?, 'Unused container kind', 'unused-container', 'container')
	`, invalidID, invalidPublicID, providerID, areaID); err == nil {
		t.Fatal("inserted an unsupported machine kind")
	}

	invalidID, invalidPublicID = testIdentifiers(t)
	if _, err := store.db.Exec(`
		insert into networks (id, public_id, name, slug, kind)
		values (?, ?, 'Unused container kind', 'unused-container-kind', 'container')
	`, invalidID, invalidPublicID); err == nil {
		t.Fatal("inserted an unsupported network kind")
	}

	if _, err := store.db.Exec(`
		insert into machine_providers (public_id, name)
		values ('invalid-id!!', 'Invalid')
	`); err == nil {
		t.Fatal("inserted invalid internal and public IDs")
	}
}

func TestEquipmentInventorySchema(t *testing.T) {
	store, err := NewSQLiteDB(filepath.Join(t.TempDir(), "equipment.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	providerID, providerPublicID := testIdentifiers(t)
	areaID, areaPublicID := testIdentifiers(t)
	manufacturerID, manufacturerPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into machine_providers (id, public_id, name, slug) values (?, ?, 'Local', 'local')`, providerID, providerPublicID)
	execInventorySQL(t, store, `insert into areas (id, public_id, machine_provider_id, name, slug) values (?, ?, ?, 'Home', 'home')`, areaID, areaPublicID, providerID)
	execInventorySQL(t, store, `insert into manufacturers (id, public_id, name, slug) values (?, ?, 'Equipment Co', 'equipment-co')`, manufacturerID, manufacturerPublicID)

	rackProductID, rackProductPublicID := testIdentifiers(t)
	switchProductID, switchProductPublicID := testIdentifiers(t)
	systemProductID, systemProductPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into products (id, public_id, manufacturer_id, kind, name) values (?, ?, ?, 'rack', 'Four Unit Rack')`, rackProductID, rackProductPublicID, manufacturerID)
	execInventorySQL(t, store, `insert into rack_specs (product_id, rack_units, mounting_standard) values (?, 4, '10-inch')`, rackProductID)
	execInventorySQL(t, store, `insert into products (id, public_id, manufacturer_id, kind, name) values (?, ?, ?, 'switch', 'Five Port Switch')`, switchProductID, switchProductPublicID, manufacturerID)
	execInventorySQL(t, store, `insert into product_port_profiles (product_id, position, port_count, connector, speed_mbps) values (?, 0, 5, 'RJ45', 1000)`, switchProductID)
	execInventorySQL(t, store, `insert into product_links (product_id, position, kind, url) values (?, 0, 'manufacturer', 'https://example.com/switch')`, switchProductID)
	execInventorySQL(t, store, `insert into products (id, public_id, manufacturer_id, kind, name) values (?, ?, ?, 'system', 'Server')`, systemProductID, systemProductPublicID, manufacturerID)

	rackAssetID, rackAssetPublicID := testIdentifiers(t)
	switchAssetID, switchAssetPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into assets (id, public_id, product_id, area_id, name) values (?, ?, ?, ?, 'Rack')`, rackAssetID, rackAssetPublicID, rackProductID, areaID)
	execInventorySQL(t, store, `insert into assets (id, public_id, product_id, parent_asset_id, name) values (?, ?, ?, ?, 'Switch')`, switchAssetID, switchAssetPublicID, switchProductID, rackAssetID)
	execInventorySQL(t, store, `insert into asset_links (asset_id, position, kind, url) values (?, 0, 'receipt', 'https://example.com/receipt')`, switchAssetID)

	switchAsset, err := store.queries.GetAssetByPublicID(context.Background(), switchAssetPublicID)
	if err != nil {
		t.Fatal(err)
	}
	if !switchAsset.EffectiveAreaPublicID.Valid || switchAsset.EffectiveAreaPublicID.String != areaPublicID {
		t.Fatalf("switch effective area = %v; want %q", switchAsset.EffectiveAreaPublicID, areaPublicID)
	}

	networkID, networkPublicID := testIdentifiers(t)
	addressID, addressPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into networks (id, public_id, area_id, name, slug, kind, cidr) values (?, ?, ?, 'Home LAN', 'home-lan', 'lan', '192.168.1.0/24')`, networkID, networkPublicID, areaID)
	execInventorySQL(t, store, `insert into addresses (id, public_id, network_id, asset_id, address) values (?, ?, ?, ?, '192.168.1.2')`, addressID, addressPublicID, networkID, switchAssetID)

	invalidID, invalidPublicID := testIdentifiers(t)
	assertInventorySQLFails(t, store, "assigned an address to a rack", `insert into addresses (id, public_id, network_id, asset_id, address) values (?, ?, ?, ?, '192.168.1.3')`, invalidID, invalidPublicID, networkID, rackAssetID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "assigned an address to multiple owners", `insert into addresses (id, public_id, network_id, area_id, asset_id, address) values (?, ?, ?, ?, ?, '192.168.1.4')`, invalidID, invalidPublicID, networkID, areaID, switchAssetID)
	assertInventorySQLFails(t, store, "changed an addressed network product kind", `update products set kind = 'system' where id = ?`, switchProductID)
	assertInventorySQLFails(t, store, "changed a containing rack product kind", `update products set kind = 'system' where id = ?`, rackProductID)
	assertInventorySQLFails(t, store, "changed an addressed asset to a system product", `update assets set product_id = ? where id = ?`, systemProductID, switchAssetID)
	assertInventorySQLFails(t, store, "changed a containing asset to a switch product", `update assets set product_id = ? where id = ?`, switchProductID, rackAssetID)
	assertInventorySQLFails(t, store, "moved an address to a rack asset", `update addresses set asset_id = ? where id = ?`, rackAssetID, addressID)
	assertInventorySQLFails(t, store, "backed a machine with a rack", `insert into machines (id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind) values (?, ?, ?, ?, ?, 'Rack Machine', 'rack-machine', 'bare_metal')`, invalidID, invalidPublicID, rackAssetID, providerID, areaID)

	otherAreaID, otherAreaPublicID := testIdentifiers(t)
	systemChildID, systemChildPublicID := testIdentifiers(t)
	backedMachineID, backedMachinePublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into areas (id, public_id, machine_provider_id, name, slug) values (?, ?, ?, 'Other Room', 'other-room')`, otherAreaID, otherAreaPublicID, providerID)
	execInventorySQL(t, store, `insert into assets (id, public_id, product_id, parent_asset_id, name) values (?, ?, ?, ?, 'Rack server')`, systemChildID, systemChildPublicID, systemProductID, rackAssetID)
	execInventorySQL(t, store, `insert into machines (id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind) values (?, ?, ?, ?, ?, 'Rack Server', 'rack-server', 'bare_metal')`, backedMachineID, backedMachinePublicID, systemChildID, providerID, areaID)
	assertInventorySQLFails(t, store, "unplaced a Machine backing Asset", `update assets set parent_asset_id = null where id = ?`, systemChildID)
	assertInventorySQLFails(t, store, "unplaced a rack containing a Machine backing Asset", `update assets set area_id = null where id = ?`, rackAssetID)
	assertInventorySQLFails(t, store, "moved a rack away from a descendant Machine", `update assets set area_id = ? where id = ?`, otherAreaID, rackAssetID)
	assertInventorySQLFails(t, store, "moved a Machine away from its backing Asset", `update machines set area_id = ? where id = ?`, otherAreaID, backedMachineID)
	unplacedSystemID, unplacedSystemPublicID := testIdentifiers(t)
	invalidMachineID, invalidMachinePublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into assets (id, public_id, product_id) values (?, ?, ?)`, unplacedSystemID, unplacedSystemPublicID, systemProductID)
	assertInventorySQLFails(t, store, "backed a Machine with an unplaced Asset", `insert into machines (id, public_id, asset_id, machine_provider_id, area_id, name, slug, kind) values (?, ?, ?, ?, ?, 'Unplaced', 'unplaced', 'bare_metal')`, invalidMachineID, invalidMachinePublicID, unplacedSystemID, providerID, areaID)

	assertInventorySQLFails(t, store, "attached ports to a system product", `insert into product_port_profiles (product_id, position, port_count, connector, speed_mbps) values (?, 0, 1, 'RJ45', 1000)`, systemProductID)
	assertInventorySQLFails(t, store, "created an invalid port profile", `insert into product_port_profiles (product_id, position, port_count, connector, speed_mbps) values (?, 1, 0, 'RJ45', 1000)`, switchProductID)
	assertInventorySQLFails(t, store, "created a duplicate product link URL", `insert into product_links (product_id, position, kind, url) values (?, 1, 'support', 'https://example.com/switch')`, switchProductID)

	adapterProductID, adapterProductPublicID := testIdentifiers(t)
	computerAssetID, computerAssetPublicID := testIdentifiers(t)
	adapterAssetID, adapterAssetPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into products (id, public_id, manufacturer_id, kind, name) values (?, ?, ?, 'network_adapter', 'Dual Port NIC')`, adapterProductID, adapterProductPublicID, manufacturerID)
	execInventorySQL(t, store, `insert into product_port_profiles (product_id, position, port_count, connector, speed_mbps) values (?, 0, 2, 'RJ45', 1000)`, adapterProductID)
	execInventorySQL(t, store, `insert into assets (id, public_id, product_id, area_id) values (?, ?, ?, ?)`, computerAssetID, computerAssetPublicID, systemProductID, areaID)
	execInventorySQL(t, store, `insert into assets (id, public_id, product_id, parent_asset_id, parent_slot) values (?, ?, ?, ?, 'PCIe x16')`, adapterAssetID, adapterAssetPublicID, adapterProductID, computerAssetID)
	assertInventorySQLFails(t, store, "stored a slot without a parent Asset", `update assets set parent_slot = 'PCIe x4' where id = ?`, rackAssetID)
	computerPurchaseID, computerPurchasePublicID := testIdentifiers(t)
	nicPurchaseID, nicPurchasePublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into purchases (id, public_id, primary_asset_id, total_price_cents, currency, purchased_on, source) values (?, ?, ?, 3500, 'USD', '2026-09-19', 'Facebook Marketplace')`, computerPurchaseID, computerPurchasePublicID, computerAssetID)
	execInventorySQL(t, store, `insert into purchases (id, public_id, primary_asset_id, total_price_cents, currency) values (?, ?, ?, 1500, 'USD')`, nicPurchaseID, nicPurchasePublicID, adapterAssetID)
	assertInventorySQLFails(t, store, "assigned an Asset to two Purchases", `insert into purchase_assets (purchase_id, asset_id) values (?, ?)`, computerPurchaseID, adapterAssetID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "stored an invalid purchase currency", `insert into purchases (id, public_id, primary_asset_id, total_price_cents, currency) values (?, ?, ?, 1000, 'usd')`, invalidID, invalidPublicID, rackAssetID)
	invalidID, invalidPublicID = testIdentifiers(t)
	assertInventorySQLFails(t, store, "stored an invalid purchase date", `insert into purchases (id, public_id, primary_asset_id, total_price_cents, currency, purchased_on) values (?, ?, ?, 1000, 'USD', '2026-13-40')`, invalidID, invalidPublicID, rackAssetID)
}

func TestMachineProviderLogoSchemaAndCascade(t *testing.T) {
	store, err := NewSQLiteDB(filepath.Join(t.TempDir(), "provider-logo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	providerID, providerPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `
		insert into machine_providers (id, public_id, name, slug)
		values (?, ?, 'Provider', 'provider')
	`, providerID, providerPublicID)
	execInventorySQL(t, store, `
		insert into machine_provider_logos (machine_provider_id, content_type, image_data)
		values (?, 'image/png', ?)
	`, providerID, []byte("\x89PNG\r\n\x1a\nLOGO"))

	assertInventorySQLFails(t, store, "accepted an unsupported provider logo type", `
		update machine_provider_logos set content_type = 'image/svg+xml'
		where machine_provider_id = ?
	`, providerID)
	assertInventorySQLFails(t, store, "accepted an oversized provider logo", `
		update machine_provider_logos set image_data = zeroblob(1048577)
		where machine_provider_id = ?
	`, providerID)

	execInventorySQL(t, store, "delete from machine_providers where id = ?", providerID)
	var logos int
	if err := store.db.QueryRow("select count(*) from machine_provider_logos").Scan(&logos); err != nil {
		t.Fatal(err)
	}
	if logos != 0 {
		t.Fatalf("provider logos after provider deletion = %d; want 0", logos)
	}
}

func TestMachineUserSchemaConstraintsAndCascade(t *testing.T) {
	store, err := NewSQLiteDB(filepath.Join(t.TempDir(), "machine-users.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	providerID, providerPublicID := testIdentifiers(t)
	areaID, areaPublicID := testIdentifiers(t)
	machineID, machinePublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into machine_providers (id, public_id, name, slug) values (?, ?, 'Local', 'local')`, providerID, providerPublicID)
	execInventorySQL(t, store, `insert into areas (id, public_id, machine_provider_id, name, slug) values (?, ?, ?, 'Home', 'home')`, areaID, areaPublicID, providerID)
	execInventorySQL(t, store, `insert into machines (id, public_id, machine_provider_id, area_id, name, slug, kind) values (?, ?, ?, ?, 'Badger', 'badger', 'bare_metal')`, machineID, machinePublicID, providerID, areaID)

	userID, userPublicID := testIdentifiers(t)
	execInventorySQL(t, store, `insert into machine_users (id, public_id, machine_id, username, is_preferred) values (?, ?, ?, 'tyler', 1)`, userID, userPublicID, machineID)
	for name, username := range map[string]string{
		"empty": "", "leading hyphen": "-root", "space": "bad user", "non-ASCII": "týler", "too long": strings.Repeat("a", 65),
	} {
		invalidID, invalidPublicID := testIdentifiers(t)
		assertInventorySQLFails(t, store, "accepted "+name+" username", `insert into machine_users (id, public_id, machine_id, username) values (?, ?, ?, ?)`, invalidID, invalidPublicID, machineID, username)
	}
	duplicateID, duplicatePublicID := testIdentifiers(t)
	assertInventorySQLFails(t, store, "accepted duplicate username", `insert into machine_users (id, public_id, machine_id, username) values (?, ?, ?, 'tyler')`, duplicateID, duplicatePublicID, machineID)
	preferredID, preferredPublicID := testIdentifiers(t)
	assertInventorySQLFails(t, store, "accepted a second preferred user", `insert into machine_users (id, public_id, machine_id, username, is_preferred) values (?, ?, ?, 'root', 1)`, preferredID, preferredPublicID, machineID)

	execInventorySQL(t, store, `delete from machines where id = ?`, machineID)
	var users int
	if err := store.db.QueryRow(`select count(*) from machine_users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 0 {
		t.Fatalf("machine users after machine deletion = %d; want 0", users)
	}
}

func assertInventorySQLFails(t *testing.T, store *SQLiteDB, message, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err == nil {
		t.Fatal(message)
	}
}

func execInventorySQL(t *testing.T, store *SQLiteDB, query string, args ...any) {
	t.Helper()
	if _, err := store.db.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func testIdentifiers(t *testing.T) (string, string) {
	t.Helper()
	internalID, err := NewInternalID()
	if err != nil {
		t.Fatal(err)
	}
	publicID, err := NewPublicID()
	if err != nil {
		t.Fatal(err)
	}
	return string(internalID), string(publicID)
}
