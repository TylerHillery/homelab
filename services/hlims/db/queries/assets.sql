-- name: CreateAsset :one
insert into assets (
    id,
    public_id,
    product_id,
    parent_asset_id,
    parent_slot,
    area_id,
    name,
    serial_number,
    system_uuid,
    notes
)
values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: GetAssetByPublicID :one
with recursive
asset_identifiers as (
    select
        id,
        public_id
    from assets
),

asset_placement as (
    select
        id as origin_id,
        id as asset_id,
        parent_asset_id,
        area_id,
        0  as depth
    from assets
    union all
    select
        asset_placement.origin_id,
        parent.id                 as asset_id,
        parent.parent_asset_id,
        parent.area_id,
        asset_placement.depth + 1 as depth
    from asset_placement
    inner join assets as parent on asset_placement.parent_asset_id = parent.id
),

effective_placements as (
    select
        origin_id,
        area_id
    from asset_placement
    where area_id is not null
),

asset_purchase_owners as (
    select
        purchases.id               as purchase_id,
        purchases.primary_asset_id as asset_id
    from purchases
    union all
    select
        purchase_assets.purchase_id,
        purchase_assets.asset_id
    from purchase_assets
)

select
    assets.id,
    assets.public_id,
    assets.product_id,
    assets.parent_asset_id,
    assets.parent_slot,
    assets.area_id,
    assets.name,
    assets.serial_number,
    assets.system_uuid,
    assets.notes,
    assets.created_at,
    assets.updated_at,
    products.public_id        as product_public_id,
    parent.public_id          as parent_asset_public_id,
    placement.public_id       as area_public_id,
    effective_area.public_id  as effective_area_public_id,
    purchase_record.public_id as purchase_public_id
from assets
inner join products on assets.product_id = products.id
left join asset_identifiers as parent on assets.parent_asset_id = parent.id
left join areas as placement on assets.area_id = placement.id
left join effective_placements on assets.id = effective_placements.origin_id
left join areas as effective_area on effective_placements.area_id = effective_area.id
left join asset_purchase_owners on assets.id = asset_purchase_owners.asset_id
left join purchases as purchase_record on asset_purchase_owners.purchase_id = purchase_record.id
where assets.public_id = ?;

-- name: ListAssetDetails :many
with recursive
asset_identifiers as (
    select
        id,
        public_id
    from assets
),

asset_placement as (
    select
        id as origin_id,
        id as asset_id,
        parent_asset_id,
        area_id,
        0  as depth
    from assets
    union all
    select
        asset_placement.origin_id,
        parent.id                 as asset_id,
        parent.parent_asset_id,
        parent.area_id,
        asset_placement.depth + 1 as depth
    from asset_placement
    inner join assets as parent on asset_placement.parent_asset_id = parent.id
),

effective_placements as (
    select
        origin_id,
        area_id
    from asset_placement
    where area_id is not null
),

asset_purchase_owners as (
    select
        purchases.id               as purchase_id,
        purchases.primary_asset_id as asset_id
    from purchases
    union all
    select
        purchase_assets.purchase_id,
        purchase_assets.asset_id
    from purchase_assets
)

select
    assets.id,
    assets.public_id,
    assets.product_id,
    assets.parent_asset_id,
    assets.parent_slot,
    assets.area_id,
    assets.name,
    assets.serial_number,
    assets.system_uuid,
    assets.notes,
    assets.created_at,
    assets.updated_at,
    products.public_id        as product_public_id,
    parent.public_id          as parent_asset_public_id,
    placement.public_id       as area_public_id,
    effective_area.public_id  as effective_area_public_id,
    purchase_record.public_id as purchase_public_id
from assets
inner join products on assets.product_id = products.id
left join asset_identifiers as parent on assets.parent_asset_id = parent.id
left join areas as placement on assets.area_id = placement.id
left join effective_placements on assets.id = effective_placements.origin_id
left join areas as effective_area on effective_placements.area_id = effective_area.id
left join asset_purchase_owners on assets.id = asset_purchase_owners.asset_id
left join purchases as purchase_record on asset_purchase_owners.purchase_id = purchase_record.id
order by coalesce(assets.name, assets.serial_number, assets.public_id);

-- name: ListContainedAssets :many
with recursive descendants (id, depth, sort_path) as (
    select
        assets.id,
        0  as depth,
        '' as sort_path
    from assets
    where assets.public_id = ?
    union all
    select
        child.id,
        descendants.depth + 1,
        descendants.sort_path || '/' || lower(coalesce(child.name, child.public_id))
        || '-' || child.public_id
    from descendants
    inner join assets as child on descendants.id = child.parent_asset_id
)

select
    child.public_id,
    child.name,
    child.parent_slot,
    products.public_id as product_public_id,
    products.name      as product_name,
    products.kind      as product_kind,
    descendants.depth
from descendants
inner join assets as child on descendants.id = child.id
inner join products on child.product_id = products.id
where descendants.depth > 0
order by descendants.sort_path;

-- name: UpdateAsset :one
update assets
set
    product_id = ?,
    parent_asset_id = ?,
    parent_slot = ?,
    area_id = ?,
    name = ?,
    serial_number = ?,
    system_uuid = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning *;

-- name: DeleteAsset :execrows
delete from assets
where public_id = ?;

-- name: CreateAssetLink :one
insert into asset_links (asset_id, position, kind, label, url)
values (?, ?, ?, ?, ?)
returning *;

-- name: ListAssetLinksByAssetID :many
select
    asset_id,
    position,
    kind,
    label,
    url
from asset_links
where asset_id = ?
order by position;

-- name: ListAssetLinks :many
select
    asset_id,
    position,
    kind,
    label,
    url
from asset_links
order by asset_id, position;

-- name: DeleteAssetLinks :execrows
delete from asset_links
where asset_id = ?;
