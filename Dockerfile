# Build the manager binary
FROM golang:1.21 as manager

WORKDIR /

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY cmd cmd
COPY pkg pkg

RUN CGO_ENABLED=0 go build -a -o manager cmd/main.go

FROM d3fk/kubectl:latest as kubectl

FROM alpine:latest
WORKDIR /
COPY --from=manager /manager .
COPY --from=kubectl /kubectl /usr/bin/kubectl

ENTRYPOINT ["/manager"]
