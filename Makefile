AWS_PROFILE ?= default

export AWS_PROFILE

.PHONY: build test deploy local

build:
	sam build

test:
	go test ./...

deploy: build
	sam deploy --profile $(AWS_PROFILE)

local:
	./scripts/local.sh

build-TasksFunction:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/tasks-api
