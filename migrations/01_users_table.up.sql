
create table if not exists users (
  id                text primary key,
  email             text,
  email_verified    boolean not null default false,
  username      	text not null,
  avatar_url        text,
  password_hash     text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists users_email_not_null_unique on users (email) where email is not null;
