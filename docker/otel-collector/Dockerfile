FROM golang:1.25-alpine AS autotune-build

WORKDIR /src
COPY autotune/ /src/autotune/
COPY healthcheck/ /src/healthcheck/
RUN CGO_ENABLED=0 GOOS=linux go build -C /src/autotune -o /out/otel-autotune-launcher . \
    && CGO_ENABLED=0 GOOS=linux go build -C /src/healthcheck -o /out/otel-healthcheck .

FROM otel/opentelemetry-collector-contrib:0.122.0

# Bundle collector configuration inside the image to avoid host-side file mounts.
COPY collector-config.yaml /etc/otelcol/config.yaml
COPY --from=autotune-build /out/otel-autotune-launcher /otel-autotune-launcher
COPY --from=autotune-build /out/otel-healthcheck /otel-healthcheck

ENTRYPOINT ["/otel-autotune-launcher"]
