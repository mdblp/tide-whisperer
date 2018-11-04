# Development
FROM golang:1.9.2-alpine AS development

WORKDIR /go/src/github.com/tidepool-org/tide-whisperer

COPY . .

RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add build-base git cyrus-sasl-dev

RUN  dos2unix build.sh && ./build.sh

RUN  ./build.sh

CMD ["./dist/tide-whisperer"]

# Release
FROM alpine:latest AS release

RUN ["apk", "add", "--no-cache", "ca-certificates", "libsasl"]

RUN ["adduser", "-D", "tidepool"]

WORKDIR /home/tidepool

USER tidepool

COPY --from=development --chown=tidepool /go/src/github.com/tidepool-org/tide-whisperer/dist/tide-whisperer .

CMD ["./tide-whisperer"]
