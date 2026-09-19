-- name: CreateMachineUser :one
insert into machine_users (
    id,
    public_id,
    machine_id,
    username,
    is_preferred,
    notes
)
values (?, ?, ?, ?, ?, ?)
returning id, public_id, machine_id, username, is_preferred, notes, created_at, updated_at;

-- name: GetMachineUserByPublicID :one
select
    machine_users.id,
    machine_users.public_id,
    machine_users.machine_id,
    machine_users.username,
    machine_users.is_preferred,
    machine_users.notes,
    machine_users.created_at,
    machine_users.updated_at,
    machines.public_id as machine_public_id
from machine_users
inner join machines on machine_users.machine_id = machines.id
where machine_users.public_id = ?;

-- name: ListMachineUsers :many
select
    machine_users.id,
    machine_users.public_id,
    machine_users.machine_id,
    machine_users.username,
    machine_users.is_preferred,
    machine_users.notes,
    machine_users.created_at,
    machine_users.updated_at,
    machines.public_id as machine_public_id
from machine_users
inner join machines on machine_users.machine_id = machines.id
order by machines.name, machine_users.username, machine_users.public_id;

-- name: UpdateMachineUser :one
update machine_users
set
    machine_id = ?,
    username = ?,
    is_preferred = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning id, public_id, machine_id, username, is_preferred, notes, created_at, updated_at;

-- name: ClearPreferredMachineUsers :exec
update machine_users
set is_preferred = 0, updated_at = strftime('%s', 'now')
where machine_id = ? and public_id != ?;

-- name: ClearAllPreferredMachineUsers :exec
update machine_users
set is_preferred = 0, updated_at = strftime('%s', 'now')
where machine_id = ?;

-- name: DeleteMachineUser :execrows
delete from machine_users
where public_id = ?;

-- name: ListTopologyMachineAddresses :many
select
    addresses.public_id,
    addresses.machine_id,
    addresses.name,
    addresses.address,
    addresses.dns_name,
    addresses.interface_name,
    addresses.is_primary,
    machines.public_id as machine_public_id,
    networks.public_id as network_public_id,
    networks.kind      as network_kind
from addresses
inner join machines on addresses.machine_id = machines.id
inner join networks on addresses.network_id = networks.id
where addresses.machine_id is not null
order by machines.public_id, networks.kind, addresses.address, addresses.public_id;
