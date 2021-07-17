FROM golang:1.15-alpine AS build
WORKDIR /app
COPY / /app
RUN go build -o servicebin

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/servicebin /app
