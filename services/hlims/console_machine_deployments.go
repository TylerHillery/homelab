package hlims

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

type machineDeploymentIndex struct {
	byMachine        map[string][]machineDeploymentGroup
	systemdByMachine map[string][]machineSystemdGroup
	byInstance       map[string]machineDeploymentMembership
}

type machineDeploymentMembership struct {
	deploymentID string
	member       string
	role         string
	systemdUnit  string
	systemdScope string
	systemdUser  string
	uses         []machineInstanceUsage
}

type machineDeploymentGroup struct {
	PublicID  string
	Name      string
	Project   string
	Directory string
	Files     []string
	Notes     string
	Services  []machineServiceView
}

type machineSystemdGroup struct {
	Scope    string
	User     string
	Services []machineServiceView
}

type machineServiceView struct {
	api.TopologyService
	Instances []machineInstanceView
}

type machineInstanceView struct {
	api.TopologyInstance
	Member      string
	Role        string
	SystemdUnit string
	Uses        []machineInstanceUsage
}

type machineInstanceUsage struct {
	Kind           string
	ServiceName    string
	InstanceName   string
	DeploymentName string
	MachineName    string
}

func loadMachineDeploymentIndex(ctx context.Context, queries *database.Queries) (machineDeploymentIndex, error) {
	index := machineDeploymentIndex{byMachine: make(map[string][]machineDeploymentGroup), systemdByMachine: make(map[string][]machineSystemdGroup), byInstance: make(map[string]machineDeploymentMembership)}
	groups, err := queries.ListDeployments(ctx)
	if err != nil {
		return index, fmt.Errorf("list deployments: %w", err)
	}
	groupNames := make(map[string]string, len(groups))
	for _, row := range groups {
		var files []string
		if err := json.Unmarshal([]byte(row.ComposeFiles), &files); err != nil {
			return index, fmt.Errorf("deployment %s Compose files: %w", row.PublicID, err)
		}
		index.byMachine[row.MachinePublicID] = append(index.byMachine[row.MachinePublicID], machineDeploymentGroup{
			PublicID: row.PublicID, Name: row.Name,
			Directory: row.WorkingDirectory, Project: row.ComposeProject,
			Files: files, Notes: row.Notes.String,
		})
		groupNames[row.PublicID] = row.Name
	}
	instances, err := queries.ListInstances(ctx)
	if err != nil {
		return index, fmt.Errorf("list deployment members: %w", err)
	}
	instanceDetails := make(map[string]database.ListInstancesRow, len(instances))
	for _, instance := range instances {
		index.byInstance[instance.PublicID] = machineDeploymentMembership{deploymentID: instance.DeploymentPublicID.String, member: instance.DeploymentMember.String, role: instance.DeploymentRole, systemdUnit: instance.SystemdUnit.String, systemdScope: instance.SystemdScope.String, systemdUser: instance.SystemdUser.String}
		if instance.SystemdUnit.Valid {
			groups := index.systemdByMachine[instance.MachinePublicID.String]
			found := false
			for _, group := range groups {
				if group.Scope == instance.SystemdScope.String && group.User == instance.SystemdUser.String {
					found = true
					break
				}
			}
			if !found {
				index.systemdByMachine[instance.MachinePublicID.String] = append(groups, machineSystemdGroup{Scope: instance.SystemdScope.String, User: instance.SystemdUser.String})
			}
		}
		instanceDetails[instance.PublicID] = instance
	}
	for machineID := range index.systemdByMachine {
		sort.Slice(index.systemdByMachine[machineID], func(i, j int) bool {
			groups := index.systemdByMachine[machineID]
			if groups[i].Scope != groups[j].Scope {
				return groups[i].Scope < groups[j].Scope
			}
			return groups[i].User < groups[j].User
		})
	}
	services, err := queries.ListServices(ctx)
	if err != nil {
		return index, fmt.Errorf("list dependency services: %w", err)
	}
	serviceNames := make(map[string]string, len(services))
	for _, service := range services {
		serviceNames[service.PublicID] = service.Name
	}
	machines, err := queries.ListMachineDetails(ctx)
	if err != nil {
		return index, fmt.Errorf("list dependency machines: %w", err)
	}
	machineNames := make(map[string]string, len(machines))
	for _, machine := range machines {
		machineNames[machine.PublicID] = machine.Name
	}
	dependencies, err := queries.ListInstanceDependencies(ctx)
	if err != nil {
		return index, fmt.Errorf("list instance usage: %w", err)
	}
	for _, dependency := range dependencies {
		provider, ok := instanceDetails[dependency.ProviderPublicID]
		if !ok {
			return index, fmt.Errorf("usage references missing Instance %s", dependency.ProviderPublicID)
		}
		membership := index.byInstance[dependency.ConsumerPublicID]
		membership.uses = append(membership.uses, machineInstanceUsage{
			Kind:        dependency.Kind,
			ServiceName: serviceNames[provider.ServicePublicID], InstanceName: provider.Name,
			DeploymentName: groupNames[provider.DeploymentPublicID.String], MachineName: machineNames[provider.MachinePublicID.String],
		})
		index.byInstance[dependency.ConsumerPublicID] = membership
	}
	return index, nil
}

func machineServicesAndDeployments(machine api.TopologyMachine, index machineDeploymentIndex) ([]machineServiceView, []machineDeploymentGroup, []machineSystemdGroup) {
	groups := append([]machineDeploymentGroup(nil), index.byMachine[machine.PublicId]...)
	systemdGroups := append([]machineSystemdGroup(nil), index.systemdByMachine[machine.PublicId]...)
	independent := make([]machineServiceView, 0, len(machine.Services))
	groupByID := make(map[string]int, len(groups))
	for i, group := range groups {
		groupByID[group.PublicID] = i
	}
	for _, service := range machine.Services {
		standalone := machineServiceView{TopologyService: service}
		for _, instance := range service.Instances {
			membership := index.byInstance[instance.PublicId]
			view := machineInstanceView{TopologyInstance: instance, Member: membership.member, Role: membership.role, SystemdUnit: membership.systemdUnit, Uses: membership.uses}
			if membership.systemdUnit != "" {
				for i := range systemdGroups {
					if systemdGroups[i].Scope == membership.systemdScope && systemdGroups[i].User == membership.systemdUser {
						appendMachineService(&systemdGroups[i].Services, service, view)
						break
					}
				}
				continue
			}
			if membership.deploymentID == "" {
				standalone.Instances = append(standalone.Instances, view)
				continue
			}
			groupIndex, ok := groupByID[membership.deploymentID]
			if !ok {
				continue
			}
			appendMachineService(&groups[groupIndex].Services, service, view)
		}
		if len(standalone.Instances) > 0 {
			independent = append(independent, standalone)
		}
	}
	return independent, groups, systemdGroups
}

func appendMachineService(services *[]machineServiceView, service api.TopologyService, instance machineInstanceView) {
	for i := range *services {
		if (*services)[i].PublicId == service.PublicId {
			(*services)[i].Instances = append((*services)[i].Instances, instance)
			return
		}
	}
	*services = append(*services, machineServiceView{TopologyService: service, Instances: []machineInstanceView{instance}})
}
