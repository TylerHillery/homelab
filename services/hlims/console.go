package hlims

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/generated/api"
	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

//go:embed console/templates/*.html console/static/*
var consoleFiles embed.FS

var consoleTemplates = template.Must(template.New("console").Funcs(template.FuncMap{
	"byteSize":          consoleByteSize,
	"initials":          consoleInitials,
	"reachabilityBadge": consoleReachabilityBadge,
	"kindLabel":         consoleKindLabel,
	"machineViews":      consoleMachineViews,
	"favoriteMachineViews": func(machines []api.TopologyMachine) []consoleMachineView {
		result := consoleMachineViews(machines, false)
		for index := range result {
			result[index].ShowChildren = false
		}
		return result
	},
	"nestedMachineViews": func(machines []api.TopologyMachine, expandAll bool, parentName string) []consoleMachineView {
		result := consoleMachineViews(machines, expandAll)
		for index := range result {
			result[index].ParentName = parentName
		}
		return result
	},
	"osLabel": consoleOSLabel,
	"providerMachines": func(provider api.TopologyProvider) int {
		return consoleMachineCount([]api.TopologyProvider{provider})
	},
	"viaLabel": func(via api.Via) string {
		return strings.ToUpper(string(via))
	},
	"viaPath": func(path string, via api.Via) string {
		return path + "?via=" + string(via)
	},
}).ParseFS(consoleFiles, "console/templates/*.html"))

type consoleServer struct {
	api apiServer
}

type consolePage struct {
	Providers      []api.TopologyProvider
	Favorites      []api.TopologyMachine
	ActiveProvider string
	Expansion      string
}

type consoleMachineView struct {
	api.TopologyMachine
	ExpandAll    bool
	ParentName   string
	ShowChildren bool
	SSH          *consoleSSHControl
}

type consoleMachineChildren struct {
	Machines   []api.TopologyMachine
	ParentName string
}

type reachabilityBadge struct {
	Kind string
	ID   string
	Name string
	URL  string
}

type consoleSSHUser struct {
	Username    string
	IsPreferred bool
}

type consoleSSHTarget struct {
	Host  string
	Label string
}

type consoleSSHControl struct {
	Users   []consoleSSHUser
	Targets []consoleSSHTarget
}

func registerConsoleHandlers(mux *http.ServeMux, server apiServer) {
	console := consoleServer{api: server}
	staticFiles, err := fs.Sub(consoleFiles, "console/static")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /console/static/", http.StripPrefix("/console/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("GET /api-docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api-docs/", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("GET /api-docs/{$}", console.apiDocs)
	mux.HandleFunc("GET /api-docs/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "API documentation page not found", http.StatusNotFound)
	})
	mux.HandleFunc("GET /console", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/console/", http.StatusPermanentRedirect)
	})
	mux.HandleFunc("GET /console/{$}", console.index)
	mux.HandleFunc("GET /console/topology", console.topologyFragment)
	mux.HandleFunc("GET /console/providers/{publicID}", console.provider)
	mux.HandleFunc("GET /console/machines/{publicID}/children", console.children)
	mux.HandleFunc("PUT /console/machines/{publicID}/favorite", console.machineFavorite)
	mux.HandleFunc("GET /console/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "console page not found", http.StatusNotFound)
	})
}

func (s consoleServer) apiDocs(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "api-docs", nil)
}

func (s consoleServer) index(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "")
}

func (s consoleServer) provider(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Vary", "HX-Request")
	publicID := r.PathValue("publicID")
	if isHTMXRequest(r) {
		page, ok := s.pageData(w, r, publicID)
		if !ok {
			return
		}
		s.render(w, "workspace", page)
		return
	}
	s.renderPage(w, r, publicID)
}

func (s consoleServer) topologyFragment(w http.ResponseWriter, r *http.Request) {
	page, ok := s.pageData(w, r, r.URL.Query().Get("provider"))
	if !ok {
		return
	}
	switch r.URL.Query().Get("expansion") {
	case "all", "none":
		page.Expansion = r.URL.Query().Get("expansion")
	}
	s.render(w, "workspace", page)
}

