// Package tui implements the interactive HLIMS terminal console.
package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	narrowWidth      = 88
	serviceCardWidth = 34
)

type topologyLoader interface {
	Topology(context.Context) (*apiclient.Topology, error)
}

type topologyMsg struct {
	requestID uint64
	topology  *apiclient.Topology
	err       error
}

type nodeKind uint8

const (
	providerNode nodeKind = iota
	areaNode
	machineNode
)

type treeNode struct {
	key        string
	parentKey  string
	name       string
	kind       nodeKind
	depth      int
	expandable bool
	machine    *apiclient.TopologyMachine
}

// Model is the root Bubble Tea model for the HLIMS console.
type Model struct {
	ctx       context.Context
	client    topologyLoader
	topology  *apiclient.Topology
	expanded  map[string]bool
	selected  int
	requestID uint64
	spinner   spinner.Model
	loading   bool
	err       error
	width     int
	height    int
}

var (
	background    = lipgloss.Color("#282c34")
	textColor     = lipgloss.Color("#abb2bf")
	blue          = lipgloss.Color("#61afef")
	purple        = lipgloss.Color("#c678dd")
	green         = lipgloss.Color("#98c379")
	errorColor    = lipgloss.Color("#e06c75")
	panelBorder   = lipgloss.Color("#5c6370")
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(textColor)
	sectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(purple)
	labelStyle    = lipgloss.NewStyle().Bold(true).Foreground(blue)
	valueStyle    = lipgloss.NewStyle().Foreground(textColor)
	mutedStyle    = lipgloss.NewStyle().Foreground(panelBorder)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(background).Background(blue)
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(panelBorder).Padding(0, 1)
	machineStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(blue).Padding(0, 1)
	serviceStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(panelBorder).Padding(0, 1)
	healthyStyle  = lipgloss.NewStyle().Foreground(green)
	errorStyle    = lipgloss.NewStyle().Foreground(errorColor)
)

// NewModel creates the initial console model.
func NewModel(ctx context.Context, client topologyLoader) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(blue)

	return Model{
		ctx:       ctx,
		client:    client,
		expanded:  make(map[string]bool),
		requestID: 1,
		spinner:   s,
		loading:   true,
	}
}

// Init loads the inventory topology.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadTopology())
}

// Update handles keyboard input, API responses, and terminal resizing.
func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.moveSelection(-1)
			return m, nil
		case "down", "j":
			m.moveSelection(1)
			return m, nil
		case "enter", " ", "right", "l":
			m.expandSelection()
			return m, nil
		case "left", "h":
			m.collapseSelection()
			return m, nil
		case "r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, m.beginLoad())
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case topologyMsg:
		if msg.requestID != m.requestID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil && msg.topology != nil {
			m.installTopology(msg.topology)
		}
	}

	if m.loading {
		var command tea.Cmd
		m.spinner, command = m.spinner.Update(message)
		return m, command
	}
	return m, nil
}

// View renders the topology tree and selected machine details.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading HLIMS console...\n"
	}

	innerWidth := max(8, m.width-4)
	header := titleStyle.Render(truncate("HLIMS OPERATIONS CONSOLE  /  TOPOLOGY", max(1, m.width-2)))
	help := "up/down or j/k move  enter/space/right expand  left collapse/parent  r refresh  q quit"
	if m.width < narrowWidth {
		help = "j/k move  enter/right open  left back\nr refresh  q quit"
	}
	footer := mutedStyle.Render(truncateLines(help, max(1, m.width-2)))

	var body string
	if m.width < narrowWidth {
		treeHeight := max(4, (m.height-13)/2)
		body = m.renderTree(innerWidth, treeHeight) + "\n" + m.renderDetails(innerWidth)
	} else {
		treeWidth := min(50, max(32, (m.width-7)*2/5))
		detailWidth := max(32, m.width-7-treeWidth)
		panelHeight := max(6, m.height-8)
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			m.renderTree(treeWidth, panelHeight),
			" ",
			m.renderDetails(detailWidth),
		)
	}

	return lipgloss.NewStyle().Background(background).Foreground(textColor).Padding(1).Render(header + "\n" + body + "\n" + footer)
}

