deps:
	rm -f ./build/bin/buf
	curl -fsSL "https://github.com/bufbuild/buf/releases/download/v1.26.1/buf-$(shell uname -s)-$(shell uname -m)" -o ./build/bin/buf --create-dirs
	chmod +x ./build/bin/buf

tools:
	chmod +x ./build/bin
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	chmod +x ./build/bin/buf
	chmod +x ./build/bin/protoc-gen-go-grpc
	chmod +x ./build/bin/protoc-gen-go

init:
	@./build/bin/buf  buf mod init

builder:
	@./build/bin/buf build

gen: update
	@./build/bin/buf generate -v --config buf.yaml

update:
	@./build/bin/buf mod update

breaking: update build
	@./build/bin/buf breaking --against .

lint:
	@./build/bin/buf lint

build:
	go build .

run:
	go run .
