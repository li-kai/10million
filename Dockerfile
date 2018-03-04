FROM golang:alpine

RUN mkdir -p /go/src/server
WORKDIR /go/src/server

COPY . /go/src/server
RUN go build -o bin *.go

FROM alpine:latest

WORKDIR /root/
COPY --from=0 /go/src/server/bin .

CMD ["./bin"]