func (m *Model) installTopology(topology *apiclient.Topology) {
	selectedKey := m.selectedKey()
	m.topology = topology

	for providerIndex := range topology.Providers {
		provider := &topology.Providers[providerIndex]
		if _, exists := m.expanded[providerKey(provider.PublicId)]; !exists {
			m.expanded[providerKey(provider.PublicId)] = true
		}
		for areaIndex := range provider.Areas {
			area := &provider.Areas[areaIndex]
			if _, exists := m.expanded[areaKey(area.PublicId)]; !exists {
				m.expanded[areaKey(area.PublicId)] = true
			}
		}
	}

	nodes := m.visibleNodes()
	if index := nodeIndex(nodes, selectedKey); index >= 0 {
		m.selected = index
		return
	}
	for index := range nodes {
		if nodes[index].kind == machineNode {
			m.selected = index
			return
		}
	}
	m.selected = min(m.selected, max(0, len(nodes)-1))
}

func (m Model) visibleNodes() []treeNode {
	if m.topology == nil {
		return nil
	}

	var nodes []treeNode
	for providerIndex := range m.topology.Providers {
		provider := &m.topology.Providers[providerIndex]
		providerID := providerKey(provider.PublicId)
		nodes = append(nodes, treeNode{
			key: providerID, name: provider.Name, kind: providerNode, expandable: len(provider.Areas) > 0,
		})
		if !m.expanded[providerID] {
			continue
		}
		for areaIndex := range provider.Areas {
			area := &provider.Areas[areaIndex]
			areaID := areaKey(area.PublicId)
			nodes = append(nodes, treeNode{
				key: areaID, parentKey: providerID, name: area.Name, kind: areaNode, depth: 1, expandable: len(area.Machines) > 0,
			})
			if !m.expanded[areaID] {
				continue
			}
			for machineIndex := range area.Machines {
				m.appendMachineNodes(&nodes, &area.Machines[machineIndex], areaID, 2)
			}
		}
	}
	return nodes
}

func (m Model) appendMachineNodes(nodes *[]treeNode, machine *apiclient.TopologyMachine, parentKey string, depth int) {
	key := machineKey(machine.PublicId)
	*nodes = append(*nodes, treeNode{
		key: key, parentKey: parentKey, name: machine.Name, kind: machineNode, depth: depth,
		expandable: len(machine.Children) > 0, machine: machine,
	})
	if !m.expanded[key] {
		return
	}
	for childIndex := range machine.Children {
		m.appendMachineNodes(nodes, &machine.Children[childIndex], key, depth+1)
	}
}

func (m *Model) moveSelection(delta int) {
	nodes := m.visibleNodes()
	if len(nodes) == 0 {
		m.selected = 0
		return
	}
	m.selected = max(0, min(len(nodes)-1, m.selected+delta))
}

func (m *Model) expandSelection() {
	nodes := m.visibleNodes()
	if m.selected < 0 || m.selected >= len(nodes) || !nodes[m.selected].expandable {
		return
	}
	m.expanded[nodes[m.selected].key] = true
}

func (m *Model) collapseSelection() {
	nodes := m.visibleNodes()
	if m.selected < 0 || m.selected >= len(nodes) {
		return
	}
	node := nodes[m.selected]
	if node.expandable && m.expanded[node.key] {
		m.expanded[node.key] = false
		return
	}
	if parentIndex := nodeIndex(nodes, node.parentKey); parentIndex >= 0 {
		m.selected = parentIndex
	}
}

func (m Model) selectedKey() string {
	nodes := m.visibleNodes()
	if m.selected < 0 || m.selected >= len(nodes) {
		return ""
	}
	return nodes[m.selected].key
}

func (m Model) selectedMachine() *apiclient.TopologyMachine {
	node := m.selectedNode()
	return node.machine
}

func (m Model) selectedNode() treeNode {
	nodes := m.visibleNodes()
	if m.selected < 0 || m.selected >= len(nodes) {
		return treeNode{}
	}
	return nodes[m.selected]
}

