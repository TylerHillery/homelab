-- name: CreatePurchase :one
insert into purchases (
    id, public_id, primary_asset_id, total_price_cents, currency,
    purchased_on, source, notes
)
values (?, ?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: GetPurchaseByPublicID :one
select
    id,
    public_id,
    primary_asset_id,
    total_price_cents,
    currency,
    purchased_on,
    source,
    notes,
    created_at,
    updated_at
from purchases
where public_id = ?;

-- name: ListPurchases :many
select
    id,
    public_id,
    primary_asset_id,
    total_price_cents,
    currency,
    purchased_on,
    source,
    notes,
    created_at,
    updated_at
from purchases
order by coalesce(purchased_on, '9999-12-31'), created_at, public_id;

-- name: UpdatePurchase :one
update purchases
set
    total_price_cents = ?,
    primary_asset_id = ?,
    currency = ?,
    purchased_on = ?,
    source = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?
returning *;

-- name: DeletePurchase :execrows
delete from purchases
where public_id = ?;

-- name: CreatePurchaseAsset :exec
insert into purchase_assets (purchase_id, asset_id)
values (?, ?);

-- name: ListPurchaseAssetsByPurchaseID :many
with selected_purchase as (
    select
        purchases.id,
        purchases.primary_asset_id
    from purchases
    where purchases.id = ?
)

select assets.public_id
from selected_purchase
inner join assets on selected_purchase.primary_asset_id = assets.id
union all
select assets.public_id
from purchase_assets
inner join assets on purchase_assets.asset_id = assets.id
inner join selected_purchase on purchase_assets.purchase_id = selected_purchase.id
order by public_id;

-- name: ListPurchaseAssets :many
select
    purchases.id     as purchase_id,
    assets.public_id as asset_public_id
from purchases
inner join assets on purchases.primary_asset_id = assets.id
union all
select
    purchase_assets.purchase_id,
    assets.public_id as asset_public_id
from purchase_assets
inner join assets on purchase_assets.asset_id = assets.id
order by purchase_assets.purchase_id, assets.public_id;

-- name: DeletePurchaseAssets :execrows
delete from purchase_assets
where purchase_id = ?;

-- name: CreatePurchaseLink :one
insert into purchase_links (purchase_id, position, kind, label, url)
values (?, ?, ?, ?, ?)
returning *;

-- name: ListPurchaseLinksByPurchaseID :many
select
    purchase_id,
    position,
    kind,
    label,
    url
from purchase_links
where purchase_id = ?
order by position;

-- name: ListPurchaseLinks :many
select
    purchase_id,
    position,
    kind,
    label,
    url
from purchase_links
order by purchase_id, position;

-- name: DeletePurchaseLinks :execrows
delete from purchase_links
where purchase_id = ?;

-- name: ListPurchaseAmounts :many
select
    purchases.currency,
    purchases.total_price_cents,
    (
        select count(*) as asset_count
        from purchase_assets
        where purchase_assets.purchase_id = purchases.id
    ) + 1 as asset_count
from purchases
order by purchases.currency;
