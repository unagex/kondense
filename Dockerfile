# syntax=docker/dockerfile:1.2

FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod go.sum ./
# Cache go modules
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd cmd
COPY pkg pkg

ARG GO_BUILD_FLAGS=""
RUN CGO_ENABLED=0 go build ${GO_BUILD_FLAGS} -o manager cmd/main.go

FROM alpine:latest
WORKDIR /
COPY --from=builder /app/manager .
COPY --from=d3fk/kubectl:latest /kubectl /usr/bin/kubectl

ENTRYPOINT ["/manager"]
