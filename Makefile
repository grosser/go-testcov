dev:
	@ls *.gotest | sed s/.gotest// | xargs -I {} -n 1 mv {}.gotest {}_test.go

undev:
	@ls *_test.go | sed s/_test.go// | xargs -I {} -n 1 mv {}_test.go {}.gotest

test:
	@make dev
	@go run main.go utils.go
	@make undev

integration:
	@make dev
	go test -race -cover -covermode=atomic
	go install
	go-testcov
	go vet
	[ -z "`go fmt`" ]
	@make undev
