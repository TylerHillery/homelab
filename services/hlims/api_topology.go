package hlims

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

func (s apiServer) GetTopology(w http.ResponseWriter, r *http.Request) {
	topology, err := s.topology(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to build topology")
		return
	}
	writeJSON(w, http.StatusOK, topology)
}

func (s apiServer) topology(ctx context.Context) (api.Topology, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return api.Topology{}, fmt.Errorf("begin topology snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	queries := s.queries.WithTx(tx)
	providers, err := queries.ListMachineProviders(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list providers: %w", err)
	}
	areas, err := queries.ListAreas(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list areas: %w", err)
	}
	machines, err := queries.ListMachineDetails(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list machines: %w", err)
	}
	users, err := queries.ListMachineUsers(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list machine users: %w", err)
	}
	addresses, err := queries.ListTopologyMachineAddresses(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list machine addresses: %w", err)
	}
	services, err := queries.ListServices(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list services: %w", err)
	}
	instances, err := queries.ListInstances(ctx)
	if err != nil {
		return api.Topology{}, fmt.Errorf("list instances: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return api.Topology{}, fmt.Errorf("commit topology snapshot: %w", err)
	}

	return buildTopology(providers, areas, machines, users, addresses, services, instances)
}

type topologyProviderBuilder struct {
	value api.TopologyProvider
	areas []*topologyAreaBuilder
}

type topologyAreaBuilder struct {
	value      api.TopologyArea
	providerID string
	machines   []*topologyMachineBuilder
}

type topologyMachineBuilder struct {
	value      api.TopologyMachine
	providerID string
	areaID     string
	parentID   string
	children   []*topologyMachineBuilder
	services   map[string]*topologyServiceBuilder
	visitState uint8
}

type topologyServiceBuilder struct {
	value api.TopologyService
}

func buildTopology(
	providerRows []database.ListMachineProvidersRow,
	areaRows []database.ListAreasRow,
	machineRows []database.ListMachineDetailsRow,
	userRows []database.ListMachineUsersRow,
	addressRows []database.ListTopologyMachineAddressesRow,
	serviceRows []database.ListServicesRow,
	instanceRows []database.ListInstancesRow,
) (api.Topology, error) {
	providers := make(map[string]*topologyProviderBuilder, len(providerRows))
	for _, row := range providerRows {
		if _, exists := providers[row.PublicID]; exists {
			return api.Topology{}, fmt.Errorf("duplicate provider %q", row.PublicID)
		}
		providers[row.PublicID] = &topologyProviderBuilder{value: api.TopologyProvider{
			PublicId: row.PublicID,
			Name:     row.Name,
			Slug:     row.Slug,
			HasLogo:  row.HasLogo,
			Areas:    []api.TopologyArea{},
		}}
	}

	areas := make(map[string]*topologyAreaBuilder, len(areaRows))
	for _, row := range areaRows {
		provider := providers[row.MachineProviderPublicID]
		if provider == nil {
			return api.Topology{}, fmt.Errorf("area %q references missing provider %q", row.PublicID, row.MachineProviderPublicID)
		}
		if _, exists := areas[row.PublicID]; exists {
			return api.Topology{}, fmt.Errorf("duplicate area %q", row.PublicID)
		}
		area := &topologyAreaBuilder{
			value: api.TopologyArea{
				PublicId: row.PublicID,
				Name:     row.Name,
				Slug:     row.Slug,
				Machines: []api.TopologyMachine{},
			},
			providerID: row.MachineProviderPublicID,
		}
		areas[row.PublicID] = area
		provider.areas = append(provider.areas, area)
	}

	machines := make(map[string]*topologyMachineBuilder, len(machineRows))
	for _, row := range machineRows {
		area := areas[row.AreaPublicID]
		if area == nil {
			return api.Topology{}, fmt.Errorf("machine %q references missing area %q", row.PublicID, row.AreaPublicID)
		}
		if providers[row.MachineProviderPublicID] == nil {
			return api.Topology{}, fmt.Errorf("machine %q references missing provider %q", row.PublicID, row.MachineProviderPublicID)
		}
		if area.providerID != row.MachineProviderPublicID {
			return api.Topology{}, fmt.Errorf("machine %q provider does not own area %q", row.PublicID, row.AreaPublicID)
		}
		if _, exists := machines[row.PublicID]; exists {
			return api.Topology{}, fmt.Errorf("duplicate machine %q", row.PublicID)
		}
		parentID := ""
		if row.ParentMachinePublicID.Valid {
			parentID = row.ParentMachinePublicID.String
		}
		machines[row.PublicID] = &topologyMachineBuilder{
			value: api.TopologyMachine{
				PublicId:               row.PublicID,
				Name:                   row.Name,
				Slug:                   row.Slug,
				Kind:                   api.MachineKind(row.Kind),
				IsFavorite:             row.IsFavorite,
				Hostname:               stringPointer(row.Hostname),
				OperatingSystem:        stringPointer(row.OperatingSystem),
				OperatingSystemVersion: stringPointer(row.OperatingSystemVersion),
				Kernel:                 stringPointer(row.Kernel),
				Architecture:           stringPointer(row.Architecture),
				CpuCount:               intPointer(row.CpuCount),
				CpuAllocation:          cpuAllocationPointer(row.CpuAllocation),
				CpuVendor:              stringPointer(row.CpuVendor),
				MemoryBytes:            int64Pointer(row.MemoryBytes),
				StorageBytes:           int64Pointer(row.StorageBytes),
				StorageMediaKind:       storageMediaPointer(row.StorageMediaKind),
				StorageInterfaceKind:   productStorageInterfacePointer(row.StorageInterfaceKind),
				VirtualizationPlatform: stringPointer(row.VirtualizationPlatform),
				Users:                  []api.TopologyMachineUser{},
				Addresses:              []api.TopologyMachineAddress{},
				Children:               []api.TopologyMachine{},
				Services:               []api.TopologyService{},
			},
			providerID: row.MachineProviderPublicID,
			areaID:     row.AreaPublicID,
			parentID:   parentID,
			services:   make(map[string]*topologyServiceBuilder),
		}
	}

	userIDs := make(map[string]struct{}, len(userRows))
	preferredUsers := make(map[string]string)
	for _, row := range userRows {
		machine := machines[row.MachinePublicID]
		if machine == nil {
			return api.Topology{}, fmt.Errorf("machine user %q references missing machine %q", row.PublicID, row.MachinePublicID)
		}
		if _, exists := userIDs[row.PublicID]; exists {
			return api.Topology{}, fmt.Errorf("duplicate machine user %q", row.PublicID)
		}
		if _, err := normalizeMachineUsername(row.Username); err != nil {
			return api.Topology{}, fmt.Errorf("machine user %q has invalid username: %w", row.PublicID, err)
		}
		if row.IsPreferred != 0 && row.IsPreferred != 1 {
			return api.Topology{}, fmt.Errorf("machine user %q has invalid preferred value %d", row.PublicID, row.IsPreferred)
		}
		if row.IsPreferred == 1 {
			if previous := preferredUsers[row.MachinePublicID]; previous != "" {
				return api.Topology{}, fmt.Errorf("machine %q has multiple preferred users %q and %q", row.MachinePublicID, previous, row.PublicID)
			}
			preferredUsers[row.MachinePublicID] = row.PublicID
		}
		userIDs[row.PublicID] = struct{}{}
		machine.value.Users = append(machine.value.Users, api.TopologyMachineUser{PublicId: row.PublicID, Username: row.Username, IsPreferred: row.IsPreferred == 1, Notes: stringPointer(row.Notes)})
	}

	addressIDs := make(map[string]struct{}, len(addressRows))
	for _, row := range addressRows {
		machine := machines[row.MachinePublicID]
		if machine == nil {
			return api.Topology{}, fmt.Errorf("machine address %q references missing machine %q", row.PublicID, row.MachinePublicID)
		}
		if _, exists := addressIDs[row.PublicID]; exists {
			return api.Topology{}, fmt.Errorf("duplicate machine address %q", row.PublicID)
		}
		if !row.MachineID.Valid {
			return api.Topology{}, fmt.Errorf("machine address %q has no machine ID", row.PublicID)
		}
		kind := api.NetworkKind(row.NetworkKind)
		if !kind.Valid() {
			return api.Topology{}, fmt.Errorf("machine address %q has invalid network kind %q", row.PublicID, row.NetworkKind)
		}
		if _, err := normalizeIP(row.Address); err != nil {
			return api.Topology{}, fmt.Errorf("machine address %q is invalid: %w", row.PublicID, err)
		}
		if row.IsPrimary != 0 && row.IsPrimary != 1 {
			return api.Topology{}, fmt.Errorf("machine address %q has invalid primary value %d", row.PublicID, row.IsPrimary)
		}
		addressIDs[row.PublicID] = struct{}{}
		machine.value.Addresses = append(machine.value.Addresses, api.TopologyMachineAddress{
			PublicId: row.PublicID, NetworkPublicId: row.NetworkPublicID, NetworkKind: kind,
			Name: stringPointer(row.Name), Address: row.Address, DnsName: stringPointer(row.DnsName),
			InterfaceName: stringPointer(row.InterfaceName), IsPrimary: row.IsPrimary == 1,
		})
	}

	for _, machine := range machines {
		if err := validateMachineParent(machine, machines); err != nil {
			return api.Topology{}, err
		}
	}
	for _, machine := range machines {
		if machine.parentID == "" {
			areas[machine.areaID].machines = append(areas[machine.areaID].machines, machine)
			continue
		}
		parent := machines[machine.parentID]
		parent.children = append(parent.children, machine)
	}

	services := make(map[string]api.TopologyService, len(serviceRows))
	for _, row := range serviceRows {
		if _, exists := services[row.PublicID]; exists {
			return api.Topology{}, fmt.Errorf("duplicate service %q", row.PublicID)
		}
		services[row.PublicID] = api.TopologyService{
			PublicId:    row.PublicID,
			Name:        row.Name,
			Slug:        row.Slug,
			Description: stringPointer(row.Description),
			HasLogo:     row.HasLogo,
			Instances:   []api.TopologyInstance{},
		}
	}
	for _, row := range instanceRows {
		machine := machines[row.MachinePublicID]
		if machine == nil {
			return api.Topology{}, fmt.Errorf("instance %q references missing machine %q", row.PublicID, row.MachinePublicID)
		}
		service, exists := services[row.ServicePublicID]
		if !exists {
			return api.Topology{}, fmt.Errorf("instance %q references missing service %q", row.PublicID, row.ServicePublicID)
		}
		group := machine.services[row.ServicePublicID]
		if group == nil {
			group = &topologyServiceBuilder{value: service}
			machine.services[row.ServicePublicID] = group
		}
		group.value.Instances = append(group.value.Instances, api.TopologyInstance{
			PublicId:     row.PublicID,
			Name:         row.Name,
			Slug:         row.Slug,
			Port:         int(row.Port),
			ResolverPath: "/" + machine.value.Slug + "/" + service.Slug + "/" + row.Slug,
			AvailableVia: topologyAvailableVia(row.HasLanRoute, row.HasTailnetRoute),
		})
	}

	result := api.Topology{Providers: make([]api.TopologyProvider, 0, len(providers))}
	providerList := make([]*topologyProviderBuilder, 0, len(providers))
	for _, provider := range providers {
		providerList = append(providerList, provider)
	}
	sort.Slice(providerList, func(i, j int) bool {
		return topologyValueLess(providerList[i].value.Name, providerList[i].value.Slug, providerList[i].value.PublicId, providerList[j].value.Name, providerList[j].value.Slug, providerList[j].value.PublicId)
	})
	for _, provider := range providerList {
		sort.Slice(provider.areas, func(i, j int) bool {
			return topologyValueLess(provider.areas[i].value.Name, provider.areas[i].value.Slug, provider.areas[i].value.PublicId, provider.areas[j].value.Name, provider.areas[j].value.Slug, provider.areas[j].value.PublicId)
		})
		for _, area := range provider.areas {
			sortMachineBuilders(area.machines)
			for _, machine := range area.machines {
				area.value.Machines = append(area.value.Machines, materializeMachine(machine))
			}
			provider.value.Areas = append(provider.value.Areas, area.value)
		}
		result.Providers = append(result.Providers, provider.value)
	}
	return result, nil
}

func topologyAvailableVia(hasLAN, hasTailnet bool) []api.Via {
	routes := make([]api.Via, 0, 2)
	if hasLAN {
		routes = append(routes, api.ViaLan)
	}
	if hasTailnet {
		routes = append(routes, api.ViaTailnet)
	}
	return routes
}

func validateMachineParent(machine *topologyMachineBuilder, machines map[string]*topologyMachineBuilder) error {
	if machine.visitState == 2 {
		return nil
	}
	if machine.visitState == 1 {
		return fmt.Errorf("machine hierarchy contains a cycle at %q", machine.value.PublicId)
	}
	machine.visitState = 1
	if machine.parentID != "" {
		parent := machines[machine.parentID]
		if parent == nil {
			return fmt.Errorf("machine %q references missing parent %q", machine.value.PublicId, machine.parentID)
		}
		if parent.providerID != machine.providerID || parent.areaID != machine.areaID {
			return fmt.Errorf("machine %q parent is outside its provider and area", machine.value.PublicId)
		}
		if err := validateMachineParent(parent, machines); err != nil {
			return err
		}
	}
	machine.visitState = 2
	return nil
}

func sortMachineBuilders(machines []*topologyMachineBuilder) {
	sort.Slice(machines, func(i, j int) bool {
		return topologyValueLess(machines[i].value.Name, machines[i].value.Slug, machines[i].value.PublicId, machines[j].value.Name, machines[j].value.Slug, machines[j].value.PublicId)
	})
	for _, machine := range machines {
		sortMachineBuilders(machine.children)
	}
}

func materializeMachine(machine *topologyMachineBuilder) api.TopologyMachine {
	sort.Slice(machine.value.Users, func(i, j int) bool {
		left, right := machine.value.Users[i], machine.value.Users[j]
		if left.IsPreferred != right.IsPreferred {
			return left.IsPreferred
		}
		if left.Username != right.Username {
			return left.Username < right.Username
		}
		return left.PublicId < right.PublicId
	})
	sort.Slice(machine.value.Addresses, func(i, j int) bool {
		left, right := machine.value.Addresses[i], machine.value.Addresses[j]
		if left.IsPrimary != right.IsPrimary {
			return left.IsPrimary
		}
		if left.NetworkKind != right.NetworkKind {
			return left.NetworkKind < right.NetworkKind
		}
		if left.Address != right.Address {
			return left.Address < right.Address
		}
		return left.PublicId < right.PublicId
	})
	for _, child := range machine.children {
		machine.value.Children = append(machine.value.Children, materializeMachine(child))
	}
	serviceList := make([]*topologyServiceBuilder, 0, len(machine.services))
	for _, service := range machine.services {
		serviceList = append(serviceList, service)
	}
	sort.Slice(serviceList, func(i, j int) bool {
		return topologyValueLess(serviceList[i].value.Name, serviceList[i].value.Slug, serviceList[i].value.PublicId, serviceList[j].value.Name, serviceList[j].value.Slug, serviceList[j].value.PublicId)
	})
	for _, service := range serviceList {
		sort.Slice(service.value.Instances, func(i, j int) bool {
			left, right := service.value.Instances[i], service.value.Instances[j]
			return topologyValueLess(left.Name, left.Slug, left.PublicId, right.Name, right.Slug, right.PublicId)
		})
		machine.value.Services = append(machine.value.Services, service.value)
	}
	return machine.value
}

func topologyValueLess(leftName, leftSlug, leftID, rightName, rightSlug, rightID string) bool {
	leftFolded, rightFolded := strings.ToLower(leftName), strings.ToLower(rightName)
	if leftFolded != rightFolded {
		return leftFolded < rightFolded
	}
	if leftName != rightName {
		return leftName < rightName
	}
	if leftSlug != rightSlug {
		return leftSlug < rightSlug
	}
	return leftID < rightID
}
