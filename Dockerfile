FROM golang:1.17.4 as builder

WORKDIR /go/src

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN go build \
    -o /go/bin/main \
    -ldflags '-s -w'


FROM scratch as runner

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/main /app/main
COPY config.yml /app/
WORKDIR /app

ENTRYPOINT ["/app/main"]