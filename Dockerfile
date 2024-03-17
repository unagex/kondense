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

# build kubectl binary
FROM curlimages/curl:latest as kubectl
WORKDIR /tmp
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl

FROM alpine:latest
WORKDIR /
COPY --from=manager /manager .
COPY --from=kubectl /tmp/kubectl /usr/bin/kubectl

ENTRYPOINT ["/manager"]
