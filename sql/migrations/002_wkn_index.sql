-- +migrate Up
create index securities_wkn_index
    on securities (wkn);

-- +migrate Down
drop index securities_wkn_index;
