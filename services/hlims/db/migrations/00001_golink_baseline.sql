-- +goose Up

create table if not exists links (
    id       text    primary key,
    short    text    not null default '',
    long     text    not null default '',
    created  integer not null default (strftime('%s', 'now')),
    lastedit integer not null default (strftime('%s', 'now')),
    owner    text    not null default ''
);

create table if not exists stats (
    id      text    not null default '',
    created integer not null default (strftime('%s', 'now')),
    clicks  integer
);

-- +goose Down

drop table stats;
drop table links;
