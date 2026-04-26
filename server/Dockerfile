# syntax=docker/dockerfile:1
FROM golang:1.23-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/manitor-server .

FROM gcr.io/distroless/static-debian12:latest
WORKDIR /data
ENV SERVER_CLIENT=:5000
COPY --from=build /out/manitor-server /manitor-server
EXPOSE 5000
ENTRYPOINT ["/manitor-server"]
