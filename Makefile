.PHONY start:
start:
	go run cmd/gache/main.go

.PHONY test:
test:
	go test -v -race -timeout 30s ./...

.DEFAULT_GOAL: start