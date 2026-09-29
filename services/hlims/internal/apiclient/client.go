// Package apiclient provides the client-side boundary for HLIMS commands.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	api "github.com/TylerHillery/homelab/services/hlims/generated/api"
)

const maxResponseSize = 16 << 20

// Resource identifies an inventory collection exposed by the API.
type Resource string

const (
	MachineProviders  Resource = "machine-providers"
	Areas             Resource = "areas"
	Manufacturers     Resource = "manufacturers"
	Products          Resource = "products"
	Assets            Resource = "assets"
	Purchases         Resource = "purchases"
	Machines          Resource = "machines"
	MachineUsers      Resource = "machine-users"
	Networks          Resource = "networks"
	Addresses         Resource = "addresses"
	Services          Resource = "services"
	Instances         Resource = "instances"
	InstanceEndpoints Resource = "instance-endpoints"
)

var resources = []Resource{
	MachineProviders,
	Areas,
	Manufacturers,
	Products,
	Assets,
	Purchases,
	Machines,
	MachineUsers,
	Networks,
	Addresses,
	Services,
	Instances,
	InstanceEndpoints,
}

// Resources returns the inventory resources in their parent-first order.
func Resources() []Resource {
	return append([]Resource(nil), resources...)
}

// Label returns a human-readable resource name.
func (r Resource) Label() string {
	return strings.NewReplacer("-", " ").Replace(string(r))
}

// SingularLabel returns a human-readable name for one resource record.
func (r Resource) SingularLabel() string {
	switch r {
	case MachineProviders:
		return "machine provider"
	case Areas:
		return "area"
	case Manufacturers:
		return "manufacturer"
	case Products:
		return "product"
	case Assets:
		return "asset"
	case Purchases:
		return "purchase"
	case Machines:
		return "machine"
	case MachineUsers:
		return "machine user"
	case Networks:
		return "network"
	case Addresses:
		return "address"
	case Services:
		return "service"
	case Instances:
		return "instance"
	case InstanceEndpoints:
		return "instance endpoint"
	default:
		return string(r)
	}
}

// Record is the common information used to present inventory in the console.
type Record struct {
	PublicID string
	Name     string
	Slug     string
	Detail   string
}

// Topology is the shared OpenAPI topology model returned to interactive clients.
type Topology = api.Topology

type (
	TopologyProvider = api.TopologyProvider
	TopologyArea     = api.TopologyArea
	TopologyMachine  = api.TopologyMachine
	TopologyService  = api.TopologyService
	TopologyInstance = api.TopologyInstance
)

// Response contains a successful API response.
type Response struct {
	StatusCode int
	Body       []byte
	ETag       string
}

// Error represents a non-successful response from the API.
type Error struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("HLIMS API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("HLIMS API returned HTTP %d (%s): %s", e.StatusCode, e.Code, e.Message)
}

// Client accesses HLIMS exclusively through its generated OpenAPI client.
type Client struct {
	api        api.ClientInterface
	apiURL     *url.URL
	httpClient *http.Client
}

