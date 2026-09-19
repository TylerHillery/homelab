package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
	tea "github.com/charmbracelet/bubbletea"
)

type fakeLoader struct {
	topology *apiclient.Topology
	err      error
	calls    int
}

func (f *fakeLoader) Topology(_ context.Context) (*apiclient.Topology, error) {
	f.calls++
	return f.topology, f.err
}

func TestInitialLoadBuildsExpandedHierarchyAndSelectsMachine(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{topology: testTopology()}
	model := NewModel(context.Background(), loader)
	message := model.loadTopology()()
	updated, _ := model.Update(message)
	result := updated.(Model)

	if loader.calls != 1 {
		t.Fatalf("Topology calls = %d, want 1", loader.calls)
	}
	if result.loading || result.err != nil {
		t.Fatalf("loading = %t, err = %v", result.loading, result.err)
	}
	nodes := result.visibleNodes()
	if len(nodes) != 3 {
		t.Fatalf("visible nodes = %d, want provider, area, root machine", len(nodes))
	}
	if nodes[result.selected].key != machineKey("machine-root") {
		t.Fatalf("selected key = %q", nodes[result.selected].key)
	}
}

func TestRecursiveMachineExpansionAndNavigation(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)

	model = updateKey(t, model, "right")
	nodes := model.visibleNodes()
	if len(nodes) != 4 || nodes[3].key != machineKey("machine-child") {
		t.Fatalf("nodes after root expansion = %#v", nodeKeys(nodes))
	}

	model = updateKey(t, model, "j")
	model = updateKey(t, model, " ")
	nodes = model.visibleNodes()
	if len(nodes) != 5 || nodes[4].key != machineKey("machine-grandchild") {
		t.Fatalf("nodes after child expansion = %#v", nodeKeys(nodes))
	}

	model = updateKey(t, model, "left")
	if model.expanded[machineKey("machine-child")] {
		t.Fatal("left did not collapse expanded child")
	}
	model = updateKey(t, model, "left")
	if model.selectedKey() != machineKey("machine-root") {
		t.Fatalf("left on collapsed child selected %q, want parent", model.selectedKey())
	}
}

func TestMovementIsBounded(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	model.selected = 0
	model = updateKey(t, model, "up")
	if model.selected != 0 {
		t.Fatalf("selection moved above first node: %d", model.selected)
	}
	for range 10 {
		model = updateKey(t, model, "down")
	}
	if model.selected != len(model.visibleNodes())-1 {
		t.Fatalf("selection = %d, want last node", model.selected)
	}
}

func TestEnterExpandsAndQuitReturnsQuitCommand(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if !model.expanded[machineKey("machine-root")] {
		t.Fatal("enter did not expand selected machine")
	}

	_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if command == nil {
		t.Fatal("q did not return a quit command")
	}
	if message := command(); message != tea.Quit() {
		t.Fatalf("q command returned %#v, want quit message", message)
	}
}

func TestRefreshPreservesSelectionExpansionAndSnapshotOnError(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	model = updateKey(t, model, "right")
	model = updateKey(t, model, "down")
	selected := model.selectedKey()

	model = updateKey(t, model, "r")
	if !model.loading || model.requestID != 2 {
		t.Fatalf("loading = %t, request ID = %d", model.loading, model.requestID)
	}
	updated, _ := model.Update(topologyMsg{requestID: 2, err: errors.New("unavailable")})
	model = updated.(Model)
	if model.loading || model.err == nil || model.topology == nil {
		t.Fatalf("refresh error state: loading=%t err=%v topology=%v", model.loading, model.err, model.topology)
	}
	if model.selectedKey() != selected || !model.expanded[machineKey("machine-root")] {
		t.Fatal("refresh error discarded tree state")
	}
}

func TestSuccessfulRefreshRetainsMachineSelectionByPublicID(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	model = updateKey(t, model, "right")
	model = updateKey(t, model, "down")
	model.requestID = 2

	replacement := testTopology()
	replacement.Providers[0].Areas[0].Machines[0].Children[0].Name = "Renamed Child"
	updated, _ := model.Update(topologyMsg{requestID: 2, topology: replacement})
	result := updated.(Model)
	if result.selectedKey() != machineKey("machine-child") {
		t.Fatalf("selected key = %q", result.selectedKey())
	}
	if result.selectedMachine().Name != "Renamed Child" {
		t.Fatalf("selected machine = %q", result.selectedMachine().Name)
	}
}

func TestStaleResponseDoesNotReplaceNewerTopology(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	current := model.topology
	model.requestID = 3
	model.loading = true

	updated, _ := model.Update(topologyMsg{requestID: 2, topology: &apiclient.Topology{}})
	result := updated.(Model)
	if result.topology != current || !result.loading {
		t.Fatal("stale response replaced or completed the current request")
	}
}

func TestDetailPaneShowsInfrastructureAndServiceLabels(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	view := model.renderDetails(70)
	for _, expected := range []string{
		"MACHINE", "KIND", "BARE_METAL", "NAME", "Root Host", "HOSTNAME", "root.example.test", "IMMEDIATE HOST",
		"CPU 8 cores AMD", "MEM 16.0 GiB", "DISK 1.0 TiB", "SYSTEM", "Linux 42",
		"SERVICES [1]", "Dashboard", "Production  :8443", "/root/dashboard/production",
	} {
		if !strings.Contains(view, expected) {
			t.Errorf("detail pane missing %q:\n%s", expected, view)
		}
	}
}

