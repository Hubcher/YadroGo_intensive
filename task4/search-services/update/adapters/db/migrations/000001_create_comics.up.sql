--- сущность комикса (id из xkcd, ссылочка на картинку и тд)
create table if not exists comics (
    id integer primary key,
    img_url text not null,
    title text,
    alt text,
    created_at timestamptz not null default now()
);
--- сущность для нормализованных слов
create table if not exists words (
    id bigserial primary key,
    word text not null unique
);
--- таблица "многие ко многим"
create table if not exists comic_words (
    comic_id integer not null references comics(id) on delete cascade,
    word_id bigint not null references words(id) on delete cascade,
    primary key (comic_id, word_id)
);

create index if not exists idx_comic_words_comic_id on comic_words(comic_id);
create index if not exists idx_comic_words_word_id on comic_words(word_id);

