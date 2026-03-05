FROM clickhouse/clickhouse-server:25.1

# Keep default image config.d files and add only custom log policy overrides.
COPY ./config.d/system-logs.xml /etc/clickhouse-server/config.d/system-logs.xml
