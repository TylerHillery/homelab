-- name: ListLinks :many
select
    short,
    long,
    created,
    lastedit,
    owner
from links
order by short;

-- name: GetLink :one
select
    short,
    long,
    created,
    lastedit,
    owner
from links
where id = ?;

-- name: UpsertLink :execrows
insert or replace into links (id, short, long, created, lastedit, owner)
values (?, ?, ?, ?, ?, ?);

-- name: DeleteLink :execrows
delete from links
where id = ?;

-- name: ListLinksByOwner :many
select
    short,
    long,
    created,
    lastedit,
    owner
from links
where lower(owner) = lower(?)
order by short;