func TestNestedMachineShowsImmediateHostAncestry(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	model = updateKey(t, model, "right")
	model = updateKey(t, model, "down")
	model = updateKey(t, model, "right")
	model = updateKey(t, model, "down")

	view := model.renderDetails(70)
	if !strings.Contains(view, "IMMEDIATE HOST  Child VM") {
		t.Fatalf("nested machine does not show its immediate host:\n%s", view)
	}
	if strings.Contains(view, "IMMEDIATE HOST  Root Host") {
		t.Fatalf("nested machine incorrectly shows root ancestor as immediate host:\n%s", view)
	}
}

func TestServicesPackHorizontallyAtWideWidth(t *testing.T) {
	t.Parallel()
	services := testServices()
	view := renderServices(services, 80)
	first := strings.Index(view, "Dashboard")
	second := strings.Index(view, "Metrics")
	if first < 0 || second < 0 || strings.Count(view[first:second], "\n") != 0 {
		t.Fatalf("wide services were not packed on one row:\n%s", view)
	}
}

func TestServicesStackAtNarrowWidth(t *testing.T) {
	t.Parallel()
	view := renderServices(testServices(), 32)
	first := strings.Index(view, "Dashboard")
	second := strings.Index(view, "Metrics")
	if first < 0 || second < 0 || !strings.Contains(view[first:second], "\n") {
		t.Fatalf("narrow services were not stacked:\n%s", view)
	}
}

func TestNarrowViewStacksTreeAndDetails(t *testing.T) {
	t.Parallel()
	model := loadedModel(t)
	model.width = 70
	model.height = 30
	view := model.View()
	treeAt := strings.Index(view, "INFRASTRUCTURE TREE")
	detailAt := strings.Index(view, "MACHINE")
	if treeAt < 0 || detailAt <= treeAt {
		t.Fatalf("narrow view does not stack tree before details:\n%s", view)
	}
	if strings.Contains(view, "up/down or j/k") {
		t.Fatal("narrow view used the wide help line")
	}
}

func TestEmptyAndInitialErrorViews(t *testing.T) {
	t.Parallel()
	model := NewModel(context.Background(), &fakeLoader{})
	model.width = 80
	model.height = 24
	updated, _ := model.Update(topologyMsg{requestID: 1, err: errors.New("API offline")})
	view := updated.(Model).View()
	if !strings.Contains(view, "API offline") || !strings.Contains(view, "Select a machine") {
		t.Fatalf("error view missing state:\n%s", view)
	}
}

func loadedModel(t *testing.T) Model {
	t.Helper()
	model := NewModel(context.Background(), &fakeLoader{})
	updated, _ := model.Update(topologyMsg{requestID: 1, topology: testTopology()})
	return updated.(Model)
}

func updateKey(t *testing.T, model Model, key string) Model {
	t.Helper()
	updated, _ := model.Update(tea.KeyMsg{Type: keyType(key), Runes: keyRunes(key)})
	return updated.(Model)
}

func keyType(key string) tea.KeyType {
	switch key {
	case "up":
		return tea.KeyUp
	case "down":
		return tea.KeyDown
	case "left":
		return tea.KeyLeft
	case "right":
		return tea.KeyRight
	case " ":
		return tea.KeySpace
	default:
		return tea.KeyRunes
	}
}

func keyRunes(key string) []rune {
	if keyType(key) == tea.KeyRunes {
		return []rune(key)
	}
	return nil
}

func nodeKeys(nodes []treeNode) []string {
	keys := make([]string, len(nodes))
	for index := range nodes {
		keys[index] = nodes[index].key
	}
	return keys
}

func testTopology() *apiclient.Topology {
	hostname := "root.example.test"
	os := "Linux"
	version := "42"
	kernel := "6.12"
	architecture := "x86_64"
	cpuCount := 8
	cpuVendor := "AMD"
	memory := int64(16 * 1024 * 1024 * 1024)
	storage := int64(1024 * 1024 * 1024 * 1024)

	return &apiclient.Topology{Providers: []apiclient.TopologyProvider{{
		PublicId: "provider-main",
		Name:     "Primary Provider",
		Slug:     "primary-provider",
		Areas: []apiclient.TopologyArea{{
			PublicId: "area-main",
			Name:     "Main Area",
			Slug:     "main-area",
			Machines: []apiclient.TopologyMachine{{
				PublicId:               "machine-root",
				Name:                   "Root Host",
				Slug:                   "root",
				Kind:                   "bare_metal",
				Hostname:               &hostname,
				OperatingSystem:        &os,
				OperatingSystemVersion: &version,
				Kernel:                 &kernel,
				Architecture:           &architecture,
				CpuCount:               &cpuCount,
				CpuVendor:              &cpuVendor,
				MemoryBytes:            &memory,
				StorageBytes:           &storage,
				Children: []apiclient.TopologyMachine{{
					PublicId: "machine-child",
					Name:     "Child VM",
					Slug:     "child",
					Kind:     "virtual_machine",
					Children: []apiclient.TopologyMachine{{
						PublicId: "machine-grandchild",
						Name:     "Nested VM",
						Slug:     "nested",
						Kind:     "virtual_machine",
					}},
				}},
				Services: []apiclient.TopologyService{{
					PublicId: "service-dashboard",
					Name:     "Dashboard",
					Slug:     "dashboard",
					Instances: []apiclient.TopologyInstance{{
						PublicId:     "instance-production",
						Name:         "Production",
						Slug:         "production",
						Port:         8443,
						ResolverPath: "/root/dashboard/production",
					}},
				}},
			}},
		}},
	}}}
}

func testServices() []apiclient.TopologyService {
	return []apiclient.TopologyService{
		{Name: "Dashboard", Instances: []apiclient.TopologyInstance{{Name: "Production", Port: 8443, ResolverPath: "/dashboard/production"}}},
		{Name: "Metrics", Instances: []apiclient.TopologyInstance{{Name: "Prometheus", Port: 9090, ResolverPath: "/metrics/prometheus"}}},
	}
}
