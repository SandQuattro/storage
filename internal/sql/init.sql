-- Ups!
create table public.files
(
    id            bigserial constraint files_pk primary key,
    file_name     text not null,
    upload_status text not null,
    storage_link  text not null
);

-- Downs!
drop table if exists public.files;