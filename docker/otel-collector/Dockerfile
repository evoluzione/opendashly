FROM otel/opentelemetry-collector-contrib:0.122.0

# Bundle collector configuration inside the image to avoid host-side file mounts.
COPY collector-config.yaml /etc/otelcol/config.yaml
