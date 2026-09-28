-- name: CreateService :one
insert into services (id, public_id, name, slug, description)
values (?, ?, ?, ?, ?)
returning id, public_id, name, slug, description, created_at, updated_at;

-- name: GetServiceByPublicID :one
select
    id,
    public_id,
    name,
    slug,
    description,
    created_at,
    updated_at
from services
where public_id = ?;

-- name: ListServices :many
select
    id,
    public_id,
    name,
    slug,
    description,
    created_at,
    updated_at,
    exists(
        select 1 as result
        from service_logos
        where service_logos.service_id = services.id
    ) as has_logo
from services
order by name;

-- name: UpdateService :one
update services
set
    name = ?,
    slug = ?,
    description = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning id, public_id, name, slug, description, created_at, updated_at;

-- name: DeleteService :execrows
delete from services
where public_id = ?;

-- name: ServiceHasLogo :one
select exists(
    select 1 as result
    from service_logos
    inner join services on service_logos.service_id = services.id
    where services.public_id = ?
) as has_logo;

-- name: UpsertServiceLogo :exec
insert into service_logos (service_id, content_type, image_data)
select
    services.id,
    sqlc.arg(content_type) as content_type,
    sqlc.arg(image_data)   as image_data
from services
where services.public_id = sqlc.arg(public_id)
on conflict (service_id) do update set
    content_type = excluded.content_type,
    image_data = excluded.image_data,
    updated_at = strftime('%s', 'now');

-- name: GetServiceLogo :one
select
    service_logos.content_type,
    service_logos.image_data,
    service_logos.updated_at
from service_logos
inner join services on service_logos.service_id = services.id
where services.public_id = ?;

-- name: DeleteServiceLogo :execrows
delete from service_logos
where service_id = (
    select services.id
    from services
    where services.public_id = ?
);

-- name: CreateInstance :one
insert into instances (
    id,
    public_id,
    service_id,
    machine_id,
    name,
    slug,
    port,
    notes
)
values (?, ?, ?, ?, ?, ?, ?, ?)
returning
    id,
    public_id,
    service_id,
    machine_id,
    name,
    slug,
    port,
    notes,
    created_at,
    updated_at;

-- name: GetInstanceByPublicID :one
select
    id,
    public_id,
    service_id,
    machine_id,
    name,
    slug,
    port,
    notes,
    created_at,
    updated_at
from instances
where public_id = ?;

-- name: GetInstanceDetailByPublicID :one
select
    instances.public_id,
    instances.name,
    instances.slug,
    instances.port,
    instances.notes,
    instances.created_at,
    instances.updated_at,
    services.public_id as service_public_id,
    machines.public_id as machine_public_id
from instances
inner join services on instances.service_id = services.id
inner join machines on instances.machine_id = machines.id
where instances.public_id = ?;

-- name: ListInstances :many
select
    instances.public_id,
    instances.name,
    instances.slug,
    instances.port,
    instances.notes,
    instances.created_at,
    instances.updated_at,
    services.public_id as service_public_id,
    machines.public_id as machine_public_id,
    exists(
        select 1 as result
        from instance_endpoints
        inner join addresses on instance_endpoints.address_id = addresses.id
        inner join networks on addresses.network_id = networks.id
        where
            instance_endpoints.instance_id = instances.id
            and networks.kind = 'lan'
    )                  as has_lan_route,
    exists(
        select 1 as result
        from instance_endpoints
        inner join addresses on instance_endpoints.address_id = addresses.id
        inner join networks on addresses.network_id = networks.id
        where
            instance_endpoints.instance_id = instances.id
            and networks.kind = 'tailnet'
    )                  as has_tailnet_route
from instances
inner join services on instances.service_id = services.id
inner join machines on instances.machine_id = machines.id
order by machines.name, services.name, instances.name;

-- name: UpdateInstance :one
update instances
set
    service_id = ?,
    machine_id = ?,
    name = ?,
    slug = ?,
    port = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning
    id,
    public_id,
    service_id,
    machine_id,
    name,
    slug,
    port,
    notes,
    created_at,
    updated_at;

-- name: DeleteInstance :execrows
delete from instances
where public_id = ?;

