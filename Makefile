PLATFORM:=$(shell uname)

ifeq ($(PLATFORM), Darwin)
	BUILD_ENVPARMS:=GOOS=darwin GOARCH=arm64 CGO_ENABLED=0
endif
ifeq ($(PLATFORM), Linux)
	BUILD_ENVPARMS:=GOOS=linux GOARCH=amd64 CGO_ENABLED=0
endif

LDFLAGS:=-s -w

POSTGRES_DSN?=postgres://postgres:password@localhost:5432/tracker?sslmode=disable

.PHONY: build
build:
	@[ -d .build ] || mkdir -p .build
	@$(BUILD_ENVPARMS) go build -ldflags "$(LDFLAGS)" -o .build/app cmd/app/main.go

.PHONY: run
run:
	@DEBUG=true \
	HTTP_ADDR=:8080 \
	POSTGRES_DSN=$(POSTGRES_DSN) \
	MIGRATIONS_PATH=migrations/sql \
	go run cmd/app/main.go

.PHONY: compose-up
compose-up:
	docker compose up -d

.PHONY: compose-down
compose-down:
	docker compose down

.PHONY: compose-shutdown
compose-shutdown:
	docker compose down --volumes

.PHONY: test
test:
	go test ./...

.PHONY: test-integration
test-integration:
	TEST_POSTGRES_DSN=$(POSTGRES_DSN) go test -tags=integration ./...

.PHONY: swag-install
swag-install:
	go install github.com/swaggo/swag/v2/cmd/swag@v2.0.0-rc5

.PHONY: swag-init
swag-init:
	swag init --v3.1 -g cmd/app/main.go -o docs/swagger --parseInternal --parseDependency

.PHONY: tidy
tidy:
	go mod tidy
