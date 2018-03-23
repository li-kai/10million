-- Sql functions and triggers for categories

----  all_categories()
-- usage : select * from all_categories();

---- create_categories(name citext)
-- usage select create_categories('example'); 

create type cate_row as (
    name citext
);

create or replace function all_categories()
returns setof cate_row as $$
declare
    cate cate_row%rowtype;
begin
    for cate in
        select name
        from categories
    loop
        return next cate;
    end loop;
    return;
end
$$ language plpgsql;

create or replace function create_categories(name citext)
returns text as $$
    insert into categories
        values(name);
    select 'Insert OK';
$$ language sql;