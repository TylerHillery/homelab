-- +goose Up
-- noqa: disable=LT01,LT02

create table machine_providers (
    id         text    not null primary key,
    public_id  text    not null unique,
    name       text    not null collate nocase,
    slug       text    not null collate nocase,
    created_at integer not null default (strftime('%s', 'now')),
    updated_at integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0)
);

create unique index machine_providers_name_idx on machine_providers (name);
create unique index machine_providers_slug_idx on machine_providers (slug);

create table machine_provider_logos (
    machine_provider_id text    not null primary key
        references machine_providers (id) on delete cascade,
    content_type        text    not null,
    image_data          blob    not null,
    created_at          integer not null default (strftime('%s', 'now')),
    updated_at          integer not null default (strftime('%s', 'now')),
    check (content_type in ('image/png', 'image/jpeg', 'image/webp')),
    check (length(image_data) between 1 and 1048576)
);

create table areas (
    id                  text    not null primary key,
    public_id           text    not null unique,
    machine_provider_id text    not null
        references machine_providers (id) on delete restrict,
    name                text    not null collate nocase,
    slug                text    not null collate nocase,
    provider_code       text,
    notes               text,
    created_at          integer not null default (strftime('%s', 'now')),
    updated_at          integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0)
);

create unique index areas_name_idx
    on areas (machine_provider_id, name);

create unique index areas_slug_idx
    on areas (machine_provider_id, slug);

create unique index areas_provider_code_idx
    on areas (machine_provider_id, provider_code)
    where provider_code is not null;

create table manufacturers (
    id         text    not null primary key,
    public_id  text    not null unique,
    name       text    not null collate nocase,
    slug       text    not null collate nocase,
    created_at integer not null default (strftime('%s', 'now')),
    updated_at integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0)
);

create unique index manufacturers_name_idx on manufacturers (name);
create unique index manufacturers_slug_idx on manufacturers (slug);

create table products (
    id              text    not null primary key,
    public_id       text    not null unique,
    manufacturer_id text    not null
        references manufacturers (id) on delete restrict,
    kind            text    not null,
    name            text    not null collate nocase,
    part_number     text,
    notes           text,
    created_at      integer not null default (strftime('%s', 'now')),
    updated_at      integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (kind in (
        'system',
        'processor',
        'memory',
        'drive',
        'rack',
        'router',
        'switch',
        'access_point',
        'network_adapter'
    ))
);

create index products_manufacturer_id_idx on products (manufacturer_id);

create unique index products_name_idx
    on products (manufacturer_id, name);

create unique index products_part_number_idx
    on products (manufacturer_id, part_number)
    where part_number is not null;

create table processor_specs (
    product_id     text    not null primary key
        references products (id) on delete cascade,
    core_count     integer not null,
    thread_count   integer not null,
    base_clock_mhz integer,
    virtualization text,
    check (core_count > 0),
    check (thread_count >= core_count),
    check (base_clock_mhz is null or base_clock_mhz > 0)
);

create table memory_specs (
    product_id     text    not null primary key
        references products (id) on delete cascade,
    capacity_bytes integer not null,
    memory_type    text    not null,
    form_factor    text,
    speed_mts      integer,
    check (capacity_bytes > 0),
    check (speed_mts is null or speed_mts > 0)
);

create table drive_specs (
    product_id     text    not null primary key
        references products (id) on delete cascade,
    capacity_bytes integer not null,
    media_kind     text    not null,
    interface_kind text,
    check (capacity_bytes > 0),
    check (media_kind in ('hdd', 'ssd')),
    check (
        interface_kind is null
        or interface_kind in ('sata', 'sas', 'nvme', 'usb', 'scsi', 'virtio', 'virtual')
    )
);

create table rack_specs (
    product_id        text    not null primary key
        references products (id) on delete cascade,
    rack_units        integer not null,
    mounting_standard text    not null,
    check (rack_units > 0),
    check (length(trim(mounting_standard)) > 0)
);

