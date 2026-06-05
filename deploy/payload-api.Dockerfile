FROM golang:1.25.7-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/payload-api ./apps/payload-api

FROM alpine:3.22

RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /out/payload-api /usr/local/bin/payload-api
EXPOSE 8083
ENTRYPOINT ["/usr/local/bin/payload-api"]
