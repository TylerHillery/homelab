-- name: ListTopologyMachineHardware :many
select
    machines.public_id   as machine_public_id,
    components.public_id as asset_public_id,
    products.public_id   as product_public_id,
    products.kind,
    products.name,
    processor_specs.generation,
    processor_specs.codename,
    memory_specs.memory_type
from machines
inner join assets as components on machines.asset_id = components.parent_asset_id
inner join products on components.product_id = products.id
left join processor_specs on products.id = processor_specs.product_id
left join memory_specs on products.id = memory_specs.product_id
where products.kind in ('processor', 'memory')
order by machines.public_id, products.kind, components.public_id;
