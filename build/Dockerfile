# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG GITVERSION=unknown
ARG GITCOMMIT=unknown
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags "-w -X github.com/modelpack/modctl/pkg/version.GitVersion=${GITVERSION} \
                   -X github.com/modelpack/modctl/pkg/version.GitCommit=${GITCOMMIT} \
                   -X github.com/modelpack/modctl/pkg/version.BuildTime=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
      -o modctl \
      main.go

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /app/modctl /usr/local/bin/modctl
ENTRYPOINT ["/usr/local/bin/modctl"]
