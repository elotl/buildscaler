FROM golang:1.16 as builder

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Run test & build binary
RUN make buildscaler

FROM debian:stable-slim

LABEL "Owner"="Elotl Inc."
LABEL "Description"="Buildscaler: CI platform integration for Kubernetes"

ARG DEBIAN_FRONTEND=noninteractive

COPY --from=builder /build/buildscaler .

RUN chmod 755 /buildscaler

ENTRYPOINT ["/buildscaler"]
