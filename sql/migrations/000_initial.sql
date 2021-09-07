-- +migrate Up
--drop table securities;
create table if not exists securities
(
    id         bigserial         not null,
    name       text              not null,
    isin       text              not null,
    wkn        text              not null,
    underlying text  default ''  not null,
    type                integer default 0 not null,
    warrant_type        integer default 0 not null,
    warrant_sub_type    integer default 0 not null,
    updated_at date              not null,
    created_at date              not null,
    constraint securities_pkey
        primary key (id),
    constraint securities_isin_key
        unique (isin)
);

--drop table info_links;
create table if not exists info_links
(
    wkn        text not null,
    url        text not null,
    created_at date not null,
    updated_at date not null,
    constraint info_links_pk
        primary key (wkn),
    constraint info_links_wkn_url
    unique (url)
);
