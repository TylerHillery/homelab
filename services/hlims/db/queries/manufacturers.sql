-- name: CreateManufacturer :one
insert into manufacturers (id, public_id, name, slug)
values (?, ?, ?, ?)
returning id, public_id, name, slug, created_at, updated_at;

-- name: GetManufacturerByPublicID :one
select
    id,
    public_id,
    name,
    slug,
    created_at,
    updated_at
from manufacturers
where public_id = ?;

-- name: ListManufacturers :many
select
    id,
    public_id,
    name,
    slug,
    created_at,
    updated_at
from manufacturers
order by name;

-- name: UpdateManufacturer :one
update manufacturers
set name = ?, slug = ?, updated_at = strftime('%s', 'now')
where public_id = ?
returning id, public_id, name, slug, created_at, updated_at;

-- name: DeleteManufacturer :execrows
delete from manufacturers
where public_id = ?;
