FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.16.3-alpine3.13@sha256:6c35b28ceee082c621c818c097f418cd55104a16e51003120c1bf37111ed9cfe as builder

ARG VERSION
ARG COMMIT
ARG DATE

ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}
ENV GO111MODULE=on

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
RUN go mod verify

COPY main.go main.go

COPY pkg/ pkg/

RUN go build \
    -a \
    -ldflags='-s -w -extldflags "-static" \
    -X main.version=${VERSION} \
    -X main.commit=${COMMIT} \
    -X main.date=${DATE} \
    -X main.builtBy=Docker' \
    -o /usr/bin/main .

FROM --platform=${BUILDPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot

WORKDIR /app

COPY --from=builder /usr/bin/main .

USER nonroot:nonroot

ENTRYPOINT ["./main"]