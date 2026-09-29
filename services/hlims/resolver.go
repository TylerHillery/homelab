package hlims

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	database "github.com/TylerHillery/homelab/services/hlims/generated/db"
)

var (
	errDestinationNotFound = errors.New("destination not found")
	errInvalidPort         = errors.New("port must be between 1 and 65535")
	errInvalidScheme       = errors.New("scheme must be http or https")
	errInvalidVia          = errors.New("via must be lan or tailnet")
	errAmbiguousInstance   = errors.New("multiple instances match; specify host=<machine-slug> or host=managed")
)

type resolvedDestination struct {
	URL string
	Via string
}

type endpoint struct {
	address     string
	dnsName     sql.NullString
	hostType    string
	networkKind string
	scheme      string
	port        int64
	basePath    string
}

type resolver struct {
	queries *database.Queries
}

func (r resolver) instance(
	ctx context.Context,
	service, instance, host, via, remainder string,
	query url.Values,
) (resolvedDestination, error) {
	rows, err := r.queries.ListInstanceHostsByServiceAndSlug(ctx, database.ListInstanceHostsByServiceAndSlugParams{
		ServiceSlug: service, InstanceSlug: instance,
	})
	if err != nil {
		return resolvedDestination{}, err
	}
	if len(rows) == 0 {
		return resolvedDestination{}, errDestinationNotFound
	}
	var selected *database.ListInstanceHostsByServiceAndSlugRow
	if host == "" {
		if len(rows) != 1 {
			return resolvedDestination{}, errAmbiguousInstance
		}
		selected = &rows[0]
	} else {
		for index := range rows {
			row := &rows[index]
			if row.HostingKind == "managed" && host == "managed" || row.HostingKind == "machine" && row.MachineSlug.String == host {
				selected = row
				break
			}
		}
		if selected == nil {
			return resolvedDestination{}, errDestinationNotFound
		}
	}
	if selected.HostingKind == "managed" {
		return r.managed(ctx, service, instance, via, remainder, query)
	}
	machine := selected.MachineSlug.String
	var target endpoint
	if via == "" {
		row, err := r.queries.GetPreferredInstanceEndpoint(ctx, database.GetPreferredInstanceEndpointParams{
			MachineSlug:  machine,
			ServiceSlug:  service,
			InstanceSlug: instance,
		})
		if err != nil {
			return resolvedDestination{}, resolveQueryError(err)
		}
		target = endpoint{
			address:     row.Address,
			dnsName:     row.DnsName,
			hostType:    row.HostType,
			networkKind: row.NetworkKind,
			scheme:      row.Scheme.String,
			port:        row.Port.Int64,
			basePath:    row.BasePath,
		}
	} else {
		if !validVia(via) {
			return resolvedDestination{}, errInvalidVia
		}
		row, err := r.queries.GetInstanceEndpointByNetworkKind(ctx, database.GetInstanceEndpointByNetworkKindParams{
			MachineSlug:  machine,
			ServiceSlug:  service,
			InstanceSlug: instance,
			NetworkKind:  via,
		})
		if err != nil {
			return resolvedDestination{}, resolveQueryError(err)
		}
		target = endpoint{
			address:     row.Address,
			dnsName:     row.DnsName,
			hostType:    row.HostType,
			networkKind: row.NetworkKind,
			scheme:      row.Scheme.String,
			port:        row.Port.Int64,
			basePath:    row.BasePath,
		}
	}

	destination, err := endpointURL(target, remainder, serviceResolverQuery(query))
	if err != nil {
		return resolvedDestination{}, err
	}
	return resolvedDestination{URL: destination, Via: target.networkKind}, nil
}

func (r resolver) managed(ctx context.Context, service, instance, via, remainder string, query url.Values) (resolvedDestination, error) {
	if via != "" {
		return resolvedDestination{}, errInvalidVia
	}
	direct, err := r.queries.GetManagedInstanceEndpoint(ctx, database.GetManagedInstanceEndpointParams{ServiceSlug: service, InstanceSlug: instance})
	if err != nil {
		return resolvedDestination{}, resolveQueryError(err)
	}
	if !direct.Valid {
		return resolvedDestination{}, errors.New("managed endpoint has no URL")
	}
	destination, err := url.Parse(direct.String)
	if err != nil || destination.Scheme != "https" || destination.Hostname() == "" {
		return resolvedDestination{}, errors.New("managed endpoint has an invalid URL")
	}
	if remainder != "" {
		destination.Path = joinURLPath(destination.Path, remainder)
	}
	parameters := destination.Query()
	for key, values := range serviceResolverQuery(query) {
		parameters[key] = values
	}
	destination.RawQuery = parameters.Encode()
	return resolvedDestination{URL: destination.String(), Via: "managed"}, nil
}

