FROM golang:1.11

COPY . /go/src/platform-connector
WORKDIR /go/src/platform-connector

ENV GO111MODULE=on

RUN go build

EXPOSE 8080

ENTRYPOINT ["/go/src/platform-connector/platform-connector"]