create table product_port_profiles (
    product_id text    not null
        references products (id) on delete cascade,
    position   integer not null,
    name       text,
    port_count integer not null,
    connector  text    not null,
    speed_mbps integer not null,
    primary key (product_id, position),
    check (position >= 0),
    check (name is null or length(trim(name)) > 0),
    check (port_count > 0),
    check (length(trim(connector)) > 0),
    check (speed_mbps > 0)
);

create table product_links (
    product_id text    not null
        references products (id) on delete cascade,
    position   integer not null,
    kind       text    not null,
    label      text,
    url        text    not null,
    primary key (product_id, position),
    unique (product_id, url),
    check (position >= 0),
    check (kind in ('manufacturer', 'retailer', 'manual', 'datasheet', 'support')),
    check (label is null or length(trim(label)) > 0),
    check (length(trim(url)) > 0)
);

-- +goose StatementBegin
create trigger processor_specs_product_kind_insert
before insert on processor_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'processor'
)
begin
    select raise(abort, 'processor specifications require a processor product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger processor_specs_product_kind_update
before update of product_id on processor_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'processor'
)
begin
    select raise(abort, 'processor specifications require a processor product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger memory_specs_product_kind_insert
before insert on memory_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'memory'
)
begin
    select raise(abort, 'memory specifications require a memory product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger memory_specs_product_kind_update
before update of product_id on memory_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'memory'
)
begin
    select raise(abort, 'memory specifications require a memory product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger drive_specs_product_kind_insert
before insert on drive_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'drive'
)
begin
    select raise(abort, 'drive specifications require a drive product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger drive_specs_product_kind_update
before update of product_id on drive_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'drive'
)
begin
    select raise(abort, 'drive specifications require a drive product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger rack_specs_product_kind_insert
before insert on rack_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'rack'
)
begin
    select raise(abort, 'rack specifications require a rack product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger rack_specs_product_kind_update
before update of product_id on rack_specs
when not exists (
    select 1 as result from products
    where products.id = new.product_id and products.kind = 'rack'
)
begin
    select raise(abort, 'rack specifications require a rack product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger product_port_profiles_product_kind_insert
before insert on product_port_profiles
when not exists (
    select 1 as result from products
    where products.id = new.product_id
        and products.kind in ('router', 'switch', 'access_point', 'network_adapter')
)
begin
    select raise(abort, 'port profiles require a network equipment product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger product_port_profiles_product_kind_update
before update of product_id on product_port_profiles
when not exists (
    select 1 as result from products
    where products.id = new.product_id
        and products.kind in ('router', 'switch', 'access_point', 'network_adapter')
)
begin
    select raise(abort, 'port profiles require a network equipment product') as result;
end;
-- +goose StatementEnd

create table assets (
    id              text    not null primary key,
    public_id       text    not null unique,
    product_id      text    not null
        references products (id) on delete restrict,
    parent_asset_id text
        references assets (id) on delete restrict,
    parent_slot     text,
    area_id         text
        references areas (id) on delete set null,
    name            text,
    serial_number   text,
    system_uuid     text,
    notes           text,
    created_at      integer not null default (strftime('%s', 'now')),
    updated_at      integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (parent_asset_id is null or parent_asset_id != id),
    check (parent_asset_id is null or area_id is null),
    check (parent_asset_id is not null or parent_slot is null),
    check (parent_slot is null or length(trim(parent_slot)) > 0)
);

create index assets_product_id_idx on assets (product_id);
create index assets_parent_asset_id_idx on assets (parent_asset_id);
create index assets_area_id_idx on assets (area_id);

create unique index assets_serial_number_idx
    on assets (product_id, serial_number)
    where serial_number is not null;

create unique index assets_system_uuid_idx
    on assets (system_uuid)
    where system_uuid is not null;

-- +goose StatementBegin
create trigger assets_parent_kind_insert
before insert on assets
when new.parent_asset_id is not null and not exists (
    select 1 as result
    from assets as parent
    inner join products on parent.product_id = products.id
    where parent.id = new.parent_asset_id and products.kind in ('system', 'rack')
)
begin
    select raise(abort, 'parent asset must have a system or rack product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger assets_parent_kind_update
before update of parent_asset_id on assets
when new.parent_asset_id is not null and not exists (
    select 1 as result
    from assets as parent
    inner join products on parent.product_id = products.id
    where parent.id = new.parent_asset_id and products.kind in ('system', 'rack')
)
begin
    select raise(abort, 'parent asset must have a system or rack product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger assets_cycle_insert
before insert on assets
when new.parent_asset_id is not null and exists (
    with recursive ancestors (id, parent_asset_id) as (
        select
            origin.id,
            origin.parent_asset_id
        from assets as origin
        where origin.id = new.parent_asset_id
        union all
        select
            assets.id,
            assets.parent_asset_id
        from assets
        inner join ancestors on assets.id = ancestors.parent_asset_id
    )

    select 1 as result from ancestors
    where ancestors.id = new.id
)
begin
    select raise(abort, 'asset hierarchy cannot contain a cycle') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger assets_cycle_update
before update of parent_asset_id on assets
when new.parent_asset_id is not null and exists (
    with recursive ancestors (id, parent_asset_id) as (
        select
            origin.id,
            origin.parent_asset_id
        from assets as origin
        where origin.id = new.parent_asset_id
        union all
        select
            assets.id,
            assets.parent_asset_id
        from assets
        inner join ancestors on assets.id = ancestors.parent_asset_id
    )

    select 1 as result from ancestors
    where ancestors.id = new.id
)
begin
    select raise(abort, 'asset hierarchy cannot contain a cycle') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger assets_machine_location_update
before update of parent_asset_id, area_id on assets
when exists (
    with recursive
    descendants (id) as (
        select old.id
        union all
        select child.id
        from assets as child
        inner join descendants on child.parent_asset_id = descendants.id
    ),

    new_ancestors (id, parent_asset_id, area_id, depth) as (
        select
            parent.id,
            parent.parent_asset_id,
            parent.area_id,
            0 as depth
        from assets as parent
        where parent.id = new.parent_asset_id
        union all
        select
            parent.id,
            parent.parent_asset_id,
            parent.area_id,
            new_ancestors.depth + 1 as depth
        from assets as parent
        inner join new_ancestors on parent.id = new_ancestors.parent_asset_id
    )

    select 1 as result
    from descendants
    inner join machines on descendants.id = machines.asset_id
    where machines.area_id != coalesce(
        new.area_id,
        (
            select new_ancestors.area_id
            from new_ancestors
            where new_ancestors.area_id is not null
            order by new_ancestors.depth
            limit 1
        ),
        ''
    )
)
begin
    select raise(abort, 'machine backing asset must remain in the machine area') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger assets_product_update
before update of product_id on assets
when not exists (
    select 1 as result
    from products
    where products.id = new.product_id
        and (
            (
                not exists (
                    select 1 as result from assets as child
                    where child.parent_asset_id = old.id
                )
                or products.kind in ('system', 'rack')
            )
            and (
                not exists (
                    select 1 as result from machines
                    where machines.asset_id = old.id
                )
                or products.kind = 'system'
            )
            and (
                not exists (
                    select 1 as result from addresses
                    where addresses.asset_id = old.id
                )
                or products.kind in ('router', 'switch', 'access_point')
            )
        )
)
begin
    select raise(abort, 'asset product conflicts with existing references') as result;
end;
-- +goose StatementEnd

create table asset_links (
    asset_id text    not null
        references assets (id) on delete cascade,
    position integer not null,
    kind     text    not null,
    label    text,
    url      text    not null,
    primary key (asset_id, position),
    unique (asset_id, url),
    check (position >= 0),
    check (kind in ('receipt', 'warranty', 'management', 'other')),
    check (label is null or length(trim(label)) > 0),
    check (length(trim(url)) > 0)
);

create table purchases (
    id                text    not null primary key,
    public_id         text    not null unique,
    primary_asset_id  text    not null unique
        references assets (id) on delete restrict,
    total_price_cents integer not null,
    currency          text    not null,
    purchased_on      text,
    source            text,
    notes             text,
    created_at        integer not null default (strftime('%s', 'now')),
    updated_at        integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (total_price_cents >= 0),
    check (length(currency) = 3 and currency not glob '*[^A-Z]*'),
    check (
        purchased_on is null
        or (
            length(purchased_on) = 10
            and date(purchased_on, '+0 days') is not null
            and date(purchased_on, '+0 days') = purchased_on
        )
    ),
    check (source is null or length(trim(source)) > 0)
);

create index purchases_currency_idx on purchases (currency);

create table purchase_assets (
    purchase_id text not null
        references purchases (id) on delete cascade,
    asset_id    text not null unique
        references assets (id) on delete restrict,
    primary key (purchase_id, asset_id)
);

-- +goose StatementBegin
create trigger purchases_primary_asset_insert
before insert on purchases
when exists (
    select 1 as result from purchase_assets
    where purchase_assets.asset_id = new.primary_asset_id
)
begin
    select raise(abort, 'asset already belongs to a purchase') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger purchases_primary_asset_update
before update of primary_asset_id on purchases
when exists (
    select 1 as result from purchase_assets
    where purchase_assets.asset_id = new.primary_asset_id
)
begin
    select raise(abort, 'asset already belongs to a purchase') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger purchase_assets_asset_insert
before insert on purchase_assets
when exists (
    select 1 as result from purchases
    where purchases.primary_asset_id = new.asset_id
)
begin
    select raise(abort, 'asset already belongs to a purchase') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger purchase_assets_asset_update
before update of asset_id on purchase_assets
when exists (
    select 1 as result from purchases
    where purchases.primary_asset_id = new.asset_id
)
begin
    select raise(abort, 'asset already belongs to a purchase') as result;
end;
-- +goose StatementEnd

create table purchase_links (
    purchase_id text    not null
        references purchases (id) on delete cascade,
    position    integer not null,
    kind        text    not null,
    label       text,
    url         text    not null,
    primary key (purchase_id, position),
    unique (purchase_id, url),
    check (position >= 0),
    check (kind in ('receipt', 'listing', 'other')),
    check (label is null or length(trim(label)) > 0),
    check (length(trim(url)) > 0)
);

create table machines (
    id                           text    not null primary key,
    public_id                    text    not null unique,
    asset_id                     text
        references assets (id) on delete restrict,
    parent_machine_id            text
        references machines (id) on delete restrict,
    machine_provider_id          text    not null
        references machine_providers (id) on delete restrict,
    area_id                      text    not null
        references areas (id) on delete restrict,
    name                         text    not null collate nocase,
    slug                         text    not null collate nocase,
    kind                         text    not null,
    is_favorite                  boolean not null default false,
    virtualization_platform      text,
    hostname                     text,
    os_machine_id                text,
    operating_system             text,
    operating_system_version     text,
    kernel                       text,
    architecture                 text,
    cpu_count                    integer,
    cpu_allocation               text,
    cpu_vendor                   text,
    memory_bytes                 integer,
    storage_bytes                integer,
    storage_media_kind           text,
    storage_interface_kind       text,
    estimated_monthly_cost_cents integer,
    cost_currency                text,
    notes                        text,
    created_at                   integer not null default (strftime('%s', 'now')),
    updated_at                   integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0),
    check (parent_machine_id is null or parent_machine_id != id),
    check (kind in ('bare_metal', 'virtual_machine')),
    check (
        kind != 'bare_metal'
        or (
            parent_machine_id is null
            and virtualization_platform is null
            and (cpu_allocation is null or cpu_allocation = 'dedicated')
        )
    ),
    check (kind != 'virtual_machine' or asset_id is null),
    check (cpu_count is null or cpu_count > 0),
    check (cpu_allocation is null or cpu_allocation in ('shared', 'dedicated')),
    check (memory_bytes is null or memory_bytes > 0),
    check (storage_bytes is null or storage_bytes > 0),
    check (storage_media_kind is null or storage_media_kind in ('hdd', 'ssd')),
    check (
        storage_interface_kind is null
        or storage_interface_kind in (
            'sata',
            'sas',
            'nvme',
            'usb',
            'scsi',
            'virtio',
            'virtual'
        )
    ),
    check (estimated_monthly_cost_cents is null or estimated_monthly_cost_cents >= 0),
    check ((estimated_monthly_cost_cents is null) = (cost_currency is null)),
    check (
        cost_currency is null
        or (length(cost_currency) = 3 and cost_currency not glob '*[^A-Z]*')
    )
);

create unique index machines_asset_id_idx
    on machines (asset_id)
    where asset_id is not null;
create index machines_parent_machine_id_idx on machines (parent_machine_id);
create index machines_machine_provider_id_idx on machines (machine_provider_id);
create index machines_area_id_idx on machines (area_id);
create unique index machines_name_idx on machines (name);
create unique index machines_slug_idx on machines (slug);
create unique index machines_provider_id_idx
    on machines (id, machine_provider_id);

create unique index machines_os_machine_id_idx
    on machines (os_machine_id)
    where os_machine_id is not null;

create table machine_users (
    id           text    not null primary key,
    public_id    text    not null unique,
    machine_id   text    not null
        references machines (id) on delete cascade,
    username     text    not null,
    is_preferred integer not null default 0,
    notes        text,
    created_at   integer not null default (strftime('%s', 'now')),
    updated_at   integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (length(username) between 1 and 64),
    check (username not glob '*[^A-Za-z0-9._-]*'),
    check (username not like '-%'),
    check (is_preferred in (0, 1))
);

create unique index machine_users_username_idx
    on machine_users (machine_id, username);

create unique index machine_users_preferred_idx
    on machine_users (machine_id)
    where is_preferred = 1;

-- +goose StatementBegin
create trigger products_kind_update
before update of kind on products
when (
    (new.kind != 'processor' and exists (
        select 1 as result from processor_specs
        where processor_specs.product_id = old.id
    ))
    or (new.kind != 'memory' and exists (
        select 1 as result from memory_specs
        where memory_specs.product_id = old.id
    ))
    or (new.kind != 'drive' and exists (
        select 1 as result from drive_specs
        where drive_specs.product_id = old.id
    ))
    or (new.kind != 'rack' and exists (
        select 1 as result from rack_specs
        where rack_specs.product_id = old.id
    ))
    or (new.kind not in ('router', 'switch', 'access_point', 'network_adapter') and exists (
        select 1 as result from product_port_profiles
        where product_port_profiles.product_id = old.id
    ))
    or (new.kind not in ('system', 'rack') and exists (
        select 1 as result
        from assets
        where assets.product_id = old.id
            and exists (
                select 1 as result from assets as child
                where child.parent_asset_id = assets.id
            )
    ))
    or (new.kind != 'system' and exists (
        select 1 as result
        from assets
        inner join machines on assets.id = machines.asset_id
        where assets.product_id = old.id
    ))
    or (new.kind not in ('router', 'switch', 'access_point') and exists (
        select 1 as result
        from assets
        inner join addresses on assets.id = addresses.asset_id
        where assets.product_id = old.id
    ))
)
begin
    select raise(abort, 'product kind conflicts with existing references') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_asset_kind_insert
before insert on machines
when new.asset_id is not null and not exists (
    select 1 as result
    from assets
    inner join products on assets.product_id = products.id
    where assets.id = new.asset_id and products.kind = 'system'
)
begin
    select raise(abort, 'machine asset must have a system product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_asset_kind_update
before update of asset_id on machines
when new.asset_id is not null and not exists (
    select 1 as result
    from assets
    inner join products on assets.product_id = products.id
    where assets.id = new.asset_id and products.kind = 'system'
)
begin
    select raise(abort, 'machine asset must have a system product') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_asset_location_insert
before insert on machines
when new.asset_id is not null and not exists (
    with recursive ancestors (id, parent_asset_id, area_id) as (
        select
assets.id,
assets.parent_asset_id,
assets.area_id
        from assets
        where assets.id = new.asset_id
        union all
        select
parent.id,
parent.parent_asset_id,
parent.area_id
        from assets as parent
        inner join ancestors on parent.id = ancestors.parent_asset_id
    )

    select 1 as result from ancestors
    where ancestors.area_id = new.area_id
)
begin
    select raise(abort, 'machine backing asset must be in the machine area') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_asset_location_update
before update of asset_id, area_id on machines
when new.asset_id is not null and not exists (
    with recursive ancestors (id, parent_asset_id, area_id) as (
        select
assets.id,
assets.parent_asset_id,
assets.area_id
        from assets
        where assets.id = new.asset_id
        union all
        select
parent.id,
parent.parent_asset_id,
parent.area_id
        from assets as parent
        inner join ancestors on parent.id = ancestors.parent_asset_id
    )

    select 1 as result from ancestors
    where ancestors.area_id = new.area_id
)
begin
    select raise(abort, 'machine backing asset must be in the machine area') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_cycle_insert
before insert on machines
when new.parent_machine_id is not null and exists (
    with recursive ancestors (id, parent_machine_id) as (
        select
            origin.id,
            origin.parent_machine_id
        from machines as origin
        where origin.id = new.parent_machine_id
        union all
        select
            machines.id,
            machines.parent_machine_id
        from machines
        inner join ancestors on machines.id = ancestors.parent_machine_id
    )

    select 1 as result from ancestors
    where ancestors.id = new.id
)
begin
    select raise(abort, 'machine hierarchy cannot contain a cycle') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_cycle_update
before update of parent_machine_id on machines
when new.parent_machine_id is not null and exists (
    with recursive ancestors (id, parent_machine_id) as (
        select
            origin.id,
            origin.parent_machine_id
        from machines as origin
        where origin.id = new.parent_machine_id
        union all
        select
            machines.id,
            machines.parent_machine_id
        from machines
        inner join ancestors on machines.id = ancestors.parent_machine_id
    )

    select 1 as result from ancestors
    where ancestors.id = new.id
)
begin
    select raise(abort, 'machine hierarchy cannot contain a cycle') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_parent_location_insert
before insert on machines
when new.parent_machine_id is not null and not exists (
    select 1 as result from machines as parent
    where parent.id = new.parent_machine_id
        and parent.machine_provider_id = new.machine_provider_id
        and parent.area_id = new.area_id
)
begin
    select raise(abort, 'local parent machine must share provider and area') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_parent_location_update
before update of parent_machine_id, machine_provider_id, area_id on machines
when new.parent_machine_id is not null and not exists (
    select 1 as result from machines as parent
    where parent.id = new.parent_machine_id
        and parent.machine_provider_id = new.machine_provider_id
        and parent.area_id = new.area_id
)
begin
    select raise(abort, 'local parent machine must share provider and area') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_child_location_update
before update of machine_provider_id, area_id on machines
when exists (
    select 1 as result from machines as child
    where child.parent_machine_id = old.id
        and (
            child.machine_provider_id != new.machine_provider_id
            or child.area_id != new.area_id
        )
)
begin
    select raise(abort, 'local child machines must share provider and area') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger areas_machine_provider_update
before update of machine_provider_id on areas
when exists (
    select 1 as result
    from machines
    where machines.area_id = new.id
        and machines.machine_provider_id != new.machine_provider_id
)
begin
    select raise(abort, 'machine area must belong to the machine provider') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_area_provider_insert
before insert on machines
when not exists (
    select 1 as result
    from areas
    where areas.id = new.area_id
        and areas.machine_provider_id = new.machine_provider_id
)
begin
    select raise(abort, 'machine area must belong to the machine provider') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger machines_area_provider_update
before update of machine_provider_id, area_id on machines
when not exists (
    select 1 as result
    from areas
    where areas.id = new.area_id
        and areas.machine_provider_id = new.machine_provider_id
)
begin
    select raise(abort, 'machine area must belong to the machine provider') as result;
end;
-- +goose StatementEnd

create table networks (
    id         text    not null primary key,
    public_id  text    not null unique,
    area_id    text
        references areas (id) on delete set null,
    name       text    not null collate nocase,
    slug       text    not null collate nocase,
    kind       text    not null,
    cidr       text,
    notes      text,
    created_at integer not null default (strftime('%s', 'now')),
    updated_at integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0),
    check (kind in ('lan', 'tailnet', 'cloud_vpc', 'public'))
);

create index networks_area_id_idx on networks (area_id);
create unique index networks_name_idx on networks (name);
create unique index networks_slug_idx on networks (slug);

create table addresses (
    id             text    not null primary key,
    public_id      text    not null unique,
    network_id     text    not null
        references networks (id) on delete cascade,
    machine_id     text
        references machines (id) on delete cascade,
    area_id        text
        references areas (id) on delete cascade,
    asset_id       text
        references assets (id) on delete cascade,
    name           text,
    address        text    not null,
    dns_name       text,
    interface_name text,
    is_primary     integer not null default 0,
    created_at     integer not null default (strftime('%s', 'now')),
    updated_at     integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (
        (machine_id is not null)
        + (area_id is not null)
        + (asset_id is not null)
        = 1
    ),
    check (is_primary in (0, 1))
);

create index addresses_network_id_idx on addresses (network_id);
create index addresses_machine_id_idx on addresses (machine_id);
create index addresses_area_id_idx on addresses (area_id);
create index addresses_asset_id_idx on addresses (asset_id);

create unique index addresses_network_address_idx on addresses (network_id, address);

-- +goose StatementBegin
create trigger addresses_asset_kind_insert
before insert on addresses
when new.asset_id is not null and not exists (
    select 1 as result
    from assets
    inner join products on assets.product_id = products.id
    where assets.id = new.asset_id
        and products.kind in ('router', 'switch', 'access_point')
)
begin
    select raise(abort, 'asset address requires network equipment') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger addresses_asset_kind_update
before update of asset_id on addresses
when new.asset_id is not null and not exists (
    select 1 as result
    from assets
    inner join products on assets.product_id = products.id
    where assets.id = new.asset_id
        and products.kind in ('router', 'switch', 'access_point')
)
begin
    select raise(abort, 'asset address requires network equipment') as result;
end;
-- +goose StatementEnd

create table services (
    id          text    not null primary key,
    public_id   text    not null unique,
    name        text    not null collate nocase,
    slug        text    not null collate nocase,
    description text,
    created_at  integer not null default (strftime('%s', 'now')),
    updated_at  integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0)
);

create unique index services_name_idx on services (name);
create unique index services_slug_idx on services (slug);

create table service_logos (
    service_id   text    not null primary key
        references services (id) on delete cascade,
    content_type text    not null,
    image_data   blob    not null,
    created_at   integer not null default (strftime('%s', 'now')),
    updated_at   integer not null default (strftime('%s', 'now')),
    check (content_type in ('image/png', 'image/jpeg', 'image/webp')),
    check (length(image_data) between 1 and 1048576)
);

create table instances (
    id         text    not null primary key,
    public_id  text    not null unique,
    service_id text    not null
        references services (id) on delete restrict,
    machine_id text    not null
        references machines (id) on delete restrict,
    name       text    not null collate nocase,
    slug       text    not null collate nocase,
    port       integer not null,
    notes      text,
    created_at integer not null default (strftime('%s', 'now')),
    updated_at integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (slug != ''),
    check (slug not glob '*[^0-9a-z-]*'),
    check (slug not like '-%' and slug not like '%-'),
    check (instr(slug, '--') = 0),
    check (port between 1 and 65535)
);

create index instances_service_id_idx on instances (service_id);
create index instances_machine_id_idx on instances (machine_id);

create unique index instances_name_idx
    on instances (service_id, machine_id, name);

create unique index instances_slug_idx
    on instances (service_id, machine_id, slug);

create table instance_endpoints (
    id           text    not null primary key,
    public_id    text    not null unique,
    instance_id  text    not null
        references instances (id) on delete cascade,
    address_id   text    not null
        references addresses (id) on delete restrict,
    name         text    not null collate nocase,
    scheme       text    not null,
    port         integer not null,
    base_path    text    not null default '',
    is_preferred integer not null default 0,
    notes        text,
    created_at   integer not null default (strftime('%s', 'now')),
    updated_at   integer not null default (strftime('%s', 'now')),
    check (length(id) = 36),
    check (length(public_id) = 12),
    check (public_id not glob '*[^0-9a-z]*'),
    check (scheme in ('http', 'https')),
    check (port between 1 and 65535),
    check (base_path = '' or substr(base_path, 1, 1) = '/'),
    check (is_preferred in (0, 1))
);

create index instance_endpoints_instance_id_idx
    on instance_endpoints (instance_id);

create index instance_endpoints_address_id_idx
    on instance_endpoints (address_id);

create unique index instance_endpoints_name_idx
    on instance_endpoints (instance_id, name);

create unique index instance_endpoints_preferred_idx
    on instance_endpoints (instance_id)
    where is_preferred = 1;

-- +goose StatementBegin
create trigger addresses_endpoint_machine_update
before update of machine_id, area_id, asset_id on addresses
when exists (
    select 1 as result
    from instance_endpoints
    inner join instances on instance_endpoints.instance_id = instances.id
    where instance_endpoints.address_id = new.id
        and (new.machine_id is null or new.machine_id != instances.machine_id)
)
begin
    select raise(abort, 'endpoint address must belong to the instance machine') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger instances_endpoint_machine_update
before update of machine_id on instances
when exists (
    select 1 as result
    from instance_endpoints
    inner join addresses on instance_endpoints.address_id = addresses.id
    where instance_endpoints.instance_id = new.id
        and (addresses.machine_id is null or addresses.machine_id != new.machine_id)
)
begin
    select raise(abort, 'endpoint address must belong to the instance machine') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger instance_endpoints_machine_insert
before insert on instance_endpoints
when not exists (
    select 1 as result
    from instances
    inner join addresses on instances.machine_id = addresses.machine_id
    where instances.id = new.instance_id
        and addresses.id = new.address_id
)
begin
    select raise(abort, 'endpoint address must belong to the instance machine') as result;
end;
-- +goose StatementEnd

-- +goose StatementBegin
create trigger instance_endpoints_machine_update
before update of instance_id, address_id on instance_endpoints
when not exists (
    select 1 as result
    from instances
    inner join addresses on instances.machine_id = addresses.machine_id
    where instances.id = new.instance_id
        and addresses.id = new.address_id
)
begin
    select raise(abort, 'endpoint address must belong to the instance machine') as result;
end;
-- +goose StatementEnd

-- +goose Down

drop table instance_endpoints;
drop table instances;
drop table service_logos;
drop table services;
drop table addresses;
drop table networks;
drop table machine_users;
drop table machines;
drop table purchase_links;
drop table purchase_assets;
drop table purchases;
drop table asset_links;
drop table assets;
drop table product_links;
drop table product_port_profiles;
drop table rack_specs;
drop table drive_specs;
drop table memory_specs;
drop table processor_specs;
drop table products;
drop table manufacturers;
drop table areas;
drop table machine_provider_logos;
drop table machine_providers;
