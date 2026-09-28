-- name: CreateProduct :one
insert into products (
    id, public_id, manufacturer_id, kind, name, part_number, notes
)
values (?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: GetProductByPublicID :one
select
    products.id,
    products.public_id,
    products.manufacturer_id,
    products.kind,
    products.name,
    products.part_number,
    products.notes,
    products.created_at,
    products.updated_at,
    manufacturers.public_id     as manufacturer_public_id,
    processor_specs.core_count,
    processor_specs.thread_count,
    processor_specs.base_clock_mhz,
    processor_specs.generation,
    processor_specs.codename,
    processor_specs.virtualization,
    memory_specs.capacity_bytes as memory_capacity_bytes,
    memory_specs.memory_type,
    memory_specs.form_factor,
    memory_specs.speed_mts,
    drive_specs.capacity_bytes  as drive_capacity_bytes,
    drive_specs.media_kind,
    drive_specs.interface_kind,
    rack_specs.rack_units,
    rack_specs.mounting_standard,
    wifi_specs.generation       as wifi_generation,
    wifi_specs.ieee_standard,
    wifi_specs.wifi_class,
    wifi_specs.max_channel_width_mhz
from products
inner join manufacturers on products.manufacturer_id = manufacturers.id
left join processor_specs on products.id = processor_specs.product_id
left join memory_specs on products.id = memory_specs.product_id
left join drive_specs on products.id = drive_specs.product_id
left join rack_specs on products.id = rack_specs.product_id
left join wifi_specs on products.id = wifi_specs.product_id
where products.public_id = ?;

-- name: ListProductDetails :many
select
    products.id,
    products.public_id,
    products.manufacturer_id,
    products.kind,
    products.name,
    products.part_number,
    products.notes,
    products.created_at,
    products.updated_at,
    manufacturers.public_id     as manufacturer_public_id,
    processor_specs.core_count,
    processor_specs.thread_count,
    processor_specs.base_clock_mhz,
    processor_specs.generation,
    processor_specs.codename,
    processor_specs.virtualization,
    memory_specs.capacity_bytes as memory_capacity_bytes,
    memory_specs.memory_type,
    memory_specs.form_factor,
    memory_specs.speed_mts,
    drive_specs.capacity_bytes  as drive_capacity_bytes,
    drive_specs.media_kind,
    drive_specs.interface_kind,
    rack_specs.rack_units,
    rack_specs.mounting_standard,
    wifi_specs.generation       as wifi_generation,
    wifi_specs.ieee_standard,
    wifi_specs.wifi_class,
    wifi_specs.max_channel_width_mhz
from products
inner join manufacturers on products.manufacturer_id = manufacturers.id
left join processor_specs on products.id = processor_specs.product_id
left join memory_specs on products.id = memory_specs.product_id
left join drive_specs on products.id = drive_specs.product_id
left join rack_specs on products.id = rack_specs.product_id
left join wifi_specs on products.id = wifi_specs.product_id
order by manufacturers.name, products.name;

-- name: UpdateProduct :one
update products
set
    manufacturer_id = ?,
    kind = ?,
    name = ?,
    part_number = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning *;

-- name: DeleteProduct :execrows
delete from products
where public_id = ?;

-- name: CreateProcessorSpec :one
insert into processor_specs (
    product_id, core_count, thread_count, base_clock_mhz, generation, codename, virtualization
)
values (?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: DeleteProcessorSpec :execrows
delete from processor_specs
where product_id = ?;

-- name: CreateMemorySpec :one
insert into memory_specs (
    product_id, capacity_bytes, memory_type, form_factor, speed_mts
)
values (?, ?, ?, ?, ?)
returning *;

-- name: DeleteMemorySpec :execrows
delete from memory_specs
where product_id = ?;

-- name: CreateDriveSpec :one
insert into drive_specs (
    product_id, capacity_bytes, media_kind, interface_kind
)
values (?, ?, ?, ?)
returning *;

-- name: DeleteDriveSpec :execrows
delete from drive_specs
where product_id = ?;

-- name: CreateRackSpec :one
insert into rack_specs (product_id, rack_units, mounting_standard)
values (?, ?, ?)
returning *;

-- name: DeleteRackSpec :execrows
delete from rack_specs
where product_id = ?;

-- name: CreateWiFiSpec :one
insert into wifi_specs (product_id, generation, ieee_standard, wifi_class, max_channel_width_mhz)
values (?, ?, ?, ?, ?)
returning *;

-- name: DeleteWiFiSpec :execrows
delete from wifi_specs
where product_id = ?;

-- name: CreateWiFiBand :exec
insert into wifi_band_specs (product_id, band_ghz, max_link_mbps)
values (?, ?, ?);

-- name: ListWiFiBandsByProductID :many
select
    product_id,
    band_ghz,
    max_link_mbps
from wifi_band_specs
where product_id = ?
order by band_ghz;

-- name: ListWiFiBands :many
select
    product_id,
    band_ghz,
    max_link_mbps
from wifi_band_specs
order by product_id, band_ghz;

-- name: CreateProductPortProfile :one
insert into product_port_profiles (
    product_id, position, name, port_count, connector, speed_mbps
)
values (?, ?, ?, ?, ?, ?)
returning *;

-- name: ListProductPortProfilesByProductID :many
select
    product_id,
    position,
    name,
    port_count,
    connector,
    speed_mbps
from product_port_profiles
where product_id = ?
order by position;

-- name: ListProductPortProfiles :many
select
    product_id,
    position,
    name,
    port_count,
    connector,
    speed_mbps
from product_port_profiles
order by product_id, position;

-- name: DeleteProductPortProfiles :execrows
delete from product_port_profiles
where product_id = ?;

-- name: CreateProductLink :one
insert into product_links (product_id, position, kind, label, url)
values (?, ?, ?, ?, ?)
returning *;

-- name: ListProductLinksByProductID :many
select
    product_id,
    position,
    kind,
    label,
    url
from product_links
where product_id = ?
order by position;

-- name: ListProductLinks :many
select
    product_id,
    position,
    kind,
    label,
    url
from product_links
order by product_id, position;

-- name: DeleteProductLinks :execrows
delete from product_links
where product_id = ?;
