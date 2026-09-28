-- name: CreateMachine :one
insert into machines (
    id,
    public_id,
    asset_id,
    parent_machine_id,
    machine_provider_id,
    area_id,
    name,
    slug,
    kind,
    is_favorite,
    virtualization_platform,
    hostname,
    os_machine_id,
    operating_system,
    operating_system_version,
    kernel,
    architecture,
    cpu_count,
    cpu_thread_count,
    cpu_allocation,
    cpu_vendor,
    memory_bytes,
    storage_bytes,
    storage_media_kind,
    storage_interface_kind,
    estimated_monthly_cost_cents,
    cost_currency,
    notes
)
values (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
returning *;

-- name: GetMachineByPublicID :one
with machine_identifiers as (
    select
        id,
        public_id
    from machines
)

select
    machines.id,
    machines.public_id,
    machines.asset_id,
    machines.parent_machine_id,
    machines.machine_provider_id,
    machines.area_id,
    machines.name,
    machines.slug,
    machines.kind,
    machines.is_favorite,
    machines.virtualization_platform,
    machines.hostname,
    machines.os_machine_id,
    machines.operating_system,
    machines.operating_system_version,
    machines.kernel,
    machines.architecture,
    machines.cpu_count,
    machines.cpu_thread_count,
    machines.cpu_allocation,
    machines.cpu_vendor,
    machines.memory_bytes,
    machines.storage_bytes,
    machines.storage_media_kind,
    machines.storage_interface_kind,
    machines.estimated_monthly_cost_cents,
    machines.cost_currency,
    machines.notes,
    machines.created_at,
    machines.updated_at,
    machine_providers.public_id as machine_provider_public_id,
    areas.public_id             as area_public_id,
    assets.public_id            as asset_public_id,
    parent.public_id            as parent_machine_public_id
from machines
inner join machine_providers on machines.machine_provider_id = machine_providers.id
inner join areas on machines.area_id = areas.id
left join assets on machines.asset_id = assets.id
left join machine_identifiers as parent on machines.parent_machine_id = parent.id
where machines.public_id = ?;

-- name: ListMachineDetails :many
with machine_identifiers as (
    select
        id,
        public_id
    from machines
)

select
    machines.id,
    machines.public_id,
    machines.asset_id,
    machines.parent_machine_id,
    machines.machine_provider_id,
    machines.area_id,
    machines.name,
    machines.slug,
    machines.kind,
    machines.is_favorite,
    machines.virtualization_platform,
    machines.hostname,
    machines.os_machine_id,
    machines.operating_system,
    machines.operating_system_version,
    machines.kernel,
    machines.architecture,
    machines.cpu_count,
    machines.cpu_thread_count,
    machines.cpu_allocation,
    machines.cpu_vendor,
    machines.memory_bytes,
    machines.storage_bytes,
    machines.storage_media_kind,
    machines.storage_interface_kind,
    machines.estimated_monthly_cost_cents,
    machines.cost_currency,
    machines.notes,
    machines.created_at,
    machines.updated_at,
    machine_providers.public_id as machine_provider_public_id,
    areas.public_id             as area_public_id,
    assets.public_id            as asset_public_id,
    parent.public_id            as parent_machine_public_id
from machines
inner join machine_providers on machines.machine_provider_id = machine_providers.id
inner join areas on machines.area_id = areas.id
left join assets on machines.asset_id = assets.id
left join machine_identifiers as parent on machines.parent_machine_id = parent.id
order by machines.name;

-- name: UpdateMachine :one
update machines
set
    asset_id = ?,
    parent_machine_id = ?,
    machine_provider_id = ?,
    area_id = ?,
    name = ?,
    slug = ?,
    kind = ?,
    is_favorite = ?,
    virtualization_platform = ?,
    hostname = ?,
    os_machine_id = ?,
    operating_system = ?,
    operating_system_version = ?,
    kernel = ?,
    architecture = ?,
    cpu_count = ?,
    cpu_thread_count = ?,
    cpu_allocation = ?,
    cpu_vendor = ?,
    memory_bytes = ?,
    storage_bytes = ?,
    storage_media_kind = ?,
    storage_interface_kind = ?,
    estimated_monthly_cost_cents = ?,
    cost_currency = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning *;

-- name: DeleteMachine :execrows
delete from machines
where public_id = ?;

-- name: SetMachineFavorite :execrows
update machines
set
    is_favorite = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?;
