.PHONY: test test-ci coverage

test:
	go test ./...

coverage:
	mkdir -p reports
	go test ./... -covermode=atomic -coverprofile=reports/coverage.out
	go tool cover -func=reports/coverage.out

test-ci: coverage
	sh scripts/check-coverage.sh
