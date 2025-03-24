# Development
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS development
ARG GOPRIVATE
ARG GITHUB_TOKEN
ENV GO111MODULE=on
WORKDIR /go/src/github.com/tidepool-org/tide-whisperer
RUN adduser -D mdblp && \
    apk add --no-cache gcc musl-dev git tzdata && \
    chown -R mdblp /go/src/github.com/tidepool-org/tide-whisperer
ARG TARGETPLATFORM
ARG BUILDPLATFORM
COPY --chown=mdblp . .
RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/" && \
    git config --global --add safe.directory /go/src/github.com/tidepool-org/tide-whisperer && \
    git config --global --add safe.directory /go/src/github.com/mdblp/tide-whisperer-v2 && \
    go env -w GOCACHE=/go-cache
RUN --mount=type=cache,target=/go-cache \
    --mount=type=cache,target=/go/pkg/mod/ \
    ./qa/build.sh $TARGETPLATFORM \
CMD ["./dist/tide-whisperer"]

# Production
FROM --platform=$BUILDPLATFORM alpine:latest AS production
WORKDIR /home/tidepool
RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache ca-certificates && \
    adduser -D tidepool
USER tidepool
COPY --from=development --chown=tidepool /go/src/github.com/tidepool-org/tide-whisperer/dist/tide-whisperer .
COPY --from=development /usr/share/zoneinfo /usr/share/zoneinfo
CMD ["./tide-whisperer"]
