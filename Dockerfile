FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY external ./external
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/nodo-api ./cmd/api

FROM alpine:3.22

RUN addgroup -S nodo && adduser -S -G nodo nodo
COPY --from=build /out/nodo-api /usr/local/bin/nodo-api

USER nodo
EXPOSE 8080
ENTRYPOINT ["nodo-api"]
