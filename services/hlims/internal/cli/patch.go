package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
	"github.com/spf13/cobra"
)

func newPatchCommand(opts *options, resource apiclient.Resource) *cobra.Command {
	var filename string
	var filter apiclient.ListFilter
	command := &cobra.Command{
		Use:   "patch ID_OR_SLUG",
		Short: "Merge verified JSON fields without replacing the rest of a record",
		Long:  "Read a JSON object from stdin or --file, merge it with the current record, and replace it only if the record has not changed since it was read.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readJSONInput(cmd.InOrStdin(), filename)
			if err != nil {
				return err
			}
			patch, err := decodeObject(body)
			if err != nil || len(patch) == 0 {
				return errors.New("patch must be a nonempty JSON object")
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			id, err := client.ResolvePublicID(cmd.Context(), resource, args[0], filter)
			if err != nil {
				return err
			}
			current, err := client.Get(cmd.Context(), resource, id)
			if err != nil {
				return err
			}
			if current.ETag == "" {
				return errors.New("server did not provide a revision ETag for safe patch")
			}
			value, err := decodeObject(current.Body)
			if err != nil {
				return fmt.Errorf("decode current record: %w", err)
			}
			stripReadOnlyFields(value, resource)
			for key := range patch {
				if !writableField(key, resource) {
					return fmt.Errorf("cannot patch read-only field %q", key)
				}
			}
			mergeFields(value, patch)
			merged, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("encode merged record: %w", err)
			}
			updated, err := client.UpdateIfMatch(cmd.Context(), resource, id, merged, current.ETag)
			return writeResponse(cmd.OutOrStdout(), updated, err, opts.compact)
		},
	}
	command.Flags().StringVarP(&filename, "file", "f", "-", "JSON patch file, or - for stdin")
	if resource == apiclient.Instances {
		command.Flags().StringVar(&filter.Service, "service", "", "narrow an Instance slug by Service slug")
		command.Flags().StringVar(&filter.Machine, "machine", "", "narrow an Instance slug by Machine slug")
	}
	command.Example = "  printf '%s\\n' '{\"notes\":\"Updated\"}' | hlims " + string(resource) + " patch ID_OR_SLUG"
	return command
}

func decodeObject(body []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("expected exactly one JSON object")
	}
	if object == nil {
		return nil, errors.New("JSON value must be an object")
	}
	return object, nil
}

func stripReadOnlyFields(value map[string]any, resource apiclient.Resource) {
	delete(value, "publicId")
	for _, field := range []string{"hasLogo", "effectiveAreaPublicId", "purchasePublicId", "trackedSubtotalCents", "homelabSubtotalCents"} {
		if !writableField(field, resource) {
			delete(value, field)
		}
	}
	// The response includes the default address host mode even for a managed
	// direct URL; only address-backed endpoints accept that write field.
	if resource == apiclient.InstanceEndpoints && value["directUrl"] != nil {
		delete(value, "hostType")
	}
}

func writableField(field string, resource apiclient.Resource) bool {
	switch field {
	case "publicId":
		return false
	case "hasLogo":
		return resource != apiclient.Services && resource != apiclient.MachineProviders
	case "effectiveAreaPublicId", "purchasePublicId":
		return resource != apiclient.Assets
	case "trackedSubtotalCents", "homelabSubtotalCents":
		return resource != apiclient.Purchases
	default:
		return true
	}
}

func mergeFields(dst, patch map[string]any) {
	for key, next := range patch {
		if next == nil {
			delete(dst, key)
			continue
		}
		if nested, ok := next.(map[string]any); ok {
			if existing, ok := dst[key].(map[string]any); ok {
				mergeFields(existing, nested)
				continue
			}
		}
		dst[key] = next
	}
}