// New constructs a client for an API URL such as http://127.0.0.1:8080/api/v1.
func New(apiURL string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("parse API URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("API URL must use http or https")
	}
	if parsed.Host == "" {
		return nil, errors.New("API URL must include a host")
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	generated, err := api.NewClient(strings.TrimRight(parsed.String(), "/"), api.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create generated API client: %w", err)
	}
	return &Client{api: generated, apiURL: parsed, httpClient: httpClient}, nil
}

// List returns one inventory collection.
func (c *Client) List(ctx context.Context, resource Resource) (*Response, error) {
	return c.ListFiltered(ctx, resource, ListFilter{})
}

// ListFilter selects records by their canonical slugs or parent public ID.
type ListFilter struct {
	Slug             string
	Service          string
	Machine          string
	HostingKind      string
	InstancePublicID string
}

// ListFiltered returns records matching the supplied resource-specific filters.
func (c *Client) ListFiltered(ctx context.Context, resource Resource, filter ListFilter) (*Response, error) {
	var response *http.Response
	var err error
	switch resource {
	case MachineProviders:
		response, err = c.api.ListMachineProviders(ctx)
	case Areas:
		response, err = c.api.ListAreas(ctx)
	case Manufacturers:
		response, err = c.api.ListManufacturers(ctx)
	case Products:
		response, err = c.api.ListProducts(ctx)
	case Assets:
		response, err = c.api.ListAssets(ctx)
	case Purchases:
		response, err = c.api.ListPurchases(ctx)
	case Machines:
		params := &api.ListMachinesParams{}
		if filter.Slug != "" {
			params.Slug = &filter.Slug
		}
		response, err = c.api.ListMachines(ctx, params)
	case MachineUsers:
		response, err = c.api.ListMachineUsers(ctx)
	case Networks:
		response, err = c.api.ListNetworks(ctx)
	case Addresses:
		response, err = c.api.ListAddresses(ctx)
	case Services:
		params := &api.ListServicesParams{}
		if filter.Slug != "" {
			params.Slug = &filter.Slug
		}
		response, err = c.api.ListServices(ctx, params)
	case Instances:
		params := &api.ListInstancesParams{}
		if filter.Slug != "" {
			params.Slug = &filter.Slug
		}
		if filter.Service != "" {
			params.Service = &filter.Service
		}
		if filter.Machine != "" {
			params.Machine = &filter.Machine
		}
		if filter.HostingKind != "" {
			kind := api.InstanceHostingKind(filter.HostingKind)
			params.HostingKind = &kind
		}
		response, err = c.api.ListInstances(ctx, params)
	case InstanceEndpoints:
		params := &api.ListInstanceEndpointsParams{}
		if filter.InstancePublicID != "" {
			params.InstancePublicId = &filter.InstancePublicID
		}
		response, err = c.api.ListInstanceEndpoints(ctx, params)
	default:
		return nil, fmt.Errorf("unknown resource %q", resource)
	}
	return readResponse(response, err)
}

// Topology fetches and decodes a complete inventory topology over HTTP.
func (c *Client) Topology(ctx context.Context) (*Topology, error) {
	response, err := c.api.GetTopology(ctx)
	result, err := readResponse(response, err)
	if err != nil {
		return nil, err
	}
	var topology Topology
	if err := json.Unmarshal(result.Body, &topology); err != nil {
		return nil, fmt.Errorf("decode topology response: %w", err)
	}
	return &topology, nil
}

// PurchaseSummary returns recorded Asset purchase totals grouped by currency.
func (c *Client) PurchaseSummary(ctx context.Context) (*Response, error) {
	response, err := c.api.GetPurchaseSummary(ctx)
	return readResponse(response, err)
}

// Get returns one inventory resource by public ID.
func (c *Client) Get(ctx context.Context, resource Resource, publicID string) (*Response, error) {
	var response *http.Response
	var err error
	switch resource {
	case MachineProviders:
		response, err = c.api.GetMachineProvider(ctx, publicID)
	case Areas:
		response, err = c.api.GetArea(ctx, publicID)
	case Manufacturers:
		response, err = c.api.GetManufacturer(ctx, publicID)
	case Products:
		response, err = c.api.GetProduct(ctx, publicID)
	case Assets:
		response, err = c.api.GetAsset(ctx, publicID)
	case Purchases:
		response, err = c.api.GetPurchase(ctx, publicID)
	case Machines:
		response, err = c.api.GetMachine(ctx, publicID)
	case MachineUsers:
		response, err = c.api.GetMachineUser(ctx, publicID)
	case Networks:
		response, err = c.api.GetNetwork(ctx, publicID)
	case Addresses:
		response, err = c.api.GetAddress(ctx, publicID)
	case Services:
		response, err = c.api.GetService(ctx, publicID)
	case Instances:
		response, err = c.api.GetInstance(ctx, publicID)
	case InstanceEndpoints:
		response, err = c.api.GetInstanceEndpoint(ctx, publicID)
	default:
		return nil, fmt.Errorf("unknown resource %q", resource)
	}
	return readResponse(response, err)
}

// ResolvePublicID accepts a public ID or an exact slug for resources that have slugs.
// Instance slugs can repeat across Machines, so callers may narrow the lookup.
func (c *Client) ResolvePublicID(ctx context.Context, resource Resource, reference string, filter ListFilter) (string, error) {
	if !hasSlug(resource) {
		return reference, nil
	}
	if looksLikePublicID(reference) {
		_, err := c.Get(ctx, resource, reference)
		if err == nil {
			return reference, nil
		}
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return "", err
		}
	}
	if resource == Services || resource == Machines || resource == Instances {
		filter.Slug = reference
	}
	response, err := c.ListFiltered(ctx, resource, filter)
	if err != nil {
		return "", err
	}
	var result struct {
		Items []struct {
			PublicID string `json:"publicId"`
			Slug     string `json:"slug"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response.Body, &result); err != nil {
		return "", fmt.Errorf("decode %s lookup: %w", resource, err)
	}
	publicID := ""
	for _, item := range result.Items {
		if item.Slug != reference {
			continue
		}
		if publicID != "" {
			return "", fmt.Errorf("%s slug %q is ambiguous; filter by parent or use a public ID", resource, reference)
		}
		publicID = item.PublicID
	}
	if publicID == "" {
		return "", fmt.Errorf("%s slug %q not found", resource, reference)
	}
	return publicID, nil
}

func hasSlug(resource Resource) bool {
	switch resource {
	case MachineProviders, Areas, Manufacturers, Machines, Networks, Services, Instances:
		return true
	default:
		return false
	}
}

func looksLikePublicID(value string) bool {
	if len(value) != 12 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' && char < 'a' || char > 'z' {
			return false
		}
	}
	return true
}

// Create creates one inventory resource from an OpenAPI-compatible JSON body.
func (c *Client) Create(ctx context.Context, resource Resource, body []byte) (*Response, error) {
	var response *http.Response
	var err error
	switch resource {
	case MachineProviders:
		response, err = c.api.CreateMachineProviderWithBody(ctx, "application/json", bytes.NewReader(body))
	case Areas:
		response, err = c.api.CreateAreaWithBody(ctx, "application/json", bytes.NewReader(body))
	case Manufacturers:
		response, err = c.api.CreateManufacturerWithBody(ctx, "application/json", bytes.NewReader(body))
	case Products:
		response, err = c.api.CreateProductWithBody(ctx, "application/json", bytes.NewReader(body))
	case Assets:
		response, err = c.api.CreateAssetWithBody(ctx, "application/json", bytes.NewReader(body))
	case Purchases:
		response, err = c.api.CreatePurchaseWithBody(ctx, "application/json", bytes.NewReader(body))
	case Machines:
		response, err = c.api.CreateMachineWithBody(ctx, "application/json", bytes.NewReader(body))
	case MachineUsers:
		response, err = c.api.CreateMachineUserWithBody(ctx, "application/json", bytes.NewReader(body))
	case Networks:
		response, err = c.api.CreateNetworkWithBody(ctx, "application/json", bytes.NewReader(body))
	case Addresses:
		response, err = c.api.CreateAddressWithBody(ctx, "application/json", bytes.NewReader(body))
	case Services:
		response, err = c.api.CreateServiceWithBody(ctx, "application/json", bytes.NewReader(body))
	case Instances:
		response, err = c.api.CreateInstanceWithBody(ctx, "application/json", bytes.NewReader(body))
	case InstanceEndpoints:
		response, err = c.api.CreateInstanceEndpointWithBody(ctx, "application/json", bytes.NewReader(body))
	default:
		return nil, fmt.Errorf("unknown resource %q", resource)
	}
	return readResponse(response, err)
}

// Update replaces one inventory resource from an OpenAPI-compatible JSON body.
func (c *Client) Update(ctx context.Context, resource Resource, publicID string, body []byte) (*Response, error) {
	return c.UpdateIfMatch(ctx, resource, publicID, body, "")
}

// UpdateIfMatch replaces one inventory resource only if its GET ETag still matches.
func (c *Client) UpdateIfMatch(ctx context.Context, resource Resource, publicID string, body []byte, etag string) (*Response, error) {
	var response *http.Response
	var err error
	var editors []api.RequestEditorFn
	if etag != "" {
		editors = append(editors, func(_ context.Context, request *http.Request) error {
			request.Header.Set("If-Match", etag)
			return nil
		})
	}
	switch resource {
	case MachineProviders:
		response, err = c.api.UpdateMachineProviderWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Areas:
		response, err = c.api.UpdateAreaWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Manufacturers:
		response, err = c.api.UpdateManufacturerWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Products:
		response, err = c.api.UpdateProductWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Assets:
		response, err = c.api.UpdateAssetWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Purchases:
		response, err = c.api.UpdatePurchaseWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Machines:
		response, err = c.api.UpdateMachineWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case MachineUsers:
		response, err = c.api.UpdateMachineUserWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Networks:
		response, err = c.api.UpdateNetworkWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Addresses:
		response, err = c.api.UpdateAddressWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Services:
		response, err = c.api.UpdateServiceWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case Instances:
		response, err = c.api.UpdateInstanceWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	case InstanceEndpoints:
		response, err = c.api.UpdateInstanceEndpointWithBody(ctx, publicID, "application/json", bytes.NewReader(body), editors...)
	default:
		return nil, fmt.Errorf("unknown resource %q", resource)
	}
	return readResponse(response, err)
}

// Delete removes one inventory resource by public ID.
func (c *Client) Delete(ctx context.Context, resource Resource, publicID string) (*Response, error) {
	var response *http.Response
	var err error
	switch resource {
	case MachineProviders:
		response, err = c.api.DeleteMachineProvider(ctx, publicID)
	case Areas:
		response, err = c.api.DeleteArea(ctx, publicID)
	case Manufacturers:
		response, err = c.api.DeleteManufacturer(ctx, publicID)
	case Products:
		response, err = c.api.DeleteProduct(ctx, publicID)
	case Assets:
		response, err = c.api.DeleteAsset(ctx, publicID)
	case Purchases:
		response, err = c.api.DeletePurchase(ctx, publicID)
	case Machines:
		response, err = c.api.DeleteMachine(ctx, publicID)
	case MachineUsers:
		response, err = c.api.DeleteMachineUser(ctx, publicID)
	case Networks:
		response, err = c.api.DeleteNetwork(ctx, publicID)
	case Addresses:
		response, err = c.api.DeleteAddress(ctx, publicID)
	case Services:
		response, err = c.api.DeleteService(ctx, publicID)
	case Instances:
		response, err = c.api.DeleteInstance(ctx, publicID)
	case InstanceEndpoints:
		response, err = c.api.DeleteInstanceEndpoint(ctx, publicID)
	default:
		return nil, fmt.Errorf("unknown resource %q", resource)
	}
	return readResponse(response, err)
}

// GetServiceLogo returns the raw logo associated with a Service.
func (c *Client) GetServiceLogo(ctx context.Context, publicID string) (*Response, error) {
	response, err := c.api.GetServiceLogo(ctx, publicID)
	return readResponse(response, err)
}

// UpdateServiceLogo stores raw PNG, JPEG, or WebP bytes for a Service.
func (c *Client) UpdateServiceLogo(ctx context.Context, publicID, contentType string, body []byte) (*Response, error) {
	response, err := c.api.UpdateServiceLogoWithBody(ctx, publicID, contentType, bytes.NewReader(body))
	return readResponse(response, err)
}

// DeleteServiceLogo removes the logo associated with a Service.
func (c *Client) DeleteServiceLogo(ctx context.Context, publicID string) (*Response, error) {
	response, err := c.api.DeleteServiceLogo(ctx, publicID)
	return readResponse(response, err)
}

// GetMachineProviderLogo returns the raw logo associated with a Machine Provider.
func (c *Client) GetMachineProviderLogo(ctx context.Context, publicID string) (*Response, error) {
	response, err := c.api.GetMachineProviderLogo(ctx, publicID)
	return readResponse(response, err)
}

// UpdateMachineProviderLogo stores raw PNG, JPEG, or WebP bytes for a Machine Provider.
func (c *Client) UpdateMachineProviderLogo(ctx context.Context, publicID, contentType string, body []byte) (*Response, error) {
	response, err := c.api.UpdateMachineProviderLogoWithBody(ctx, publicID, contentType, bytes.NewReader(body))
	return readResponse(response, err)
}

// DeleteMachineProviderLogo removes the logo associated with a Machine Provider.
func (c *Client) DeleteMachineProviderLogo(ctx context.Context, publicID string) (*Response, error) {
	response, err := c.api.DeleteMachineProviderLogo(ctx, publicID)
	return readResponse(response, err)
}

// ResolveInstance resolves a Service and Instance, selecting a host when needed.
func (c *Client) ResolveInstance(ctx context.Context, service, instance, host, via string) (*Response, error) {
	params := &api.ResolveInstanceParams{}
	if host != "" {
		value := api.Host(host)
		params.Host = &value
	}
	if via != "" {
		value := api.Via(via)
		params.Via = &value
	}
	response, err := c.api.ResolveInstance(ctx, service, instance, params)
	return readResponse(response, err)
}

// ResolvePort resolves an ad hoc Machine port route.
func (c *Client) ResolvePort(ctx context.Context, machine string, port int, via, scheme string) (*Response, error) {
	params := &api.ResolveMachinePortParams{}
	if via != "" {
		value := api.Via(via)
		params.Via = &value
	}
	if scheme != "" {
		value := api.Scheme(scheme)
		params.Scheme = &value
	}
	response, err := c.api.ResolveMachinePort(ctx, machine, port, params)
	return readResponse(response, err)
}

// DestinationURL decodes a resolver response into its destination URL.
func DestinationURL(response *Response) (string, error) {
	if response == nil {
		return "", errors.New("resolver returned no response")
	}
	var destination api.ResolvedDestination
	if err := json.Unmarshal(response.Body, &destination); err != nil {
		return "", fmt.Errorf("decode resolver response: %w", err)
	}
	if destination.Url == "" {
		return "", errors.New("resolver response did not include a URL")
	}
	return destination.Url, nil
}

// OpenAPISpec fetches the OpenAPI document published by the target server.
func (c *Client) OpenAPISpec(ctx context.Context) (*Response, error) {
	specURL := *c.apiURL
	apiPath := strings.TrimRight(specURL.Path, "/")
	if !strings.HasSuffix(apiPath, "/api/v1") {
		return nil, errors.New("API URL path must end with /api/v1 to locate /openapi.yaml")
	}
	specURL.Path = strings.TrimSuffix(apiPath, "/api/v1") + "/openapi.yaml"
	specURL.RawPath = ""
	specURL.RawQuery = ""
	specURL.Fragment = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, specURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create OpenAPI request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	return readResponse(response, err)
}

// ListRecords returns presentation-friendly fields for a collection.
func (c *Client) ListRecords(ctx context.Context, resource Resource) ([]Record, error) {
	response, err := c.List(ctx, resource)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", resource, err)
	}
	records := make([]Record, 0, len(payload.Items))
	for _, item := range payload.Items {
		name := stringValue(item["name"])
		if name == "" {
			name = stringValue(item["address"])
		}
		if name == "" {
			name = stringValue(item["username"])
		}
		records = append(records, Record{
			PublicID: stringValue(item["publicId"]),
			Name:     name,
			Slug:     stringValue(item["slug"]),
			Detail:   recordDetail(item),
		})
	}
	return records, nil
}

func readResponse(response *http.Response, requestErr error) (*Response, error) {
	if requestErr != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		return nil, fmt.Errorf("call HLIMS API: %w", requestErr)
	}
	if response == nil {
		return nil, errors.New("HLIMS API returned no response")
	}
	defer func() {
		_ = response.Body.Close()
	}()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read HLIMS API response: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, errors.New("HLIMS API response exceeds 16 MiB")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		apiErr := &Error{StatusCode: response.StatusCode}
		var payload api.Error
		if json.Unmarshal(body, &payload) == nil {
			apiErr.Code = payload.Code
			apiErr.Message = payload.Message
		}
		return nil, apiErr
	}
	return &Response{StatusCode: response.StatusCode, Body: body, ETag: response.Header.Get("ETag")}, nil
}

func recordDetail(item map[string]any) string {
	for _, key := range []string{"kind", "address", "scheme", "port", "providerCode"} {
		if value := stringValue(item[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}
