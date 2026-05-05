# syntax=docker/dockerfile:1
FROM hub.hamdocker.ir/golang:1.23-bookworm AS build
WORKDIR /src
RUN go env -w GOPROXY=http://5.63.13.218:8081/repository/Aliii/
RUN go env -w GOSUMDB=*
RUN go env | grep GOPROXY

COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/manitor-server .

FROM hub.hamdocker.ir/alpine:latest
WORKDIR /data
ENV SERVER_CLIENT=:5000
COPY --from=build /out/manitor-server /manitor-server
EXPOSE 5000
ENTRYPOINT ["/manitor-server"]
