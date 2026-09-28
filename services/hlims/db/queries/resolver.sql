-- name: ListMachines :many
select
    public_id,
    name,
    slug,
    hostname,
    kind,
    is_favorite
from machines
order by name;

-- name: GetPreferredInstanceEndpoint :one
select
    addresses.address,
    networks.kind as network_kind,
    instance_endpoints.scheme,
    instance_endpoints.port,
    instance_endpoints.host_type,
    instance_endpoints.base_path,
    coalesce(
        case
            when dns_records.name = '@' then dns_zones.name
            else dns_records.name || '.' || dns_zones.name
        end,
        addresses.dns_name
    )             as dns_name
from instances
inner join services on instances.service_id = services.id
inner join machines on instances.machine_id = machines.id
inner join instance_endpoints on instances.id = instance_endpoints.instance_id
inner join addresses on instance_endpoints.address_id = addresses.id
left join dns_records on instance_endpoints.dns_record_id = dns_records.id
left join dns_zones on dns_records.zone_id = dns_zones.id
inner join networks on addresses.network_id = networks.id
where
    machines.slug = @machine_slug
    and services.slug = @service_slug
    and instances.slug = @instance_slug
order by instance_endpoints.is_preferred desc, instance_endpoints.name asc
limit 1;

-- name: GetInstanceEndpointByNetworkKind :one
select
    addresses.address,
    networks.kind as network_kind,
    instance_endpoints.scheme,
    instance_endpoints.port,
    instance_endpoints.host_type,
    instance_endpoints.base_path,
    coalesce(
        case
            when dns_records.name = '@' then dns_zones.name
            else dns_records.name || '.' || dns_zones.name
        end,
        addresses.dns_name
    )             as dns_name
from instances
inner join services on instances.service_id = services.id
inner join machines on instances.machine_id = machines.id
inner join instance_endpoints on instances.id = instance_endpoints.instance_id
inner join addresses on instance_endpoints.address_id = addresses.id
left join dns_records on instance_endpoints.dns_record_id = dns_records.id
left join dns_zones on dns_records.zone_id = dns_zones.id
inner join networks on addresses.network_id = networks.id
where
    machines.slug = @machine_slug
    and services.slug = @service_slug
    and instances.slug = @instance_slug
    and networks.kind = @network_kind
order by instance_endpoints.is_preferred desc, instance_endpoints.name asc
limit 1;

-- name: GetPreferredMachineAddress :one
select
    addresses.address,
    addresses.dns_name,
    networks.kind as network_kind
from addresses
inner join networks on addresses.network_id = networks.id
inner join machines on addresses.machine_id = machines.id
where
    machines.slug = @machine_slug
    and networks.kind in ('tailnet', 'lan')
order by
    case networks.kind when 'tailnet' then 0 else 1 end asc,
    addresses.is_primary desc,
    addresses.name asc
limit 1;

-- name: GetMachineAddressByNetworkKind :one
select
    addresses.address,
    addresses.dns_name,
    networks.kind as network_kind
from addresses
inner join networks on addresses.network_id = networks.id
inner join machines on addresses.machine_id = machines.id
where
    machines.slug = @machine_slug
    and networks.kind = @network_kind
order by addresses.is_primary desc, addresses.name asc
limit 1;
