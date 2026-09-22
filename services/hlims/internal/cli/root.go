// Package cli implements the user-facing HLIMS command line client.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
	"github.com/TylerHillery/homelab/services/hlims/internal/tui"
	"github.com/spf13/cobra"
)

const (
	defaultAPIURL = "http://127.0.0.1:8080/api/v1"
	maxInputSize  = 16 << 20
)

// Dependencies contains replaceable process boundaries used by the CLI.
type Dependencies struct {
	Input      io.Reader
	Output     io.Writer
	Error      io.Writer
	HTTPClient *http.Client
	OpenURL    func(context.Context, string) error
}

type options struct {
	apiURL  string
	compact bool
	timeout time.Duration
	deps    Dependencies
}

// NewRootCommand constructs the complete hlims command tree.
func NewRootCommand(deps Dependencies) *cobra.Command {
	if deps.Input == nil {
		deps.Input = os.Stdin
	}
	if deps.Output == nil {
		deps.Output = os.Stdout
	}
	if deps.Error == nil {
		deps.Error = os.Stderr
	}

	apiURL := os.Getenv("HLIMS_API_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	opts := &options{apiURL: apiURL, deps: deps}
	root := &cobra.Command{
		Use:   "hlims",
		Short: "Interact with the Home Lab Information Management System",
		Long: `Interact with HLIMS exclusively through its versioned HTTP API.

Inventory and resolver commands emit JSON to stdout and diagnostics to stderr.
Create and update commands read one JSON document from stdin by default or from
--file. Data commands never prompt; destructive operations require --yes. The
console command is explicitly interactive. This makes data commands safe to
compose with pipes, jq, scripts, and coding agents.`,
		Example: `  hlims machines list | jq '.items[] | {publicId, name}'
  printf '%s\n' '{"name":"Grafana"}' | hlims services create
  hlims services update SERVICE_ID --file service.json
  hlims open badger/grafana/production
  hlims console`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetIn(deps.Input)
	root.SetOut(deps.Output)
	root.SetErr(deps.Error)
	root.PersistentFlags().StringVar(&opts.apiURL, "api-url", apiURL, "HLIMS API base URL")
	root.PersistentFlags().BoolVar(&opts.compact, "compact", false, "emit compact JSON on one line")
	root.PersistentFlags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "HTTP request timeout")

	for _, resource := range apiclient.Resources() {
		root.AddCommand(newResourceCommand(opts, resource))
	}
	root.AddCommand(newPurchaseSummaryCommand(opts), newResolveCommand(opts), newOpenCommand(opts), newOpenAPICommand(opts), newConsoleCommand(opts))
	return root
}

func newPurchaseSummaryCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "purchase-summary",
		Short: "Show recorded equipment purchase totals by currency",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			response, err := client.PurchaseSummary(cmd.Context())
			return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
		},
	}
}

func newResourceCommand(opts *options, resource apiclient.Resource) *cobra.Command {
	command := &cobra.Command{
		Use:   string(resource),
		Short: "Manage " + resource.Label(),
		Long: fmt.Sprintf(`Manage %s through the HLIMS API.

All responses are JSON. Create and update accept a JSON document from stdin or
--file. Public IDs from create/list responses are used by get/update/delete.
Run "hlims openapi" for the authoritative request and response schemas.`, resource.Label()),
	}
	command.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List " + resource.Label(),
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				client, err := opts.client()
				if err != nil {
					return err
				}
				response, err := client.List(cmd.Context(), resource)
				return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
			},
		},
		&cobra.Command{
			Use:   "get PUBLIC_ID",
			Short: "Get one " + resource.SingularLabel(),
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				client, err := opts.client()
				if err != nil {
					return err
				}
				response, err := client.Get(cmd.Context(), resource, args[0])
				return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
			},
		},
		newWriteCommand(opts, resource, false),
		newWriteCommand(opts, resource, true),
		newDeleteCommand(opts, resource),
	)
	return command
}

