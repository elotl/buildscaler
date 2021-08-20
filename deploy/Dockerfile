FROM alpine:3.9
RUN apk update && apk add curl ca-certificates
COPY adapter /
ENTRYPOINT ["/adapter", "--logtostderr=true"]