func (s consoleServer) children(w http.ResponseWriter, r *http.Request) {
	topology, err := s.api.topology(r.Context())
	if err != nil {
		s.renderError(w, err)
		return
	}
	machine := findConsoleMachine(topology, r.PathValue("publicID"))
	if machine == nil {
		http.Error(w, "machine not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Vary", "HX-Request")
	s.render(w, "machine-children", consoleMachineChildren{Machines: machine.Children, ParentName: machine.Name})
}

func (s consoleServer) machineFavorite(w http.ResponseWriter, r *http.Request) {
	value, err := strconv.ParseBool(r.URL.Query().Get("value"))
	if err != nil {
		http.Error(w, "favorite value must be true or false", http.StatusBadRequest)
		return
	}
	rows, err := s.api.queries.SetMachineFavorite(r.Context(), database.SetMachineFavoriteParams{
		IsFavorite: value,
		PublicID:   r.PathValue("publicID"),
	})
	if err != nil {
		s.renderError(w, err)
		return
	}
	if rows == 0 {
		http.Error(w, "machine not found", http.StatusNotFound)
		return
	}
	page, ok := s.pageData(w, r, r.URL.Query().Get("provider"))
	if !ok {
		return
	}
	if r.URL.Query().Get("expansion") == "all" {
		page.Expansion = "all"
	}
	s.render(w, "workspace", page)
}

func (s consoleServer) renderPage(w http.ResponseWriter, r *http.Request, providerID string) {
	page, ok := s.pageData(w, r, providerID)
	if !ok {
		return
	}
	s.render(w, "page", page)
}

func (s consoleServer) pageData(w http.ResponseWriter, r *http.Request, providerID string) (consolePage, bool) {
	topology, err := s.api.topology(r.Context())
	if err != nil {
		s.renderError(w, err)
		return consolePage{}, false
	}
	page := newConsolePage(topology, providerID)
	if providerID != "" && len(page.Providers) == 0 {
		http.Error(w, "provider not found", http.StatusNotFound)
		return consolePage{}, false
	}
	return page, true
}

func newConsolePage(topology api.Topology, providerID string) consolePage {
	providers := topology.Providers
	if providerID != "" {
		providers = nil
		for _, provider := range topology.Providers {
			if provider.PublicId == providerID {
				providers = []api.TopologyProvider{provider}
				break
			}
		}
	}
	favorites := []api.TopologyMachine{}
	if providerID == "" {
		favorites = consoleFavoriteMachines(topology.Providers)
	}
	return consolePage{Providers: providers, Favorites: favorites, ActiveProvider: providerID}
}

func (s consoleServer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src http: https:; img-src 'self' data:; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := consoleTemplates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("rendering console template %q: %v", name, err)
	}
}

func (s consoleServer) renderError(w http.ResponseWriter, err error) {
	log.Printf("building console topology: %v", err)
	http.Error(w, "failed to load console", http.StatusInternalServerError)
}

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func consoleReachabilityBadge(kind, publicID string, suffix any, name, url string) reachabilityBadge {
	id := publicID
	if value := fmt.Sprint(suffix); value != "" {
		id += "-" + value
	}
	return reachabilityBadge{Kind: kind, ID: id, Name: name, URL: url}
}

func findConsoleMachine(topology api.Topology, publicID string) *api.TopologyMachine {
	for providerIndex := range topology.Providers {
		provider := &topology.Providers[providerIndex]
		for areaIndex := range provider.Areas {
			if machine := findConsoleMachineIn(&provider.Areas[areaIndex].Machines, publicID); machine != nil {
				return machine
			}
		}
	}
	return nil
}

func findConsoleMachineIn(machines *[]api.TopologyMachine, publicID string) *api.TopologyMachine {
	for index := range *machines {
		machine := &(*machines)[index]
		if machine.PublicId == publicID {
			return machine
		}
		if child := findConsoleMachineIn(&machine.Children, publicID); child != nil {
			return child
		}
	}
	return nil
}

func consoleMachineCount(providers []api.TopologyProvider) int {
	count := 0
	var walk func([]api.TopologyMachine)
	walk = func(machines []api.TopologyMachine) {
		count += len(machines)
		for _, machine := range machines {
			walk(machine.Children)
		}
	}
	for _, provider := range providers {
		for _, area := range provider.Areas {
			walk(area.Machines)
		}
	}
	return count
}