func (r resolver) machinePort(
	ctx context.Context,
	machine string,
	port int,
	via, scheme, remainder string,
	query url.Values,
) (resolvedDestination, error) {
	if port < 1 || port > 65535 {
		return resolvedDestination{}, errInvalidPort
	}
	if scheme == "" {
		scheme = string(InstanceSchemeHTTP)
	}
	if !InstanceScheme(scheme).Valid() {
		return resolvedDestination{}, errInvalidScheme
	}

	var target endpoint
	if via == "" {
		row, err := r.queries.GetPreferredMachineAddress(ctx, machine)
		if err != nil {
			return resolvedDestination{}, resolveQueryError(err)
		}
		target = endpoint{
			address:     row.Address,
			dnsName:     row.DnsName,
			networkKind: row.NetworkKind,
			scheme:      scheme,
			port:        int64(port),
		}
	} else {
		if !validVia(via) {
			return resolvedDestination{}, errInvalidVia
		}
		row, err := r.queries.GetMachineAddressByNetworkKind(ctx, database.GetMachineAddressByNetworkKindParams{
			MachineSlug: machine,
			NetworkKind: via,
		})
		if err != nil {
			return resolvedDestination{}, resolveQueryError(err)
		}
		target = endpoint{
			address:     row.Address,
			dnsName:     row.DnsName,
			networkKind: row.NetworkKind,
			scheme:      scheme,
			port:        int64(port),
		}
	}

	destination, err := endpointURL(target, remainder, resolverQuery(query))
	if err != nil {
		return resolvedDestination{}, err
	}
	return resolvedDestination{URL: destination, Via: target.networkKind}, nil
}

func endpointURL(target endpoint, remainder string, query url.Values) (string, error) {
	if !InstanceScheme(target.scheme).Valid() {
		return "", errInvalidScheme
	}
	if target.port < 1 || target.port > 65535 {
		return "", errInvalidPort
	}

	host := target.address
	if target.hostType == "dns" && (!target.dnsName.Valid || target.dnsName.String == "") {
		return "", errors.New("DNS endpoint has no DNS name")
	}
	if target.hostType != "ip" && target.dnsName.Valid && target.dnsName.String != "" {
		host = strings.TrimSuffix(target.dnsName.String, ".")
	}
	if !defaultPort(target.scheme, target.port) {
		host = net.JoinHostPort(host, strconv.FormatInt(target.port, 10))
	} else if net.ParseIP(host) != nil && strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	destination := url.URL{
		Scheme:   target.scheme,
		Host:     host,
		Path:     joinURLPath(target.basePath, remainder),
		RawQuery: query.Encode(),
	}
	if destination.Hostname() == "" {
		return "", fmt.Errorf("invalid endpoint address %q", target.address)
	}
	return destination.String(), nil
}

func joinURLPath(base, remainder string) string {
	base = strings.Trim(base, "/")
	remainder = strings.Trim(remainder, "/")
	switch {
	case base == "" && remainder == "":
		return "/"
	case base == "":
		return "/" + remainder
	case remainder == "":
		return "/" + base
	default:
		return "/" + base + "/" + remainder
	}
}

func resolverQuery(query url.Values) url.Values {
	forwarded := make(url.Values, len(query))
	for key, values := range query {
		forwarded[key] = append([]string(nil), values...)
	}
	forwarded.Del("via")
	forwarded.Del("scheme")
	return forwarded
}

func serviceResolverQuery(query url.Values) url.Values {
	forwarded := resolverQuery(query)
	forwarded.Del("host")
	return forwarded
}

func validVia(via string) bool {
	return NetworkKind(via).Valid()
}

func defaultPort(scheme string, port int64) bool {
	return scheme == "http" && port == 80 || scheme == "https" && port == 443
}

func resolveQueryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errDestinationNotFound
	}
	return err
}
