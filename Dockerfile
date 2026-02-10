FROM golang:1.26-alpine AS builder
WORKDIR /src
RUN apk add --no-cache upx
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o bin/gatus-sidecar cmd/root.go
RUN upx --best --lzma bin/gatus-sidecar

FROM scratch
COPY --from=builder /src/bin/gatus-sidecar /gatus-sidecar
ENTRYPOINT ["/gatus-sidecar"]
