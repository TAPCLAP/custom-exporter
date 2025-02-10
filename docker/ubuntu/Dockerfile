FROM docker.io/golang:1.23 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go ./
COPY ./pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./custom-exporter .

FROM docker.io/ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates \
    && apt-get autoclean \
    && apt-get --purge autoremove \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /app/custom-exporter /custom-exporter
ENTRYPOINT ["/custom-exporter"]

