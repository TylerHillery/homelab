package hlims

import (
	"fmt"
	"net/http"
	"sort"
)

type serviceCatalogPage struct {
	View         string
	Services     []serviceCatalogCard
	ManagedCount int
}

type serviceCatalogCard struct {
	PublicID    string
	Name        string
	Description string
	HasLogo     bool
	Instances   []serviceCatalogInstance
	Groups      []serviceCatalogGroup
	PrimaryPath string
	PrimaryName string
}

type serviceCatalogGroup struct {
	PublicID  string
	Name      string
	Machine   string
	MachineID string
	Instances []serviceCatalogInstance
}

type serviceCatalogInstance struct {
	Name            string
	Kind            string
	Role            string
	Member          string
	SystemdUnit     string
	SystemdScope    string
	SystemdUser     string
	Provider        string
	MachineName     string
	MachinePublicID string
	ResolverPath    string
	Links           []serviceCatalogLink
}

type serviceCatalogLink struct {
	Name      string
	URL       string
	Network   string
	Preferred bool
}

func (s consoleServer) services(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services, err := s.api.queries.ListServices(ctx)
	if err != nil {
		s.renderError(w, fmt.Errorf("list services: %w", err))
		return
	}
	instances, err := s.api.queries.ListInstances(ctx)
	if err != nil {
		s.renderError(w, fmt.Errorf("list instances: %w", err))
		return
	}
	deployments, err := s.api.queries.ListDeployments(ctx)
	if err != nil {
		s.renderError(w, fmt.Errorf("list deployments: %w", err))
		return
	}
	machines, err := s.api.queries.ListMachineDetails(ctx)
	if err != nil {
		s.renderError(w, fmt.Errorf("list machines: %w", err))
		return
	}
	endpoints, err := s.api.queries.ListInstanceEndpoints(ctx)
	if err != nil {
		s.renderError(w, fmt.Errorf("list endpoints: %w", err))
		return
	}
	addressEndpoints, err := s.api.queries.ListTopologyInstanceEndpoints(ctx)
	if err != nil {
		s.renderError(w, fmt.Errorf("list address endpoints: %w", err))
		return
	}
	links := make(map[string]serviceCatalogLink, len(addressEndpoints))
	for _, row := range addressEndpoints {
		url, err := endpointURL(endpoint{address: row.Address, dnsName: row.DnsName, hostType: row.HostType, scheme: row.Scheme.String, port: row.Port.Int64, basePath: row.BasePath}, "", nil)
		if err != nil {
			s.renderError(w, fmt.Errorf("endpoint %s: %w", row.PublicID, err))
			return
		}
		links[row.PublicID] = serviceCatalogLink{Name: row.Name, URL: url, Network: row.NetworkKind, Preferred: row.IsPreferred == 1}
	}
	byInstance := make(map[string][]serviceCatalogLink)
	for _, row := range endpoints {
		link, ok := links[row.PublicID]
		if row.DirectUrl.Valid {
			link = serviceCatalogLink{Name: row.Name, URL: row.DirectUrl.String, Network: "managed", Preferred: row.IsPreferred == 1}
		} else if !ok {
			s.renderError(w, fmt.Errorf("endpoint %s has no destination", row.PublicID))
			return
		}
		byInstance[row.InstancePublicID] = append(byInstance[row.InstancePublicID], link)
	}
	machineNames := make(map[string]string, len(machines))
	machineSlugs := make(map[string]string, len(machines))
	for _, machine := range machines {
		machineNames[machine.PublicID] = machine.Name
		machineSlugs[machine.PublicID] = machine.Slug
	}
	page := serviceCatalogPage{View: "services", Services: make([]serviceCatalogCard, 0, len(services))}
	serviceIndex := make(map[string]int, len(services))
	serviceSlugs := make(map[string]string, len(services))
	deploymentNames := make(map[string]serviceCatalogGroup, len(deployments))
	for _, deployment := range deployments {
		deploymentNames[deployment.PublicID] = serviceCatalogGroup{
			PublicID: deployment.PublicID, Name: deployment.Name,
			Machine: deployment.MachineName, MachineID: deployment.MachinePublicID,
		}
	}
	instanceNames := make(map[string]int, len(instances))
	for _, instance := range instances {
		instanceNames[instance.ServicePublicID+"/"+instance.Slug]++
	}
	for _, service := range services {
		serviceIndex[service.PublicID] = len(page.Services)
		serviceSlugs[service.PublicID] = service.Slug
		page.Services = append(page.Services, serviceCatalogCard{PublicID: service.PublicID, Name: service.Name, Description: service.Description.String, HasLogo: service.HasLogo, Instances: []serviceCatalogInstance{}})
	}
	for _, row := range instances {
		index, ok := serviceIndex[row.ServicePublicID]
		if !ok {
			s.renderError(w, fmt.Errorf("instance %s references missing service", row.PublicID))
			return
		}
		service := &page.Services[index]
		item := serviceCatalogInstance{Name: row.Name, Kind: row.HostingKind, Role: row.DeploymentRole, Member: row.DeploymentMember.String, SystemdUnit: row.SystemdUnit.String, SystemdScope: row.SystemdScope.String, SystemdUser: row.SystemdUser.String, Provider: row.ManagedProvider.String, MachinePublicID: row.MachinePublicID.String, MachineName: machineNames[row.MachinePublicID.String], Links: byInstance[row.PublicID]}
		if row.HostingKind == "managed" {
			page.ManagedCount++
			item.ResolverPath = "/" + serviceSlugs[row.ServicePublicID] + "/" + row.Slug
			if instanceNames[row.ServicePublicID+"/"+row.Slug] > 1 {
				item.ResolverPath += "?host=managed"
			}
		} else if row.MachinePublicID.Valid {
			item.ResolverPath = "/" + serviceSlugs[row.ServicePublicID] + "/" + row.Slug + "?host=" + machineSlugs[row.MachinePublicID.String]
		}
		service.Instances = append(service.Instances, item)
		group := deploymentNames[row.DeploymentPublicID.String]
		if !row.DeploymentPublicID.Valid {
			group = serviceCatalogGroup{Name: "Independent instances"}
		}
		groupIndex := -1
		for i := range service.Groups {
			if service.Groups[i].PublicID == group.PublicID {
				groupIndex = i
				break
			}
		}
		if groupIndex == -1 {
			service.Groups = append(service.Groups, group)
			groupIndex = len(service.Groups) - 1
		}
		service.Groups[groupIndex].Instances = append(service.Groups[groupIndex].Instances, item)
	}
	for index := range page.Services {
		service := &page.Services[index]
		sort.Slice(service.Groups, func(i, j int) bool {
			if service.Groups[i].PublicID == "" || service.Groups[j].PublicID == "" {
				return service.Groups[i].PublicID != ""
			}
			return service.Groups[i].Name < service.Groups[j].Name
		})
		selectedPreferred := false
		for _, instance := range service.Instances {
			for _, link := range instance.Links {
				if service.PrimaryPath == "" || link.Preferred {
					service.PrimaryPath = instance.ResolverPath
					service.PrimaryName = instance.Name
				}
				if link.Preferred {
					selectedPreferred = true
					break
				}
			}
			if selectedPreferred {
				break
			}
		}
	}
	s.render(w, "page", page)
}
