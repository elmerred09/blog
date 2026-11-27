-- +goose Up
-- The tag chip's color, by palette name. The web app maps each name to its
-- tint and text color, so the palette can be retuned without touching rows.
-- A CHECK rather than an ENUM: a later migration can drop a name from it.
-- The constant default is metadata only (no table rewrite); the CHECK scans
-- tags once, a small table.
ALTER TABLE tags
    ADD COLUMN color TEXT NOT NULL DEFAULT 'slate',
    ADD CONSTRAINT tags_color_check CHECK (
        color IN ('slate', 'rose', 'orange', 'amber', 'green', 'teal', 'violet', 'pink')
    );

-- +goose Down
ALTER TABLE tags
    DROP CONSTRAINT tags_color_check,
    DROP COLUMN color;
