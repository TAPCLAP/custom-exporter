FROM --platform=$BUILDPLATFORM golang:1.22 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go ./
COPY ./pkg ./pkg

ARG TARGETPLATFORM
RUN echo "Building BUILDPLATFORM: '${BUILDPLATFORM}', TARGETPLATFORM: '$TARGETPLATFORM'"; \
    export GOOS=$(echo $TARGETPLATFORM | cut -d / -f1); \
    export GOARCH=$(echo $TARGETPLATFORM | cut -d / -f2); \
    if [ "$GOARCH" = "arm" ]; then \
        export GOARCH="arm"; \
        export GOARM=$(echo $TARGETPLATFORM | cut -d / -f3 | sed 's/v//g'); \
    fi; \
    export CGO_ENABLED=0; \
    echo "Building for $GOOS/$GOARCH/$GOARM"; \
    go build -ldflags="-s -w" -o ./custom-exporter .


FROM scratch
COPY --from=build /app/custom-exporter /custom-exporter
ENTRYPOINT ["/custom-exporter"]