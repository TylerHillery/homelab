-- name: CreateInstanceDependency :one
insert into instance_dependencies (
    id, public_id, consumer_instance_id, provider_instance_id, kind, notes
)
values (?, ?, ?, ?, ?, ?)
returning id, public_id, consumer_instance_id, provider_instance_id, kind, notes, created_at;

-- name: GetInstanceDependencyByPublicID :one
select
    instance_dependencies.id,
    instance_dependencies.public_id,
    instance_dependencies.consumer_instance_id,
    instance_dependencies.provider_instance_id,
    instance_dependencies.kind,
    instance_dependencies.notes,
    instance_dependencies.created_at,
    consumer.public_id as consumer_public_id,
    provider.public_id as provider_public_id
from instance_dependencies
inner join instances as consumer on instance_dependencies.consumer_instance_id = consumer.id
inner join instances as provider on instance_dependencies.provider_instance_id = provider.id
where instance_dependencies.public_id = ?;

-- name: ListInstanceDependencies :many
select
    instance_dependencies.id,
    instance_dependencies.public_id,
    instance_dependencies.consumer_instance_id,
    instance_dependencies.provider_instance_id,
    instance_dependencies.kind,
    instance_dependencies.notes,
    instance_dependencies.created_at,
    consumer.public_id as consumer_public_id,
    provider.public_id as provider_public_id
from instance_dependencies
inner join instances as consumer on instance_dependencies.consumer_instance_id = consumer.id
inner join instances as provider on instance_dependencies.provider_instance_id = provider.id
order by consumer.public_id, provider.public_id;

-- name: UpdateInstanceDependency :execrows
update instance_dependencies
set consumer_instance_id = ?, provider_instance_id = ?, kind = ?, notes = ?
where public_id = ?;

-- name: DeleteInstanceDependency :execrows
delete from instance_dependencies
where public_id = ?;
