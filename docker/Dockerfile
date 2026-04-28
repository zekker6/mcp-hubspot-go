# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o /out/mcp-hubspot \
    ./cmd/mcp-hubspot

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/mcp-hubspot /mcp-hubspot

EXPOSE 8012

ENTRYPOINT ["/mcp-hubspot"]
