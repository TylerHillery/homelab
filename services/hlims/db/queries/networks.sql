-- name: CreateNetwork :one
insert into networks (id, public_id, area_id, name, slug, kind, cidr, notes)
values (?, ?, ?, ?, ?, ?, ?, ?)
returning id, public_id, area_id, name, slug, kind, cidr, notes, created_at, updated_at;

-- name: GetNetworkByPublicID :one
select
    networks.id,
    networks.public_id,
    networks.area_id,
    networks.name,
    networks.slug,
    networks.kind,
    networks.cidr,
    networks.notes,
    networks.created_at,
    networks.updated_at,
    areas.public_id as area_public_id
from networks
left join areas on networks.area_id = areas.id
where networks.public_id = ?;

-- name: ListNetworks :many
select
    networks.id,
    networks.public_id,
    networks.area_id,
    networks.name,
    networks.slug,
    networks.kind,
    networks.cidr,
    networks.notes,
    networks.created_at,
    networks.updated_at,
    areas.public_id as area_public_id
from networks
left join areas on networks.area_id = areas.id
order by networks.name;

-- name: UpdateNetwork :one
update networks
set
    area_id = ?,
    name = ?,
    slug = ?,
    kind = ?,
    cidr = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning id, public_id, area_id, name, slug, kind, cidr, notes, created_at, updated_at;

-- name: DeleteNetwork :execrows
delete from networks
where public_id = ?;

-- name: CreateAddress :one
insert into addresses (
    id,
    public_id,
    network_id,
    machine_id,
    area_id,
    name,
    address,
    dns_name,
    interface_name,
    is_primary
)
values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
returning
    id,
    public_id,
    network_id,
    machine_id,
    area_id,
    name,
    address,
    dns_name,
    interface_name,
    is_primary,
    created_at,
    updated_at;

-- name: GetAddressByPublicID :one
select
    addresses.id,
    addresses.public_id,
    addresses.network_id,
    addresses.machine_id,
    addresses.area_id,
    addresses.name,
    addresses.address,
    addresses.dns_name,
    addresses.interface_name,
    addresses.is_primary,
    addresses.created_at,
    addresses.updated_at,
    networks.public_id as network_public_id,
    machines.public_id as machine_public_id,
    areas.public_id    as area_public_id
from addresses
inner join networks on addresses.network_id = networks.id
left join machines on addresses.machine_id = machines.id
left join areas on addresses.area_id = areas.id
where addresses.public_id = ?;

-- name: ListAddresses :many
select
    addresses.id,
    addresses.public_id,
    addresses.network_id,
    addresses.machine_id,
    addresses.area_id,
    addresses.name,
    addresses.address,
    addresses.dns_name,
    addresses.interface_name,
    addresses.is_primary,
    addresses.created_at,
    addresses.updated_at,
    networks.public_id as network_public_id,
    machines.public_id as machine_public_id,
    areas.public_id    as area_public_id
from addresses
inner join networks on addresses.network_id = networks.id
left join machines on addresses.machine_id = machines.id
left join areas on addresses.area_id = areas.id
order by networks.name, addresses.address;

-- name: UpdateAddress :one
update addresses
set
    network_id = ?,
    machine_id = ?,
    area_id = ?,
    name = ?,
    address = ?,
    dns_name = ?,
    interface_name = ?,
    is_primary = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning
    id,
    public_id,
    network_id,
    machine_id,
    area_id,
    name,
    address,
    dns_name,
    interface_name,
    is_primary,
    created_at,
    updated_at;

-- name: DeleteAddress :execrows
delete from addresses
where public_id = ?;
