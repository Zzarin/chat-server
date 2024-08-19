-- +goose Up
create table chats (
    id bigserial primary key,
    users text[] not null,
    created_at timestamptz default now() not null
);

create table messages (
    id int,
    sender text not null,
    message_text text not null,
    posted_at timestamptz default now() not null
);

create index if not exists messages_id_idx on "messages"(id);

-- +goose Down
drop table "chats";
drop table "messages";
