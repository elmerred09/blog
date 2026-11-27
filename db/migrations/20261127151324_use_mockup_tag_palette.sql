-- +goose Up
-- The tag palette becomes the devTalk mockup's fifteen colors plus slate,
-- the neutral default. Names it drops move to the nearest new one. Both
-- statements read all of tags, a small table.
ALTER TABLE tags DROP CONSTRAINT tags_color_check;

UPDATE tags
SET color = CASE color
        WHEN 'rose' THEN 'coral'
        WHEN 'orange' THEN 'rust'
        WHEN 'violet' THEN 'purple'
    END
WHERE color IN ('rose', 'orange', 'violet');

ALTER TABLE tags ADD CONSTRAINT tags_color_check CHECK (
    color IN (
        'slate', 'coral', 'rust', 'amber', 'yellow', 'olive', 'brown', 'forest',
        'green', 'teal', 'petrol', 'blue', 'navy', 'indigo', 'purple', 'pink'
    )
);

-- +goose Down
-- Back to the first palette: the three renamed colors return, and any other
-- color the first palette lacks becomes slate.
ALTER TABLE tags DROP CONSTRAINT tags_color_check;

UPDATE tags
SET color = CASE color
        WHEN 'coral' THEN 'rose'
        WHEN 'rust' THEN 'orange'
        WHEN 'purple' THEN 'violet'
        ELSE 'slate'
    END
WHERE color NOT IN ('slate', 'amber', 'green', 'teal', 'pink');

ALTER TABLE tags ADD CONSTRAINT tags_color_check CHECK (
    color IN ('slate', 'rose', 'orange', 'amber', 'green', 'teal', 'violet', 'pink')
);