-- name: CreateInstanceEndpoint :one
insert into instance_endpoints (
    id,
    public_id,
    instance_id,
    address_id,
    dns_record_id,
    name,
    scheme,
    port,
    base_path,
    host_type,
    is_preferred,
    notes
)
values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
returning
    id,
    public_id,
    instance_id,
    address_id,
    dns_record_id,
    name,
    scheme,
    port,
    base_path,
    host_type,
    is_preferred,
    notes,
    created_at,
    updated_at;

-- name: GetInstanceEndpointByPublicID :one
select
    instance_endpoints.public_id,
    instance_endpoints.name,
    instance_endpoints.scheme,
    instance_endpoints.port,
    instance_endpoints.base_path,
    instance_endpoints.host_type,
    instance_endpoints.is_preferred,
    instance_endpoints.notes,
    instance_endpoints.created_at,
    instance_endpoints.updated_at,
    instances.public_id   as instance_public_id,
    addresses.public_id   as address_public_id,
    dns_records.public_id as dns_record_public_id
from instance_endpoints
inner join instances on instance_endpoints.instance_id = instances.id
inner join addresses on instance_endpoints.address_id = addresses.id
left join dns_records on instance_endpoints.dns_record_id = dns_records.id
where instance_endpoints.public_id = ?;

-- name: ListInstanceEndpoints :many
select
    instance_endpoints.public_id,
    instance_endpoints.name,
    instance_endpoints.scheme,
    instance_endpoints.port,
    instance_endpoints.base_path,
    instance_endpoints.host_type,
    instance_endpoints.is_preferred,
    instance_endpoints.notes,
    instance_endpoints.created_at,
    instance_endpoints.updated_at,
    instances.public_id   as instance_public_id,
    addresses.public_id   as address_public_id,
    dns_records.public_id as dns_record_public_id
from instance_endpoints
inner join instances on instance_endpoints.instance_id = instances.id
inner join addresses on instance_endpoints.address_id = addresses.id
left join dns_records on instance_endpoints.dns_record_id = dns_records.id
order by instances.name, instance_endpoints.name;

-- name: ListTopologyInstanceEndpoints :many
select
    instance_endpoints.public_id,
    instances.public_id   as instance_public_id,
    instance_endpoints.name,
    instance_endpoints.scheme,
    instance_endpoints.port,
    instance_endpoints.base_path,
    instance_endpoints.host_type,
    instance_endpoints.is_preferred,
    addresses.address,
    dns_records.public_id as dns_record_public_id,
    networks.kind         as network_kind,
    coalesce(
        case
            when dns_records.name = '@' then dns_zones.name
            else dns_records.name || '.' || dns_zones.name
        end,
        addresses.dns_name
    )                     as dns_name
from instance_endpoints
inner join instances on instance_endpoints.instance_id = instances.id
inner join addresses on instance_endpoints.address_id = addresses.id
left join dns_records on instance_endpoints.dns_record_id = dns_records.id
left join dns_zones on dns_records.zone_id = dns_zones.id
inner join networks on addresses.network_id = networks.id
order by instances.public_id asc, instance_endpoints.is_preferred desc, instance_endpoints.name asc;

-- name: UpdateInstanceEndpoint :one
update instance_endpoints
set
    instance_id = ?,
    address_id = ?,
    dns_record_id = ?,
    name = ?,
    scheme = ?,
    port = ?,
    base_path = ?,
    host_type = ?,
    is_preferred = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning
    id,
    public_id,
    instance_id,
    address_id,
    dns_record_id,
    name,
    scheme,
    port,
    base_path,
    host_type,
    is_preferred,
    notes,
    created_at,
    updated_at;

-- name: ClearPreferredInstanceEndpoints :exec
update instance_endpoints
set is_preferred = 0, updated_at = strftime('%s', 'now')
where instance_id = ? and public_id != ?;

-- name: ClearAllPreferredInstanceEndpoints :exec
update instance_endpoints
set is_preferred = 0, updated_at = strftime('%s', 'now')
where instance_id = ?;

-- name: DeleteInstanceEndpoint :execrows
delete from instance_endpoints
where public_id = ?;

-- name: ListInstancesByMachineSlug :many
select
    instances.public_id,
    instances.name,
    instances.slug,
    instances.port,
    services.public_id as service_public_id,
    services.name      as service_name,
    services.slug      as service_slug
from instances
inner join services on instances.service_id = services.id
inner join machines on instances.machine_id = machines.id
where machines.slug = ?
order by services.name, instances.name;
