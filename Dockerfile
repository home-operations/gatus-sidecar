ARG GO_VERSION
FROM golang:${GO_VERSION}-alpine AS builder
ARG VERSION=dev
ARG REVISION=dev
WORKDIR /src
RUN apk add --no-cache upx
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=${VERSION} -X main.Gitsha=${REVISION}" -trimpath -o /out/gatus-sidecar ./cmd/gatus-sidecar
RUN upx --best --lzma /out/gatus-sidecar

FROM scratch
COPY --from=builder /out/gatus-sidecar /gatus-sidecar
ENTRYPOINT ["/gatus-sidecar"]
