-- +goose Up
-- The count of whitespace-separated tokens in the markdown, so code and
-- markup count as words. The API turns it into reading minutes.
ALTER TABLE posts
    ADD COLUMN word_count integer NOT NULL GENERATED ALWAYS AS (regexp_count(body_md, '\S+')) STORED;

-- +goose Down
ALTER TABLE posts DROP COLUMN word_count;
