FROM golang:latest

RUN apt-get update
RUN apt-get install vim -y

ADD . /go/src/github.com/jumpcloud
RUN go install github.com/jumpcloud

EXPOSE 8080
ENTRYPOINT /go/bin/jumpcloud