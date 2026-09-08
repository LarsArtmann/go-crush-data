-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages ADD COLUMN selftest_rogue TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE messages DROP COLUMN selftest_rogue;
-- +goose StatementEnd
