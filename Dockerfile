FROM golang:1.15.5-alpine3.12@sha256:072f74098dd1e4e8e1c05102aa2571c1f5a4c307f3b9cdc9e0ed9f6ed5b37ef6 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
RUN go mod verify

COPY main.go main.go

COPY testdata/ testdata/
COPY pkg/ pkg/

RUN go build \
    -a \
    -ldflags='-w -s -extldflags "-static"' \
    -o /usr/bin/main .

FROM gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /usr/bin/main .

USER nonroot:nonroot

ENTRYPOINT ["./main"]