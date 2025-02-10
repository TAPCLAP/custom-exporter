FROM docker.io/golang:1.23 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go ./
COPY ./pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./custom-exporter .

FROM docker.io/alpine:3.20.1 AS certificates
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=certificates /etc/ssl /etc/ssl
COPY --from=build /app/custom-exporter /custom-exporter
ENTRYPOINT ["/custom-exporter"]