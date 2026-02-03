FROM golang:1.24

COPY . /tests

WORKDIR /tests

ENTRYPOINT [ "go", "test", "-race", "-v", "./..." ]
