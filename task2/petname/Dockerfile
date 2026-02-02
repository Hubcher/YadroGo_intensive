FROM golang:1.25 AS build

RUN apt update && apt install -y protobuf-compiler
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
ENV PATH="$PATH:$(go env GOPATH)/bin"

COPY go.mod go.sum server.go /src/
COPY proto /src/proto

RUN cd /src && \
    protoc --go_out=.      --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/petname.proto


ENV CGO_ENABLED=0
RUN cd /src && go build -o /petname server.go

FROM alpine:3.20

COPY --from=build /petname /petname

ENTRYPOINT [ "/petname" ]
