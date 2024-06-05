pwd = $(shell pwd)
local_db_addr = "localhost:15432"
local_db_creds = "local:local"
migrations_dir = "migrations"

# help - list of available commands

.PHONY: help
help:              ## show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

# build / run / stop

.PHONY: clean
clean: stop

.PHONY: stop
stop:              ## stop containers
	@docker compose down --remove-orphans

.PHONY: build
build:             ## build docker compose image (e.g. to rebuild image cache)
	@docker compose build

.PHONY: run
run:               ## runs google-feed-parser and all necessary components
	@docker compose up --build google-feed-parser

.PHONY: infra
infra:             ## runs infrastructure (rabbitmq, postgres) and migrations
	@docker compose up -d rabbitmq -d postgres -d migrator

# testing

.PHONY: test
test: unit integration ## run all tests

.PHONY: unit
unit:              ## run unit tests
	@docker compose run --rm google-feed-parser go test -race -count=1 -run Unit ./...

.PHONY: integration
integration:       ## run integration tests
	@docker compose run --rm google-feed-parser go test -race -count=1 -run Integration ./...

# migrations

.PHONY: new-migration
new-migration:     ## create new migration
	@echo "Enter migration name:";\
	read mig_name;\
	goose -dir $(migrations_dir) create $$mig_name sql

.PHONY: migration-up
migration-up:      ## migrate the DB to the most recent version available
	@docker compose run --rm migrator sh -c 'goose -dir $(migrations_dir) postgres $${DATABASE_URL} up'

.PHONY: migration-down
migration-down:    ## rollback the version by 1
	@docker compose run --rm migrator sh -c 'goose -dir $(migrations_dir) postgres $${DATABASE_URL} down'

.PHONY: migration-reset
migration-reset:  ## rollback all migrations
	@docker compose run --rm migrator sh -c 'goose -dir $(migrations_dir) postgres $${DATABASE_URL} reset'

.PHONY: migrations-validate
migration-validate: ## check migration files without running them
	@if ! docker compose run --rm migrator sh -c 'goose -dir $(migrations_dir) validate'; then \
		exit 1; \
	else \
		echo "all migration files are valid"; \
	fi;

# code generation

.PHONY: generate
generate: infra     ## generate all the mocks and models for the project
	@@go generate ./...

.PHONY: generate-db
generate-db: infra ## generate golang models from database
	@dns="postgresql://$(local_db_creds)@$(local_db_addr)/postgres?sslmode=disable"; \
	path="./internal/platform/storage/gen"; \
	goose_table_name="goose_db_version"; \
	jet -dsn=$$dns -ignore-tables=$$goose_table_name -path=$$path

# Linters and formatters

.PHONY: golangci
golangci:          ## run golangci-lint
	@docker run --rm \
		-v $(pwd):/src \
		-w /src \
		golangci/golangci-lint:v1.57.2-alpine golangci-lint run

.PHONY: format
format:            ## format golang code using gofumpt
	@img_tag="gofumpt-060-go-1222:0.0.1"; \
	if ! docker image inspect $$img_tag > /dev/null; then \
		docker build -t $$img_tag -f gofumpt.Dockerfile .; \
	fi; \
	echo "formatting using gofumpt..."; \
	docker run --rm \
		-v $(pwd):/src \
		-w /src \
		$$img_tag gofumpt -l -w .; \
	echo "done"

# jet 

.PHONY: install-jet
install-jet:       ## install jet v2.11.1 for generating models from DB schema.
	@go install github.com/go-jet/jet/v2/cmd/jet@v2.11.1
