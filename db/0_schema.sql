create extension if not exists citext;
create extension if not exists pgcrypto;

drop table if exists users cascade;
drop table if exists categories cascade;
drop table if exists projects cascade;
drop table if exists payments cascade;
drop table if exists comments cascade;
drop table if exists logs cascade;
drop table if exists tags cascade;

create table if not exists users (
    user_id serial primary key,
    email citext unique not null,
    password varchar(255) not null check (length(password) >= 8),
    username citext not null,
    total_donation numeric(10, 2) not null default 0 check (total_donation >= 0),
    image citext not null default '',
    is_admin boolean not null default false
);

create table if not exists categories (
    name citext unique primary key,
    proj_num integer not null default 0 check (proj_num >= 0)
);

create table if not exists projects (
    project_id serial primary key,
    title varchar(100) not null,
    user_id integer not null references users (user_id) on update cascade on delete cascade,
    category citext not null references categories (name) on update cascade on delete cascade,
    description text not null,
    verified boolean not null default false,
    image citext,
    amount_raised numeric(10, 2) not null check (amount_raised >= 0) default 0, --10 sf, 2dp--
    amount_required numeric(10, 2) not null check (amount_required > 0),
    start_time timestamp not null default now(),
    end_time timestamp not null check (start_time <= end_time),
    constraint valid_amount check (amount_raised <= amount_required)
);

create table if not exists tags(
    project_id integer not null references projects (project_id) on update cascade on delete cascade,
    tag_name citext not null,
    primary key(project_id, tag_name)
);

create or replace function ensure_valid_project(_project_id int, _payment_time timestamp)
returns boolean
as $$
declare
    is_verified boolean;
    project_end timestamp;
begin
    select verified, end_time
    into is_verified, project_end
    from projects
    where project_id = _project_id;

    return is_verified = 't' and project_end >= _payment_time;
end
$$ language plpgsql;

create table if not exists payments (
    id serial primary key,
    user_id integer not null references users (user_id) on update cascade on delete cascade,
    project_id integer not null references projects (project_id) on update cascade on delete cascade,
    moment timestamp not null default now(),
    amount numeric(10, 2) not null check (amount > 0), --10 sf, 2dp--
    constraint valid_payment check (
        ensure_valid_project(project_id, moment) = 't'
    )
);

create table if not exists comments (
    id serial primary key,
    user_id integer not null references users (user_id) on update cascade on delete cascade,
    project_id integer not null references projects (project_id) on update cascade on delete cascade,
    moment timestamp not null default now(),
    content text not null
);

create table if not exists logs (
    id serial primary key,
    user_id integer default null,
    project_id integer default null,
    moment timestamp not null default now(),
    content text not null,
    log_level integer not null
);

-- create view whole_project_info as
--     select p.project_id, p.title, p.description,
--         p.start_time, p.end_time,
--         p.amount_required, coalesce(sum(f.amount), 0) as amount_raised,
--         p.verified, p.category, p.user_id
--     from projects p left outer join payments f
--     on p.project_id = f.project_id
--     group by p.project_id, p.user_id;

-- -- SOURCE: https://www.meetspaceapp.com/2016/04/12/passwords-postgresql-pgcrypto.html
-- PREPARE create_user (text, varchar) AS
--     INSERT INTO users (email, password)
--     VALUES($1, crypt($2, gen_salt('bf', 8)));

-- PREPARE select_user (text, varchar) AS
--     SELECT id, email, is_admin FROM users
--     WHERE email = $1
--     AND password = crypt($2, password);
