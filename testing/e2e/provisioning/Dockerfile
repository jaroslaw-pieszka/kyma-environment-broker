FROM --platform=$BUILDPLATFORM golang:1.24.5-alpine3.22 AS builder

ENV SRC_DIR=/go/src/github.com/kyma-project/kyma-environment-broker/tests/e2e/provisioning

WORKDIR $SRC_DIR
COPY . $SRC_DIR

ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go test -ldflags="-s -w" -c ./test

FROM europe-docker.pkg.dev/kyma-project/prod/external/alpine:3.19.1
LABEL source=git@github.com:kyma-project/kyma-environment-broker.git

WORKDIR /app

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /go/src/github.com/kyma-project/kyma-environment-broker/tests/e2e/provisioning/test.test .

ENTRYPOINT ["/app/test.test"]
