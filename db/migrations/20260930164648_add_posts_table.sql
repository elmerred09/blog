-- +goose Up
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    slug TEXT NOT NULL UNIQUE CHECK (slug = lower(slug)),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    body_md TEXT NOT NULL,
    body_html TEXT NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title,'')), 'A') ||
        setweight(to_tsvector('english', coalesce(summary,'')), 'B') ||
        setweight(to_tsvector('english', coalesce(body_md,'')), 'C')
    ) STORED,
    published_at TIMESTAMPTZ NULL,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX posts_search_idx ON posts USING GIN (search_vector);

CREATE INDEX posts_pagination_idx ON posts (published_at DESC, id DESC) WHERE published_at IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP TABLE posts;
