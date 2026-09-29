-- name: CreateDeployment :one
insert into deployments (
    id, public_id, machine_id, name, slug, working_directory,
    compose_project, compose_files, notes
)
values (?, ?, ?, ?, ?, ?, ?, ?, ?)
returning
    id, public_id, machine_id, name, slug, working_directory,
    compose_project, compose_files, notes, created_at, updated_at;

-- name: GetDeploymentByPublicID :one
select
    deployments.id,
    deployments.public_id,
    deployments.machine_id,
    deployments.name,
    deployments.slug,
    deployments.working_directory,
    deployments.compose_project,
    deployments.compose_files,
    deployments.notes,
    deployments.created_at,
    deployments.updated_at,
    machines.public_id as machine_public_id,
    machines.name      as machine_name
from deployments
inner join machines on deployments.machine_id = machines.id
where deployments.public_id = ?;

-- name: ListDeployments :many
select
    deployments.id,
    deployments.public_id,
    deployments.machine_id,
    deployments.name,
    deployments.slug,
    deployments.working_directory,
    deployments.compose_project,
    deployments.compose_files,
    deployments.notes,
    deployments.created_at,
    deployments.updated_at,
    machines.public_id as machine_public_id,
    machines.name      as machine_name
from deployments
inner join machines on deployments.machine_id = machines.id
order by machines.name, deployments.name;

-- name: UpdateDeployment :execrows
update deployments
set
    machine_id = ?,
    name = ?,
    slug = ?,
    working_directory = ?,
    compose_project = ?,
    compose_files = ?,
    notes = ?,
    updated_at = strftime('%s', 'now')
where public_id = ?;

-- name: DeleteDeployment :execrows
delete from deployments
where public_id = ?;
