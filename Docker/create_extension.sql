CREATE EXTENSION IF NOT EXISTS vector;
create table items (id serial primary key, doc text, embedding vector(4096));
