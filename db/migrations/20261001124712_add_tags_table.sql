-- +goose Up
CREATE TABLE tags (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    slug TEXT NOT NULL UNIQUE CHECK (slug = lower(slug)),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE post_tags (
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

CREATE INDEX post_tags_tag_id_idx ON post_tags (tag_id, post_id);

-- +goose Down
DROP TABLE post_tags;
DROP TABLE tags;
