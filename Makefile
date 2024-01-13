BIN_DIR=$(shell go env GOPATH)/bin

MIGRATE=$(BIN_DIR)/migrate
SQLC=$(BIN_DIR)/sqlc
SWAG=$(BIN_DIR)/swag

POSTGRESQL_DSN ?= postgres://metagaming:metagaming@localhost:5432/metagaming?sslmode=disable

test:			## Run unit tests
	@go test ./...

api-docs:		## Generate API Docs
	@test ! -f $(SWAG) && go install github.com/swaggo/swag/cmd/swag@latest || true
	@$(SWAG) init --dir internal/controller/rest --output internal/controller/rest/docs --generalInfo router.go

db-queries:		## Generate SQLC code.
	@test ! -f $(SQLC) && go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest || true
	@$(SQLC) --file internal/infra/storage/postgres/internal/sqlc.yaml generate

db-migration:	## Run database migrations. 
	@test ! -f $(MIGRATE) && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest || true
	@read -p 'Migration name: ' MIGRATION_NAME && $(MIGRATE) create -ext sql -dir internal/infra/storage/postgres/internal/migrations -seq $$MIGRATION_NAME

db-migrate:		## Apply the migrations to the database
	@test ! -f $(MIGRATE) && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest || true
	@$(MIGRATE) -path internal/infra/storage/postgres/internal/migrations -database "${POSTGRESQL_DSN}" up