func consoleInstanceCount(providers []api.TopologyProvider) int {
	count := 0
	var walk func([]api.TopologyMachine)
	walk = func(machines []api.TopologyMachine) {
		for _, machine := range machines {
			for _, service := range machine.Services {
				count += len(service.Instances)
			}
			walk(machine.Children)
		}
	}
	for _, provider := range providers {
		for _, area := range provider.Areas {
			walk(area.Machines)
		}
	}
	return count
}

func consoleByteSize(value *int64) string {
	if value == nil {
		return ""
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	size := float64(*value)
	unit := 0
	for size >= 1000 && unit < len(units)-1 {
		size /= 1000
		unit++
	}
	if size >= 10 || unit == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unit])
	}
	return fmt.Sprintf("%.1f %s", size, units[unit])
}

func consoleInitials(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "--"
	}
	initials := []rune(strings.ToUpper(parts[0]))[:1]
	if len(parts) > 1 {
		initials = append(initials, []rune(strings.ToUpper(parts[len(parts)-1]))[0])
	}
	return string(initials)
}

func consoleKindLabel(kind api.MachineKind) string {
	return strings.ToUpper(strings.ReplaceAll(string(kind), "_", " "))
}

func consoleOSLabel(machine api.TopologyMachine) string {
	parts := make([]string, 0, 2)
	if machine.OperatingSystem != nil {
		parts = append(parts, *machine.OperatingSystem)
	}
	if machine.OperatingSystemVersion != nil {
		parts = append(parts, *machine.OperatingSystemVersion)
	}
	return strings.Join(parts, " ")
}

func consoleMachineViews(machines []api.TopologyMachine, expandAll bool) []consoleMachineView {
	result := make([]consoleMachineView, 0, len(machines))
	for _, machine := range machines {
		result = append(result, consoleMachineView{TopologyMachine: machine, ExpandAll: expandAll, ShowChildren: true, SSH: consoleSSHControlFor(machine)})
	}
	return result
}

func consoleSSHControlFor(machine api.TopologyMachine) *consoleSSHControl {
	if len(machine.Users) == 0 {
		return nil
	}
	targets := make([]consoleSSHTarget, 0, 1+2*len(machine.Addresses))
	seen := make(map[string]struct{}, cap(targets))
	addTarget := func(label, host string) {
		if host == "" {
			return
		}
		if _, exists := seen[host]; exists {
			return
		}
		seen[host] = struct{}{}
		targets = append(targets, consoleSSHTarget{Label: label, Host: host})
	}
	if machine.Hostname != nil {
		addTarget("Hostname", *machine.Hostname)
	}
	for _, address := range machine.Addresses {
		network := strings.ToUpper(strings.ReplaceAll(string(address.NetworkKind), "_", " "))
		if address.DnsName != nil {
			addTarget(network+" DNS", *address.DnsName)
		}
		addTarget(network+" IP", address.Address)
	}
	if len(targets) == 0 {
		return nil
	}

	users := append([]api.TopologyMachineUser(nil), machine.Users...)
	sort.SliceStable(users, func(i, j int) bool {
		if users[i].IsPreferred != users[j].IsPreferred {
			return users[i].IsPreferred
		}
		return users[i].Username < users[j].Username
	})
	sshUsers := make([]consoleSSHUser, 0, len(users))
	for _, user := range users {
		sshUsers = append(sshUsers, consoleSSHUser{Username: string(user.Username), IsPreferred: user.IsPreferred})
	}
	return &consoleSSHControl{Users: sshUsers, Targets: targets}
}

func consoleFavoriteMachines(providers []api.TopologyProvider) []api.TopologyMachine {
	favorites := []api.TopologyMachine{}
	var walk func([]api.TopologyMachine)
	walk = func(machines []api.TopologyMachine) {
		for _, machine := range machines {
			if machine.IsFavorite {
				favorites = append(favorites, machine)
			}
			walk(machine.Children)
		}
	}
	for _, provider := range providers {
		for _, area := range provider.Areas {
			walk(area.Machines)
		}
	}
	return favorites
}
