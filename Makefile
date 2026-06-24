.PHONY: test lint integration coverage clean vet tidy

test:
	go test -race -count=1 -timeout=120s github.com/Irishsmurf/pinergy-go

coverage:
	go test -race -count=1 -coverprofile=coverage.out -covermode=atomic github.com/Irishsmurf/pinergy-go
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

vet:
	go vet ./...

integration:
	go test -tags=integration -race -count=1 -timeout=300s -v github.com/Irishsmurf/pinergy-go

tidy:
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod/go.sum not tidy" && exit 1)

clean:
	rm -f coverage.out coverage.html

.DEFAULT_GOAL := test
