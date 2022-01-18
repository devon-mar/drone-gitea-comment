FROM --platform=$BUILDPLATFORM golang:1.17 as builder
ARG TARGETOS TARGETARCH

WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/src/app/drone-gitea-comment /bin/drone-gitea-comment
ENTRYPOINT ["/bin/drone-gitea-comment"]