FROM golang:1.25 AS build

COPY . /src
RUN cd /src && CGO_ENABLED=0 go build -o /hello /src/server.go

FROM alpine:3.20

COPY --from=build /hello /hello

ENTRYPOINT [ "/hello" ]
