package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
	"github.com/spf13/cobra"
)

const maxLogoBytes = 1 << 20

func newLogoCommand(opts *options, resource apiclient.Resource) *cobra.Command {
	logo := &cobra.Command{Use: "logo", Short: "Manage a PNG, JPEG, or WebP logo"}
	var filename string
	set := &cobra.Command{
		Use:   "set ID_OR_SLUG",
		Short: "Upload a verified logo (maximum 1 MiB)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return errors.New("--file is required (use - for stdin)")
			}
			body, err := readLogoInput(cmd.InOrStdin(), filename)
			if err != nil {
				return err
			}
			contentType := http.DetectContentType(body)
			if contentType != "image/png" && contentType != "image/jpeg" && contentType != "image/webp" {
				return fmt.Errorf("unsupported logo image type %q; use PNG, JPEG, or WebP", contentType)
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			id, err := client.ResolvePublicID(cmd.Context(), resource, args[0], apiclient.ListFilter{})
			if err != nil {
				return err
			}
			if resource == apiclient.Services {
				_, err = client.UpdateServiceLogo(cmd.Context(), id, contentType, body)
			} else {
				_, err = client.UpdateMachineProviderLogo(cmd.Context(), id, contentType, body)
			}
			if err != nil {
				return err
			}
			var stored *apiclient.Response
			if resource == apiclient.Services {
				stored, err = client.GetServiceLogo(cmd.Context(), id)
			} else {
				stored, err = client.GetMachineProviderLogo(cmd.Context(), id)
			}
			if err != nil {
				return err
			}
			if !bytes.Equal(stored.Body, body) {
				return errors.New("uploaded logo does not match the stored logo")
			}
			record, err := client.Get(cmd.Context(), resource, id)
			return writeResponse(cmd.OutOrStdout(), record, err, opts.compact)
		},
	}
	set.Flags().StringVarP(&filename, "file", "f", "", "local image file, or - for stdin")
	var confirmed bool
	remove := &cobra.Command{
		Use:   "delete ID_OR_SLUG",
		Short: "Delete an uploaded logo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				return errors.New("refusing to delete without --yes")
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			id, err := client.ResolvePublicID(cmd.Context(), resource, args[0], apiclient.ListFilter{})
			if err != nil {
				return err
			}
			if resource == apiclient.Services {
				_, err = client.DeleteServiceLogo(cmd.Context(), id)
			} else {
				_, err = client.DeleteMachineProviderLogo(cmd.Context(), id)
			}
			if err != nil {
				return err
			}
			record, err := client.Get(cmd.Context(), resource, id)
			return writeResponse(cmd.OutOrStdout(), record, err, opts.compact)
		},
	}
	remove.Flags().BoolVar(&confirmed, "yes", false, "confirm logo deletion")
	logo.AddCommand(set, remove)
	return logo
}

func readLogoInput(stdin io.Reader, filename string) ([]byte, error) {
	source := stdin
	if filename != "-" {
		file, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("open logo file: %w", err)
		}
		defer func() { _ = file.Close() }()
		source = file
	}
	data, err := io.ReadAll(io.LimitReader(source, maxLogoBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read logo: %w", err)
	}
	if len(data) == 0 || len(data) > maxLogoBytes {
		return nil, errors.New("logo must contain between 1 byte and 1 MiB")
	}
	return data, nil
}
