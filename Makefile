export AWS_PROFILE ?= process

.PHONY: build test deploy local

build:
	sam build

test:
	go test ./...

deploy:
	sam deploy --profile $(AWS_PROFILE)

local:
	./scripts/local.sh

build-TasksFunction:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/bootstrap ./cmd/tasks-api
