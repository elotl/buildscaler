FROM golang:1.16 as builder

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY . .
RUN go mod download

# Run test & build binary
RUN make buildscaler

FROM debian:stable-slim

LABEL "Owner"="Elotl Inc."
LABEL "Description"="Buildscaler: CI platform integration for Kubernetes"

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update -y && \
        apt-get upgrade -y && \
        apt-get install -y ca-certificates

COPY --from=builder /build/buildscaler .

RUN chmod 755 /buildscaler

ENTRYPOINT ["/buildscaler"]
