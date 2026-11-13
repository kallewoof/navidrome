-- +goose Up
-- +goose StatementBegin
-- How a user's listens ended, counted next to play_count: a skip stopped before the play-count
-- threshold, a completion at 90% of the song or later.
alter table annotation add column skip_count integer default 0 not null;
alter table annotation add column complete_count integer default 0 not null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table annotation drop column complete_count;
alter table annotation drop column skip_count;
-- +goose StatementEnd
