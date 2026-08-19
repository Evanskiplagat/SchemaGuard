FROM golang:1.23 AS builder

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/schemaguard ./cmd/schemaguard

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/schemaguard /schemaguard

ENTRYPOINT ["/schemaguard"]
