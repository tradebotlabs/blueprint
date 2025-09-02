GOPATH:=$(shell go env GOPATH)

.PHONY: init
init:
		
	@go get -u google.golang.org/protobuf@v1.26.0 
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest	
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	
.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/blueprint/blueprint.proto

.PHONY: update
update:
	@go get -u

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: build
build:
	@go build -o blueprint-srv cmd/*.go

.PHONY: test
test:
	@go test -v ./... -cover

.PHONY: docker
docker:
	@docker build -t blueprint:latest .	

.PHONY: push
docker:
	@docker build -t blueprint:latest .
