FROM golang:1.24.2 AS build

WORKDIR /opt/app

COPY go.mod go.sum ./

RUN go mod download

COPY api api
COPY cmd cmd
COPY internal internal

RUN GOOS=linux go build -ldflags "-linkmode external -extldflags '-static' -s -w" \
    -o /opt/app/cmd/server \
    /opt/app/cmd/server

FROM gcr.io/distroless/static-debian12
WORKDIR /opt

COPY --from=build --chown=nonroot:nonroot /opt/app/cmd/server .

USER nonroot
ENTRYPOINT ["/opt/server"]

EXPOSE 7654