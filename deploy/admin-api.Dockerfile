FROM golang:1.25.7-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/admin-api ./apps/admin-api

FROM alpine:3.22

RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /out/admin-api /usr/local/bin/admin-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/admin-api"]
