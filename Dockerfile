FROM golang:1.11.3-stretch
COPY . /go/src/dproxy
WORKDIR /go/src/dproxy/cmd/dproxy
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' .
RUN CGO_ENABLED=0 GOOS=linux go install -a -ldflags '-extldflags "-static"'
ENTRYPOINT ["/go/bin/dproxy"]
