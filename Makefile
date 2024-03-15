# default task since it's first
.PHONY: all
all: build test

BINARY = go-testcov
$(BINARY): *.go go.mod go.sum
	go build -o $(BINARY)

.PHONY: build
build: $(BINARY) ## Build binary

.PHONY: test
test: build ## Unit test
	cd test && ../$(BINARY)

install: ## Install binary
	go install
