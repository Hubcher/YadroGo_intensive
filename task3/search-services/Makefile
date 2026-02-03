lint: protolint golint

protobuf:
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/words/words.proto

protolint:
	protolint .

golint:
	golangci-lint run -E gocritic -v ./...
