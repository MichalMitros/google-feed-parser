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
infra:             ## runs infrastructure (rabbitmq, postgres)
	@docker compose up -d rabbitmq -d postgres && \
	sleep 2 && \
	goose -dir $(migrations_dir) postgres "postgres://$(local_db_creds)@$(local_db_addr)/postgres?sslmode=disable" up

# testing

.PHONY: test
test: unit ## run all tests

.PHONY: unit
unit:              ## run unit tests
	@docker compose run --rm google-feed-parser go test -race -count=1 -run Unit ./...

# migrations

.PHONY: new-migration
new-migration:     ## create new migration
	@echo "Enter migration name:";\
	read mig_name;\
	goose -dir $(migrations_dir) create $$mig_name sql

.PHONY: migration-up
migration-up:      ## migrate the DB to the most recent version available
	@goose -dir $(migrations_dir) postgres "postgres://$(local_db_creds)@$(local_db_addr)/postgres?sslmode=disable" up

.PHONY: migration-down
migration-down:    ## rollback the version by 1
	@goose -dir $(migrations_dir) postgres "postgres://$(local_db_creds)@$(local_db_addr)/postgres?sslmode=disable" down

.PHONY: migrations-reset
migrations-reset:  ## rollback all migrations
	@goose -dir $(migrations_dir) postgres "postgres://$(local_db_creds)@$(local_db_addr)/postgres?sslmode=disable" reset

.PHONY: migrations-validate
migrations-validate: ## check migration files without running them
	@if ! goose -dir $(migrations_dir) validate; then \
		exit 1; \
	else \
		echo "all migration files are valid"; \
	fi;

.PHONY: install-goose
install-goose:     ## install goose@v3.20.0 for managing migrations
	@go install github.com/pressly/goose/v3/cmd/goose@v3.20.0

# code generation

.PHONY: generate
generate: infra     ## generate all the mocks and models for the project
	@@go generate ./...

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
