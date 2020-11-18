export GO111MODULE=on
export PATH:=/usr/local/go/bin:$(PATH)


default: test
test: vet acceptance

fmt:
	@echo Running go fmt
	@go fmt

lint:
	@echo Running go lint
	@golint --set_exit_status ./...

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

acceptance:
	@echo "Starting acceptance tests..."
	@go test ./... -race -covermode=atomic -coverprofile=coverage.txt -timeout 60m -parallel=2 -v
