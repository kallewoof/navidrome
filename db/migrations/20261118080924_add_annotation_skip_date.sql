-- +goose Up
-- +goose StatementBegin
-- When the user last skipped the item, next to play_date: together they say when it was last heard, which
-- Smart Shuffle uses to keep songs heard on any device out for a while.
alter table annotation add column skip_date datetime;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table annotation drop column skip_date;
-- +goose StatementEnd
