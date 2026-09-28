-- name: CreateDNSZone :one
insert into dns_zones (id, public_id, name, registrar, notes)
values (?, ?, ?, ?, ?)
returning *;

-- name: ListDNSZones :many
select
    id,
    public_id,
    name,
    registrar,
    notes
from dns_zones
order by name;

-- name: GetDNSZoneByPublicID :one
select
    id,
    public_id,
    name,
    registrar,
    notes
from dns_zones
where public_id = ?;

-- name: CreateDNSRecord :one
insert into dns_records (id, public_id, zone_id, name, kind, address_id)
values (?, ?, ?, ?, ?, ?)
returning *;

-- name: ListDNSRecords :many
select
    dns_records.id,
    dns_records.public_id,
    dns_zones.public_id as zone_public_id,
    dns_records.name,
    dns_records.kind,
    addresses.public_id as address_public_id,
    addresses.address,
    cast(case
        when dns_records.name = '@' then dns_zones.name
        else dns_records.name || '.' || dns_zones.name
    end as text)        as fqdn
from dns_records
inner join dns_zones on dns_records.zone_id = dns_zones.id
inner join addresses on dns_records.address_id = addresses.id
order by dns_zones.name, dns_records.name, dns_records.kind, addresses.address;

-- name: GetDNSRecordByPublicID :one
select
    dns_records.id,
    dns_records.public_id,
    dns_zones.public_id as zone_public_id,
    dns_records.name,
    dns_records.kind,
    addresses.public_id as address_public_id,
    addresses.address,
    cast(case
        when dns_records.name = '@' then dns_zones.name
        else dns_records.name || '.' || dns_zones.name
    end as text)        as fqdn
from dns_records
inner join dns_zones on dns_records.zone_id = dns_zones.id
inner join addresses on dns_records.address_id = addresses.id
where dns_records.public_id = ?;

-- name: CreateIngressRoute :one
insert into ingress_routes (endpoint_id, ingress_instance_id, kind, target)
values (?, ?, ?, ?)
returning *;

-- name: GetEndpointIngressIDs :one
select
    instance_endpoints.id,
    instances.machine_id
from instance_endpoints
inner join instances on instance_endpoints.instance_id = instances.id
where instance_endpoints.public_id = ?;

-- name: ListIngressRoutes :many
select
    instance_endpoints.public_id as endpoint_public_id,
    instances.public_id          as ingress_instance_public_id,
    services.name                as ingress_service_name,
    ingress_routes.kind,
    ingress_routes.target
from ingress_routes
inner join instance_endpoints on ingress_routes.endpoint_id = instance_endpoints.id
inner join instances on ingress_routes.ingress_instance_id = instances.id
inner join services on instances.service_id = services.id
order by instance_endpoints.public_id;

-- name: GetIngressRouteByEndpointPublicID :one
select
    instance_endpoints.public_id as endpoint_public_id,
    instances.public_id          as ingress_instance_public_id,
    services.name                as ingress_service_name,
    ingress_routes.kind,
    ingress_routes.target
from ingress_routes
inner join instance_endpoints on ingress_routes.endpoint_id = instance_endpoints.id
inner join instances on ingress_routes.ingress_instance_id = instances.id
inner join services on instances.service_id = services.id
where instance_endpoints.public_id = ?;
