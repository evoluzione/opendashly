FROM golang:1.25-alpine AS autotune-build

WORKDIR /src
COPY autotune/main.go /src/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/otel-autotune-launcher /src/main.go

FROM otel/opentelemetry-collector-contrib:0.122.0

# Bundle collector configuration inside the image to avoid host-side file mounts.
COPY collector-config.yaml /etc/otelcol/config.yaml
COPY --from=autotune-build /out/otel-autotune-launcher /otel-autotune-launcher

ENTRYPOINT ["/otel-autotune-launcher"]
