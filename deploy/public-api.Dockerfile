FROM golang:1.25.7-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/public-api ./apps/public-api

FROM alpine:3.22

RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /out/public-api /usr/local/bin/public-api
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/public-api"]
