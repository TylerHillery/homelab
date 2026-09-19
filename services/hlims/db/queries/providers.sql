-- name: CreateMachineProvider :one
insert into machine_providers (id, public_id, name, slug)
values (?, ?, ?, ?)
returning id, public_id, name, slug, created_at, updated_at;

-- name: GetMachineProviderByPublicID :one
select
    id,
    public_id,
    name,
    slug,
    created_at,
    updated_at
from machine_providers
where public_id = ?;

-- name: ListMachineProviders :many
select
    id,
    public_id,
    name,
    slug,
    created_at,
    updated_at,
    exists(
        select 1 as result
        from machine_provider_logos
        where machine_provider_logos.machine_provider_id = machine_providers.id
    ) as has_logo
from machine_providers
order by name;

-- name: UpdateMachineProvider :one
update machine_providers
set name = ?, slug = ?, updated_at = strftime('%s', 'now')
where public_id = ?
returning id, public_id, name, slug, created_at, updated_at;

-- name: DeleteMachineProvider :execrows
delete from machine_providers
where public_id = ?;

-- name: MachineProviderHasLogo :one
select exists(
    select 1 as result
    from machine_provider_logos
    inner join machine_providers
        on machine_provider_logos.machine_provider_id = machine_providers.id
    where machine_providers.public_id = ?
) as has_logo;

-- name: UpsertMachineProviderLogo :exec
insert into machine_provider_logos (machine_provider_id, content_type, image_data)
select
    machine_providers.id,
    sqlc.arg(content_type) as content_type,
    sqlc.arg(image_data)   as image_data
from machine_providers
where machine_providers.public_id = sqlc.arg(public_id)
on conflict (machine_provider_id) do update set
    content_type = excluded.content_type,
    image_data = excluded.image_data,
    updated_at = strftime('%s', 'now');

-- name: GetMachineProviderLogo :one
select
    machine_provider_logos.content_type,
    machine_provider_logos.image_data,
    machine_provider_logos.updated_at
from machine_provider_logos
inner join machine_providers
    on machine_provider_logos.machine_provider_id = machine_providers.id
where machine_providers.public_id = ?;

-- name: DeleteMachineProviderLogo :execrows
delete from machine_provider_logos
where machine_provider_id = (
    select machine_providers.id
    from machine_providers
    where machine_providers.public_id = ?
);

-- name: CreateArea :one
insert into areas (
    id,
    public_id,
    machine_provider_id,
    name,
    slug,
    provider_code,
    notes
)
values (?, ?, ?, ?, ?, ?, ?)
returning
    id,
    public_id,
    machine_provider_id,
    name,
    slug,
    provider_code,
    notes,
    created_at,
    updated_at;

-- name: GetAreaByPublicID :one
select
    id,
    public_id,
    machine_provider_id,
    name,
    slug,
    provider_code,
    notes,
    created_at,
    updated_at
from areas
where public_id = ?;

-- name: GetAreaDetailByPublicID :one
select
    areas.public_id,
    machine_providers.public_id as machine_provider_public_id,
    areas.name,
    areas.slug,
    areas.provider_code,
    areas.notes
from areas
inner join machine_providers on areas.machine_provider_id = machine_providers.id
where areas.public_id = ?;

-- name: ListAreas :many
select
    areas.id,
    areas.public_id,
    areas.name,
    areas.slug,
    areas.provider_code,
    areas.notes,
    areas.created_at,
    areas.updated_at,
    machine_providers.public_id as machine_provider_public_id
from areas
inner join machine_providers on areas.machine_provider_id = machine_providers.id
order by machine_providers.name, areas.name;

-- name: UpdateArea :one
update areas
set
    machine_provider_id = ?,
    name = ?,
    slug = ?,
    provider_code = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning
    id,
    public_id,
    machine_provider_id,
    name,
    slug,
    provider_code,
    notes,
    created_at,
    updated_at;

-- name: DeleteArea :execrows
delete from areas
where public_id = ?;