func newWriteCommand(opts *options, resource apiclient.Resource, update bool) *cobra.Command {
	verb := "create"
	use := "create"
	short := "Create a " + resource.SingularLabel()
	if update {
		verb = "update"
		use = "update PUBLIC_ID"
		short = "Replace one " + resource.SingularLabel()
	}
	var filename string
	command := &cobra.Command{
		Use:   use,
		Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if update {
				return cobra.ExactArgs(1)(cmd, args)
			}
			return cobra.NoArgs(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readJSONInput(cmd.InOrStdin(), filename)
			if err != nil {
				return err
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			var response *apiclient.Response
			if update {
				response, err = client.Update(cmd.Context(), resource, args[0], body)
			} else {
				response, err = client.Create(cmd.Context(), resource, body)
			}
			return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
		},
	}
	command.Flags().StringVarP(&filename, "file", "f", "-", "JSON request file, or - for stdin")
	publicID := ""
	if update {
		publicID = " PUBLIC_ID"
	}
	command.Example = "  hlims " + string(resource) + " " + verb + publicID + " --file record.json\n" +
		"  printf '%s\\n' '" + createExample(resource) + "' | hlims " + string(resource) + " " + verb + publicID
	return command
}

func newDeleteCommand(opts *options, resource apiclient.Resource) *cobra.Command {
	var confirmed bool
	command := &cobra.Command{
		Use:   "delete PUBLIC_ID",
		Short: "Delete one " + resource.SingularLabel(),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				return errors.New("refusing to delete without --yes")
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			response, err := client.Delete(cmd.Context(), resource, args[0])
			return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
		},
	}
	command.Flags().BoolVar(&confirmed, "yes", false, "confirm deletion")
	return command
}

func newResolveCommand(opts *options) *cobra.Command {
	resolve := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve an inventory destination without opening it",
		Long:  "Resolve an inventory destination and emit a JSON object containing its URL and selected network.",
	}
	var instanceVia string
	instance := &cobra.Command{
		Use:   "instance MACHINE SERVICE INSTANCE",
		Short: "Resolve a canonical service instance",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			response, err := client.ResolveInstance(cmd.Context(), args[0], args[1], args[2], instanceVia)
			return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
		},
	}
	instance.Flags().StringVar(&instanceVia, "via", "", "network kind: lan or tailnet")
	instance.Example = "  hlims resolve instance badger grafana production\n  hlims resolve instance badger grafana production --via tailnet"

	var portVia, scheme string
	port := &cobra.Command{
		Use:   "port MACHINE PORT",
		Short: "Resolve an ad hoc machine port",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			response, err := client.ResolvePort(cmd.Context(), args[0], value, portVia, scheme)
			return writeResponse(cmd.OutOrStdout(), response, err, opts.compact)
		},
	}
	port.Flags().StringVar(&portVia, "via", "", "network kind: lan or tailnet")
	port.Flags().StringVar(&scheme, "scheme", "", "destination scheme: http or https")
	port.Example = "  hlims resolve port badger 5173\n  hlims resolve port badger 5173 --via tailnet --scheme https"
	resolve.AddCommand(instance, port)
	return resolve
}

func newConsoleCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "console",
		Short: "Open the interactive terminal console",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			return tui.Run(cmd.Context(), client, cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
}

func newOpenAPICommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "openapi",
		Short: "Print the server's OpenAPI document",
		Long:  "Fetch the live OpenAPI YAML document for schema discovery, client generation, or agent tooling.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := opts.client()
			if err != nil {
				return err
			}
			response, err := client.OpenAPISpec(cmd.Context())
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(response.Body)
			return err
		},
	}
}

func (o *options) client() (*apiclient.Client, error) {
	httpClient := o.deps.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: o.timeout}
	}
	return apiclient.New(o.apiURL, httpClient)
}

