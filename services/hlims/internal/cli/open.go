package cli

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/TylerHillery/homelab/services/hlims/internal/apiclient"
	"github.com/spf13/cobra"
)

type openTarget struct {
	machine  string
	service  string
	instance string
	host     string
	port     int
	via      string
	scheme   string
	suffix   []string
	query    url.Values
	fragment string
}

func newOpenCommand(opts *options) *cobra.Command {
	var printOnly bool
	command := &cobra.Command{
		Use:   "open PATH",
		Short: "Resolve a go-style path and open it in the default browser",
		Long: `Resolve a canonical instance or ad hoc machine-port path through the
HLIMS API, then open the resulting destination in the local default browser.
Additional path segments, query parameters, and fragments are preserved.

Use --print for scripts, testing, or environments without a browser.`,
		Example: `  hlims open grafana/production
  hlims open grafana/production?host=badger
  hlims open go/badger/5173?via=tailnet
  hlims open 'grafana/production/d/overview?refresh=30s'
  hlims open --print opencode/production`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := parseOpenTarget(args[0])
			if err != nil {
				return err
			}
			client, err := opts.client()
			if err != nil {
				return err
			}
			var response *apiclient.Response
			if target.port != 0 {
				response, err = client.ResolvePort(cmd.Context(), target.machine, target.port, target.via, target.scheme)
			} else {
				response, err = client.ResolveInstance(cmd.Context(), target.service, target.instance, target.host, target.via)
			}
			if err != nil {
				return err
			}
			destination, err := apiclient.DestinationURL(response)
			if err != nil {
				return err
			}
			destination, err = appendOpenTarget(destination, target)
			if err != nil {
				return err
			}
			if printOnly {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), destination)
				return err
			}
			return opts.openURL(cmd.Context(), destination)
		},
	}
	command.Flags().BoolVarP(&printOnly, "print", "n", false, "print the destination without opening a browser")
	return command
}

func parseOpenTarget(raw string) (openTarget, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return openTarget{}, fmt.Errorf("parse path: %w", err)
	}
	if parsed.Opaque != "" {
		return openTarget{}, errors.New("path must be a go-style HTTP path")
	}
	if parsed.IsAbs() {
		if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return openTarget{}, errors.New("full go URL must use http or https and include a host")
		}
	} else if parsed.Host != "" || strings.HasPrefix(raw, "//") {
		return openTarget{}, errors.New("scheme-relative URLs are not supported")
	}

	escapedSegments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	segments := make([]string, len(escapedSegments))
	for index, escaped := range escapedSegments {
		segment, err := url.PathUnescape(escaped)
		if err != nil {
			return openTarget{}, fmt.Errorf("decode path segment: %w", err)
		}
		segments[index] = segment
	}
	if len(segments) > 0 && segments[0] == "go" {
		segments = segments[1:]
		escapedSegments = escapedSegments[1:]
	}
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return openTarget{}, errors.New("path contains an empty or invalid segment")
		}
	}
	if len(segments) < 2 {
		return openTarget{}, errors.New("path must be SERVICE/INSTANCE or MACHINE/PORT")
	}

	query := parsed.Query()
	target := openTarget{
		machine:  segments[0],
		host:     query.Get("host"),
		via:      query.Get("via"),
		query:    query,
		fragment: parsed.Fragment,
	}
	target.query.Del("via")
	target.query.Del("host")

	// This intentionally mirrors the browser resolver: a numeric second segment
	// selects an ad hoc port, even when additional path segments follow.
	port, portErr := strconv.Atoi(segments[1])
	if portErr == nil {
		if target.host != "" {
			return openTarget{}, errors.New("ad hoc machine ports do not use a host selector")
		}
		if port < 1 || port > 65535 {
			return openTarget{}, errors.New("port must be between 1 and 65535")
		}
		target.port = port
		target.scheme = target.query.Get("scheme")
		target.query.Del("scheme")
		target.suffix = escapedSegments[2:]
		return target, nil
	}
	target.service = segments[0]
	target.instance = segments[1]
	target.suffix = escapedSegments[2:]
	return target, nil
}

func appendOpenTarget(destination string, target openTarget) (string, error) {
	parsed, err := url.Parse(destination)
	if err != nil {
		return "", fmt.Errorf("parse resolved destination: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("resolved destination must use http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("resolved destination must include a host")
	}
	if len(target.suffix) > 0 {
		escapedPath := strings.TrimRight(parsed.EscapedPath(), "/") + "/" + strings.Join(target.suffix, "/")
		decodedPath, err := url.PathUnescape(escapedPath)
		if err != nil {
			return "", fmt.Errorf("decode forwarded path: %w", err)
		}
		parsed.Path = decodedPath
		parsed.RawPath = escapedPath
	}
	query := parsed.Query()
	for key, values := range target.query {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	if target.fragment != "" {
		parsed.Fragment = target.fragment
	}
	return parsed.String(), nil
}
