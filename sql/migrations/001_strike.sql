-- +migrate Up
alter table securities
    add strike float default 0 not null;
alter table securities
    add expire date default null;


-- +migrate Down
alter table securities
    drop column strike;
alter table securities
    drop column expire;