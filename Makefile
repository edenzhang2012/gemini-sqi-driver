IMAGE_NAME=gemini-sqi-driver
SERVER_DOCKERFILE=build/server/Dockerfile
CLIENT_DOCKERFILE=build/client/Dockerfile
VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
MAKEFLAGS += --jobs all
GO:=CGO_ENABLED=0 go
LDFLAGS:=--ldflags "\
		-X 'main.Version=$(VERSION)' \
		-X 'main.BuildNo=$(shell git rev-parse HEAD)' \
		-X 'main.BuildTime=$(shell date +'%Y-%m-%d %H:%M:%S')'"

default: vet build-bin

vet:
	go vet ./...

build-bin: vet
	@CGO_ENABLED=0 GOOS=linux go build -o main ${LDFLAGS} ./cmd/server/main.go
	@CGO_ENABLED=0 GOOS=linux go build -o client ${LDFLAGS} ./cmd/client/main.go
	
build-image: build-bin
	docker build -t $(IMAGE_NAME) -f $(SERVER_DOCKERFILE) .
	docker build -t $(IMAGE_NAME)-client -f $(CLIENT_DOCKERFILE) .