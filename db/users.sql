drop type if exists user_row cascade;
drop trigger if exists take_log on users;

create type user_row as (
    user_id int,
    email citext,
    username citext,
    total_donation numeric(10, 2),
    image citext,
    is_admin boolean
);

create or replace function all_users(_num_per_page int, _idx_page int)
returns setof user_row as $$
declare
    usr user_row%rowtype;
    usr_row_cursor refcursor;
    i int;
begin
    insert into logs(content, log_level)
        values ('Select all users', 1);
    open usr_row_cursor for
        select user_id, email, username, total_donation, image, is_admin
        from users
        order by user_id;
    move absolute (_idx_page - 1) * _num_per_page from usr_row_cursor;
    i := 0;
    loop
        if i >= _num_per_page then
            exit;
        end if;
        i := i + 1;
        fetch usr_row_cursor into usr;
        exit when not found;
        return next usr;
    end loop;
    close usr_row_cursor;
    return;
end
$$ language plpgsql;

create or replace function get_users_quarter_donations()
returns numeric[] as $$
declare
    quarter_donation numeric;
    donations numeric array;
begin
    select percentile_disc(array[0,0.25,0.5,0.75,1])
    within group (order by total_donation)
    into donations
    from users;
    return donations;
end
$$ language plpgsql;

create type donor_row as (
    username citext,
    total_donation numeric(10, 2),
    image citext
);

create or replace function top_donors()
returns setof donor_row as $$
declare
    usr donor_row%rowtype;
    usr_row_cursor refcursor;
    i int;
begin
    insert into logs(content, log_level)
        values ('Select all users', 1);
    open usr_row_cursor for
        select username, total_donation, image
        from users
        order by total_donation desc
        limit 3;
    i := 0;
    loop
        i := i + 1;
        fetch usr_row_cursor into usr;
        exit when not found;
        return next usr;
    end loop;
    close usr_row_cursor;
    return;
end
$$ language plpgsql;

create or replace function get_user(_email citext, _password varchar(255))
returns user_row as $$
declare
    usr user_row%rowtype;
begin
    select user_id, email, username, total_donation, image, is_admin
        into usr
        from users
        where email = _email and password = crypt(_password, password);
    insert into logs(user_id, content, log_level)
        values (usr.user_id, 'Select user', 1);
    return usr;
end
$$ language plpgsql;

create or replace function create_user(
    _email citext,
    _password varchar(255),
    _username citext,
    _image citext)
returns integer as $$
    INSERT INTO users (email, password, username, image)
        VALUES(_email, crypt(_password, gen_salt('bf', 8)), _username, _image);
    select max(user_id)
        from users
$$ language sql;

create trigger take_log after insert or update or delete on users
for each row execute procedure create_log_user(' on users');
