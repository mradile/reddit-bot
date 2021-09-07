VERSION := `git describe --tags`
BINARY_NAME = mswkn
NOW = `date +"%Y-%m-%d_%H-%M-%S"`
MAIN_GO_PATH=cmd/${BINARY_NAME}/main.go
DOCKER_IMAGE=registry.gitlab.com/mswkn/bot:snapshot
PKG = gitlab.com/mswkn/bot


all: clean gen check staticcheck test build build-linux
check: vet lint staticcheck

.PHONY: build
build:
	CGO_ENABLED=0 go build -v -o release/${BINARY_NAME} -ldflags="-X main.version=${VERSION}" ${MAIN_GO_PATH}

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o release/linux-amd64/${BINARY_NAME} -ldflags="-X main.version=${VERSION}" ${MAIN_GO_PATH}

.PHONY: test
test: gen
	go test -race ./...

.PHONY: cover
cover: gen
	go test -race -coverprofile=cover.out ./...
	go tool cover -func=cover.out

.PHONY: vet
vet:
	go vet ./...

.PHONY: staticcheck
staticcheck:
	staticcheck ./...

.PHONY: lint
lint:
	revive -formatter friendly ./...

.PHONY: gen
gen:
	go generate ./...
	rice embed-go -i ${PKG}/pkg/db

.PHONY: sqlboiler
sqlboiler:
	sqlboiler --wipe psql

.PHONY: run
run:
	go  run -race ${MAIN_GO_PATH}

.PHONY: docker
docker:
	docker build -t ${DOCKER_IMAGE} .

.PHONY: docker-run
docker-run:
	docker run --rm -ti ${DOCKER_IMAGE}

.PHONY: docker-push
docker-push: docker
	docker push ${DOCKER_IMAGE}

.PHONY: clean
clean:
	rm -rf release/*
	rm -f cover.out
	go clean -testcache
	rice clean