func (o *options) openURL(ctx context.Context, target string) error {
	if o.deps.OpenURL != nil {
		return o.deps.OpenURL(ctx, target)
	}
	return openBrowser(ctx, target)
}

func readJSONInput(stdin io.Reader, filename string) ([]byte, error) {
	if filename == "-" {
		return readJSON(stdin)
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open JSON request: %w", err)
	}
	body, readErr := readJSON(file)
	if readErr != nil {
		_ = file.Close()
		return nil, readErr
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close JSON request: %w", err)
	}
	return body, nil
}

func readJSON(source io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(source, maxInputSize+1))
	if err != nil {
		return nil, fmt.Errorf("read JSON request: %w", err)
	}
	if len(body) > maxInputSize {
		return nil, errors.New("JSON request exceeds 16 MiB")
	}
	return validateJSON(body)
}

func validateJSON(body []byte) ([]byte, error) {
	if !json.Valid(body) {
		return nil, errors.New("request body is not valid JSON")
	}
	return body, nil
}

func createExample(resource apiclient.Resource) string {
	switch resource {
	case apiclient.MachineProviders:
		return `{"name":"Homelab"}`
	case apiclient.Areas:
		return `{"machineProviderPublicId":"homeprovider","name":"Primary Home"}`
	case apiclient.Manufacturers:
		return `{"name":"Framework"}`
	case apiclient.Products:
		return `{"manufacturerPublicId":"framework8n4q","name":"Desktop Computer","kind":"system"}`
	case apiclient.Assets:
		return `{"productPublicId":"desktop8k2q5","placement":{"type":"area","areaPublicId":"home4q8m2v7k"},"name":"Badger chassis"}`
	case apiclient.Purchases:
		return `{"assetPublicIds":["badger8m2k5q"],"totalPriceCents":3500,"currency":"USD","source":"Facebook Marketplace"}`
	case apiclient.Machines:
		return `{"machineProviderPublicId":"homeprovider","areaPublicId":"home4q8m2v7k","name":"Badger","kind":"bare_metal"}`
	case apiclient.MachineUsers:
		return `{"machinePublicId":"badmach8k2q5n","username":"tyler","isPreferred":true}`
	case apiclient.Networks:
		return `{"name":"Home LAN","kind":"lan","cidr":"192.168.68.0/24"}`
	case apiclient.Addresses:
		return `{"networkPublicId":"homelan7k2p9","machinePublicId":"badmach8k2q5","address":"192.168.68.60"}`
	case apiclient.Services:
		return `{"name":"Grafana"}`
	case apiclient.Instances:
		return `{"servicePublicId":"grafana5k2mx","machinePublicId":"badmach8k2q5n","name":"Production","port":3000}`
	case apiclient.InstanceEndpoints:
		return `{"instancePublicId":"grafprd8n4qx","addressPublicId":"badts4n8p2km","name":"Tailnet","scheme":"https","port":443}`
	default:
		return `{}`
	}
}

func writeResponse(output io.Writer, response *apiclient.Response, requestErr error, compact bool) error {
	if requestErr != nil {
		return requestErr
	}
	if len(response.Body) == 0 {
		return nil
	}
	if !json.Valid(response.Body) {
		return errors.New("HLIMS API returned a non-JSON success response")
	}
	var formatted bytes.Buffer
	format := json.Indent
	if compact {
		format = func(destination *bytes.Buffer, source []byte, _, _ string) error {
			return json.Compact(destination, source)
		}
	}
	if err := format(&formatted, response.Body, "", "  "); err != nil {
		return fmt.Errorf("format HLIMS API response: %w", err)
	}
	formatted.WriteByte('\n')
	_, err := output.Write(formatted.Bytes())
	return err
}

// Execute runs the hlims command with process-standard dependencies.
func Execute(ctx context.Context) error {
	command := NewRootCommand(Dependencies{})
	command.SetContext(ctx)
	return command.Execute()
}
