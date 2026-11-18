-- +goose Up
-- +goose StatementBegin
-- How the client that saved the queue plays it (model.PlayQueue.ClientState): a JSON object the server stores and
-- returns without reading, so another device that adopts the queue can play it the same way.
alter table playqueue add column client_state text not null default '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table playqueue drop column client_state;
-- +goose StatementEnd
