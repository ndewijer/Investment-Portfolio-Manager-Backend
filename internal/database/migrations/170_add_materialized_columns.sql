-- +goose Up
ALTER TABLE fund_history_materialized ADD COLUMN sale_proceeds FLOAT NOT NULL DEFAULT 0;
ALTER TABLE fund_history_materialized ADD COLUMN original_cost FLOAT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE fund_history_materialized DROP COLUMN sale_proceeds;
ALTER TABLE fund_history_materialized DROP COLUMN original_cost;
