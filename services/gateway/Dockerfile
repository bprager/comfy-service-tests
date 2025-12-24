FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/gateway ./cmd/gateway

FROM alpine:3.19
RUN adduser -D -H app
USER app
COPY --from=build /out/gateway /usr/local/bin/gateway
EXPOSE 8084
ENTRYPOINT ["/usr/local/bin/gateway"]
