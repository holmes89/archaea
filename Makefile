.PHONY: test vet lint hooks

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit .githooks/pre-push
