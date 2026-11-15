-- +goose Up
-- +goose StatementBegin
-- Users' corrections of song tags, applied by the scanner over the file's own tags (model.TagOverlay). Keyed by
-- media file ID, which survives rescans and moves; tags is a JSON object of raw tag name to values.
create table tag_overlay (
    media_file_id varchar not null primary key,
    tags text not null,
    updated_at datetime not null,
    updated_by varchar not null default ''
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table tag_overlay;
-- +goose StatementEnd
