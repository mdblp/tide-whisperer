# Development
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS development
ARG GOPRIVATE
ARG GITHUB_TOKEN
ENV GO111MODULE=on
WORKDIR /go/src/github.com/tidepool-org/tide-whisperer
RUN adduser -D mdblp && \
    apk add --no-cache git tzdata && \
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
FROM gcr.io/distroless/static:nonroot AS production
WORKDIR /home/nonroot
USER nonroot
COPY --from=development --chown=nonroot /go/src/github.com/mdblp/tide-whisperer/dist/tide-whisperer .
CMD ["./tide-whisperer"]
CMD ["./tide-whisperer"]