func (m Model) immediateHost(node treeNode) *apiclient.TopologyMachine {
	if node.kind != machineNode || !strings.HasPrefix(node.parentKey, "machine:") {
		return nil
	}
	for _, candidate := range m.visibleNodes() {
		if candidate.key == node.parentKey {
			return candidate.machine
		}
	}
	return nil
}

func (m Model) renderTree(width, height int) string {
	lines := []string{sectionStyle.Render("INFRASTRUCTURE TREE"), mutedStyle.Render("TYPE  NAME")}
	nodes := m.visibleNodes()
	if len(nodes) == 0 {
		message := "No topology records."
		if m.loading {
			message = m.spinner.View() + " Loading topology..."
		} else if m.err != nil {
			message = m.err.Error()
		}
		lines = append(lines, renderStatus(message, m.err != nil, width))
		return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
	}

	viewportHeight := max(1, height-3)
	start := max(0, m.selected-viewportHeight+1)
	end := min(len(nodes), start+viewportHeight)
	if end-start < viewportHeight {
		start = max(0, end-viewportHeight)
	}
	for index := start; index < end; index++ {
		node := nodes[index]
		marker := "   "
		if node.expandable {
			marker = "[+]"
			if m.expanded[node.key] {
				marker = "[-]"
			}
		}
		kind := map[nodeKind]string{providerNode: "P", areaNode: "A", machineNode: "M"}[node.kind]
		line := strings.Repeat("  ", node.depth) + marker + " " + kind + " " + node.name
		line = truncate(line, max(1, width))
		if index == m.selected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	if m.loading {
		lines = append(lines, mutedStyle.Render(m.spinner.View()+" Refreshing topology..."))
	} else if m.err != nil {
		lines = append(lines, errorStyle.Render(truncate("Refresh failed: "+m.err.Error(), width)))
	}
	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func (m Model) renderDetails(width int) string {
	lines := []string{sectionStyle.Render("MACHINE")}
	node := m.selectedNode()
	machine := node.machine
	if machine == nil {
		lines = append(lines, mutedStyle.Render(truncate("Select a machine to inspect capacity and workloads.", width)))
		return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
	}

	os := joinPresent(pointerValue(machine.OperatingSystem), pointerValue(machine.OperatingSystemVersion))
	host := "-"
	if parent := m.immediateHost(node); parent != nil {
		host = parent.Name
		if hostname := pointerValue(parent.Hostname); hostname != "-" {
			host += " / " + hostname
		}
	}
	cardWidth := max(1, width-4)
	facts := strings.Join([]string{
		"CPU " + cpuDescription(machine),
		"MEM " + byteCount(machine.MemoryBytes),
		"DISK " + storageDescription(machine),
	}, "  |  ")
	machineCard := []string{
		detailLine("KIND", strings.ToUpper(string(machine.Kind)), cardWidth),
		detailLine("NAME", machine.Name, cardWidth),
		detailLine("HOSTNAME", pointerValue(machine.Hostname), cardWidth),
		detailLine("IMMEDIATE HOST", host, cardWidth),
		mutedStyle.Render(truncate(facts, cardWidth)),
		detailLine("SYSTEM", os, cardWidth),
		detailLine("KERNEL", pointerValue(machine.Kernel), cardWidth),
		detailLine("ARCH", pointerValue(machine.Architecture), cardWidth),
		detailLine("ID", machine.PublicId, cardWidth),
	}
	lines = append(lines, machineStyle.Width(cardWidth).Render(strings.Join(machineCard, "\n")))

	lines = append(lines, "", sectionStyle.Render(fmt.Sprintf("SERVICES [%d]", len(machine.Services))))
	if len(machine.Services) == 0 {
		lines = append(lines, mutedStyle.Render("No services assigned."))
	} else {
		lines = append(lines, renderServices(machine.Services, max(1, width)))
	}

	return panelStyle.Width(width).Render(strings.Join(lines, "\n"))
}

func renderServices(services []apiclient.TopologyService, width int) string {
	cardWidth := min(serviceCardWidth, width)
	columns := max(1, width/(cardWidth+1))
	var rows []string
	for start := 0; start < len(services); start += columns {
		end := min(len(services), start+columns)
		cards := make([]string, 0, end-start)
		for _, service := range services[start:end] {
			contentWidth := max(1, cardWidth-4)
			lines := []string{titleStyle.Render(truncate(service.Name, contentWidth))}
			if len(service.Instances) == 0 {
				lines = append(lines, mutedStyle.Render("No instances."))
			}
			for _, instance := range service.Instances {
				lines = append(lines,
					healthyStyle.Render(truncate(fmt.Sprintf("%s  :%d", instance.Name, instance.Port), contentWidth)),
					mutedStyle.Render(truncate(instance.ResolverPath, contentWidth)),
				)
			}
			cards = append(cards, serviceStyle.Width(contentWidth).Render(strings.Join(lines, "\n")))
		}
		row := cards[0]
		for _, card := range cards[1:] {
			row = lipgloss.JoinHorizontal(lipgloss.Top, row, " ", card)
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}

func (m Model) loadTopology() tea.Cmd {
	requestID := m.requestID
	return func() tea.Msg {
		topology, err := m.client.Topology(m.ctx)
		return topologyMsg{requestID: requestID, topology: topology, err: err}
	}
}

func (m *Model) beginLoad() tea.Cmd {
	m.requestID++
	return m.loadTopology()
}

func providerKey(publicID string) string { return "provider:" + publicID }
func areaKey(publicID string) string     { return "area:" + publicID }
func machineKey(publicID string) string  { return "machine:" + publicID }

func nodeIndex(nodes []treeNode, key string) int {
	for index := range nodes {
		if nodes[index].key == key {
			return index
		}
	}
	return -1
}

func pointerValue(value *string) string {
	if value == nil || *value == "" {
		return "-"
	}
	return *value
}

func joinPresent(values ...string) string {
	var present []string
	for _, value := range values {
		if value != "-" && value != "" {
			present = append(present, value)
		}
	}
	if len(present) == 0 {
		return "-"
	}
	return strings.Join(present, " ")
}

func cpuDescription(machine *apiclient.TopologyMachine) string {
	var values []string
	if machine.CpuCount != nil {
		values = append(values, fmt.Sprintf("%d cores", *machine.CpuCount))
	}
	if machine.CpuVendor != nil && *machine.CpuVendor != "" {
		values = append(values, *machine.CpuVendor)
	}
	if machine.CpuAllocation != nil {
		values = append(values, string(*machine.CpuAllocation))
	}
	return joinPresent(values...)
}

func storageDescription(machine *apiclient.TopologyMachine) string {
	values := []string{byteCount(machine.StorageBytes)}
	if machine.StorageMediaKind != nil {
		values = append(values, string(*machine.StorageMediaKind))
	}
	if machine.StorageInterfaceKind != nil {
		values = append(values, string(*machine.StorageInterfaceKind))
	}
	return joinPresent(values...)
}

func byteCount(value *int64) string {
	if value == nil {
		return "-"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	amount := float64(*value)
	unit := 0
	for amount >= 1024 && unit < len(units)-1 {
		amount /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", *value, units[unit])
	}
	return fmt.Sprintf("%.1f %s", amount, units[unit])
}

func detailLine(label, value string, width int) string {
	label = fmt.Sprintf("%-16s", label)
	if width <= len(label) {
		return labelStyle.Render(truncate(strings.TrimSpace(label), width))
	}
	return labelStyle.Render(label) + valueStyle.Render(truncate(value, max(1, width-len(label))))
}

func renderStatus(message string, isError bool, width int) string {
	message = truncate(message, width)
	if isError {
		return errorStyle.Render(message)
	}
	return mutedStyle.Render(message)
}

func truncate(value string, width int) string {
	characters := []rune(value)
	if len(characters) <= width {
		return value
	}
	if width <= 3 {
		return string(characters[:max(0, width)])
	}
	return string(characters[:width-3]) + "..."
}

func truncateLines(value string, width int) string {
	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = truncate(lines[index], width)
	}
	return strings.Join(lines, "\n")
}

// Run starts the interactive console in the terminal's alternate screen.
func Run(ctx context.Context, client topologyLoader, input io.Reader, output io.Writer) error {
	program := tea.NewProgram(NewModel(ctx, client), tea.WithContext(ctx), tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen())
	_, err := program.Run()
	return err
}
