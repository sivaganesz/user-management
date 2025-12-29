# Makefile

MAIN = cmd/api/main.go
APP = myapp

.PHONY: run build test clean install

# Run the application
run:
	go run $(MAIN)

# Install dependencies
install:
	go mod download
	go mod tidy
