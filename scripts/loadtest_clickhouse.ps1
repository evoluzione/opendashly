param(
    [int]$Rows = 2000000,
    [string]$ClickHouseService = "clickhouse",
    [string]$User = "otel",
    [string]$Password = "otelpass",
    [string]$Database = "telemetry",
    [switch]$SkipInsert,
    [switch]$SkipBench,
    [switch]$TruncateFirst
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-CHQuery {
    param(
        [Parameter(Mandatory = $true)][string]$Sql,
        [switch]$Silent
    )

    $args = @(
        "compose", "exec", "-T", $ClickHouseService,
        "clickhouse-client",
        "--user", $User,
        "--password", $Password,
        "--database", $Database,
        "--query", $Sql
    )

    if ($Silent) {
        & docker @args | Out-Null
    }
    else {
        & docker @args
    }

    if ($LASTEXITCODE -ne 0) {
        throw "ClickHouse query failed (exit code $LASTEXITCODE)."
    }
}

function Get-CHScalar {
    param(
        [Parameter(Mandatory = $true)][string]$Sql
    )

    $args = @(
        "compose", "exec", "-T", $ClickHouseService,
        "clickhouse-client",
        "--user", $User,
        "--password", $Password,
        "--database", $Database,
        "--query", $Sql
    )

    $output = & docker @args
    if ($LASTEXITCODE -ne 0) {
        throw "ClickHouse scalar query failed (exit code $LASTEXITCODE)."
    }

    return ("$output".Trim())
}

function Test-CHColumnExists {
    param(
        [Parameter(Mandatory = $true)][string]$Table,
        [Parameter(Mandatory = $true)][string]$Column
    )

    $existsSql = "SELECT count() FROM system.columns WHERE database = '$Database' AND table = '$Table' AND name = '$Column'"
    $count = Get-CHScalar -Sql $existsSql
    return $count -ne "0"
}

function Measure-CHQuery {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Sql
    )

    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    Invoke-CHQuery -Sql $Sql -Silent
    $sw.Stop()

    [PSCustomObject]@{
        Query = $Name
        Ms    = [math]::Round($sw.Elapsed.TotalMilliseconds, 2)
    }
}

Write-Host "== OpenDashly ClickHouse load test ==" -ForegroundColor Cyan
Write-Host "Rows per signal table: $Rows"
Write-Host "Target service: $ClickHouseService | DB: $Database"

# Sanity check
Invoke-CHQuery -Sql "SELECT 1" -Silent

if (-not $SkipInsert) {
    if ($TruncateFirst) {
        Write-Host "Truncating OTel tables..." -ForegroundColor Yellow
        Invoke-CHQuery -Sql "TRUNCATE TABLE IF EXISTS otel_logs" -Silent
        Invoke-CHQuery -Sql "TRUNCATE TABLE IF EXISTS otel_traces" -Silent
        Invoke-CHQuery -Sql "TRUNCATE TABLE IF EXISTS otel_metrics_sum" -Silent
        Invoke-CHQuery -Sql "TRUNCATE TABLE IF EXISTS otel_metrics_gauge" -Silent
    }

    Write-Host "Inserting logs..." -ForegroundColor Green
    $hasEventName = Test-CHColumnExists -Table "otel_logs" -Column "EventName"
    if ($hasEventName) {
        Write-Host "Detected EventName column in otel_logs: using extended insert." -ForegroundColor DarkGray
    }
    else {
        Write-Host "EventName column not present in otel_logs: using compatible insert." -ForegroundColor DarkGray
    }

    $logsInsertColumns = @"
    Timestamp, TimestampTime, TraceId, SpanId, TraceFlags, SeverityText, SeverityNumber,
    ServiceName, Body, ResourceSchemaUrl, ResourceAttributes, ScopeSchemaUrl,
    ScopeName, ScopeVersion, ScopeAttributes, LogAttributes
"@
    $logsSelectTail = @"
    map('scope', 'loadtest') AS ScopeAttributes,
    map('env', if(number % 2 = 0, 'prod', 'staging'), 'region', concat('eu-', toString(number % 3 + 1))) AS LogAttributes
"@

    if ($hasEventName) {
        $logsInsertColumns += ", EventName"
        $logsSelectTail += ",`n    'loadtest_event' AS EventName"
    }

    $logsSql = @"
INSERT INTO otel_logs (
    $logsInsertColumns
)
SELECT
    ts AS Timestamp,
    toDateTime(ts) AS TimestampTime,
    lower(hex(MD5(concat('trace-', toString(number))))) AS TraceId,
    substring(lower(hex(MD5(concat('span-', toString(number))))), 1, 16) AS SpanId,
    toUInt8(1) AS TraceFlags,
    multiIf(number % 20 = 0, 'ERROR', number % 10 = 0, 'WARN', number % 3 = 0, 'DEBUG', 'INFO') AS SeverityText,
    multiIf(number % 20 = 0, toUInt8(17), number % 10 = 0, toUInt8(13), number % 3 = 0, toUInt8(5), toUInt8(9)) AS SeverityNumber,
    concat('svc-', toString(number % 30)) AS ServiceName,
    concat('log message #', toString(number), ' status=', if(number % 20 = 0, 'error', 'ok')) AS Body,
    'https://opentelemetry.io/schemas/1.0.0' AS ResourceSchemaUrl,
    map('service.name', concat('svc-', toString(number % 30)), 'host.name', concat('host-', toString(number % 500))) AS ResourceAttributes,
    'https://opentelemetry.io/schemas/1.0.0' AS ScopeSchemaUrl,
    'opendashly-loadtest' AS ScopeName,
    '1.0.0' AS ScopeVersion,
    $logsSelectTail
FROM
(
    SELECT
        number,
        now64(9) - toIntervalSecond(number % 604800) AS ts
    FROM numbers($Rows)
)
"@
    Invoke-CHQuery -Sql $logsSql -Silent

    Write-Host "Inserting traces..." -ForegroundColor Green
    $tracesSql = @"
INSERT INTO otel_traces (
    Timestamp, TraceId, SpanId, ParentSpanId, TraceState,
    SpanName, SpanKind, ServiceName, ResourceAttributes,
    ScopeName, ScopeVersion, SpanAttributes, Duration,
    StatusCode, StatusMessage,
    Events.Timestamp, Events.Name, Events.Attributes,
    Links.TraceId, Links.SpanId, Links.TraceState, Links.Attributes
)
SELECT
    ts AS Timestamp,
        lower(hex(MD5(concat('trace-', toString(traceBucket))))) AS TraceId,
    substring(lower(hex(MD5(concat('span-', toString(number))))), 1, 16) AS SpanId,
        if(number % 4 = 0, substring(lower(hex(MD5(concat('trace-parent-', toString(traceBucket), '-', toString(number % 37))))), 1, 16), '') AS ParentSpanId,
    '' AS TraceState,
        multiIf(
            kindSelector = 0, concat(if(number % 2 = 0, 'GET ', 'POST '), '/api/v1/resource/', toString(number % 200)),
            kindSelector = 1, concat('HTTP GET https://ext-', toString(number % 40), '.example.net/api/', toString(number % 120)),
            kindSelector = 2, concat('publish topic-', toString(number % 16)),
            kindSelector = 3, concat('consume topic-', toString(number % 16)),
            concat('internal.step.', toString(number % 60))
        ) AS SpanName,
        multiIf(
            kindSelector = 0, 'SERVER',
            kindSelector = 1, 'CLIENT',
            kindSelector = 2, 'PRODUCER',
            kindSelector = 3, 'CONSUMER',
            'INTERNAL'
        ) AS SpanKind,
        concat('svc-', toString(traceBucket % 30)) AS ServiceName,
        map('service.name', concat('svc-', toString(traceBucket % 30)), 'k8s.namespace.name', if(number % 2 = 0, 'prod', 'stage')) AS ResourceAttributes,
    'opendashly-loadtest' AS ScopeName,
    '1.0.0' AS ScopeVersion,
        map(
            'http.method', if(number % 2 = 0, 'GET', 'POST'),
            'http.route', concat('/api/v1/resource/', toString(number % 200)),
            'messaging.system', if(kindSelector IN (2, 3), 'kafka', ''),
            'messaging.operation', if(kindSelector = 2, 'publish', if(kindSelector = 3, 'process', '')),
            'net.peer.ip', if(kindSelector IN (1, 2, 3), concat('34.', toString(number % 250 + 1), '.', toString(number % 240 + 10), '.', toString(number % 220 + 20)), concat('10.', toString(number % 250 + 1), '.', toString(number % 240 + 10), '.', toString(number % 220 + 20)))
        ) AS SpanAttributes,
        toUInt64((
            multiIf(
                kindSelector = 0, (number % 2200 + 20),
                kindSelector = 1, (number % 3000 + 30),
                kindSelector = 2, (number % 1200 + 10),
                kindSelector = 3, (number % 1400 + 10),
                (number % 700 + 5)
            )
        ) * 1000000) AS Duration,
        if(number % 19 = 0, 'STATUS_CODE_ERROR', 'STATUS_CODE_OK') AS StatusCode,
        if(number % 19 = 0, concat('synthetic error in ', toString(SpanKind)), '') AS StatusMessage,
        [ts + toIntervalMillisecond(number % 800 + 5)] AS ``Events.Timestamp``,
        [multiIf(number % 19 = 0, 'exception', kindSelector = 2, 'queue.publish', kindSelector = 3, 'queue.consume', kindSelector = 1, 'http.client', 'app.step')] AS ``Events.Name``,
        [
            multiIf(
                number % 19 = 0,
                map('exception.type', 'SyntheticError', 'exception.message', concat('boom-', toString(number))),
                kindSelector = 2,
                map('messaging.system', 'kafka', 'messaging.destination', concat('topic-', toString(number % 16))),
                kindSelector = 3,
                map('messaging.system', 'kafka', 'messaging.destination', concat('topic-', toString(number % 16))),
                kindSelector = 1,
                map('http.url', concat('https://ext-', toString(number % 40), '.example.net/api/', toString(number % 120)), 'http.status_code', if(number % 19 = 0, '500', '200')),
                map('component', 'handler', 'step', toString(number % 12))
            )
        ] AS ``Events.Attributes``,
    [] AS ``Links.TraceId``,
    [] AS ``Links.SpanId``,
    [] AS ``Links.TraceState``,
    [] AS ``Links.Attributes``
FROM
(
    SELECT
        number,
                cityHash64(number) % toUInt64(greatest(1, intDiv($Rows, 14))) AS normalBucket,
                cityHash64(number) % toUInt64(greatest(1, intDiv($Rows, 120))) AS hotspotBucket,
                if(number % 7 = 0, hotspotBucket, normalBucket) AS traceBucket,
                cityHash64(number + 17) % 5 AS kindSelector,
        now64(9) - toIntervalSecond(number % 604800) AS ts
    FROM numbers($Rows)
)
"@
    Invoke-CHQuery -Sql $tracesSql -Silent

    Write-Host "Inserting metrics_sum..." -ForegroundColor Green
    $sumSql = @"
INSERT INTO otel_metrics_sum (
    ResourceAttributes, ResourceSchemaUrl, ScopeName, ScopeVersion, ScopeAttributes,
    ScopeDroppedAttrCount, ScopeSchemaUrl, ServiceName, MetricName, MetricDescription,
    MetricUnit, Attributes, StartTimeUnix, TimeUnix, Value, Flags,
    Exemplars.FilteredAttributes, Exemplars.TimeUnix, Exemplars.Value, Exemplars.SpanId, Exemplars.TraceId,
    AggregationTemporality, IsMonotonic
)
SELECT
    map('service.name', concat('svc-', toString(number % 30))) AS ResourceAttributes,
    'https://opentelemetry.io/schemas/1.0.0' AS ResourceSchemaUrl,
    'opendashly-loadtest' AS ScopeName,
    '1.0.0' AS ScopeVersion,
    map('scope', 'loadtest') AS ScopeAttributes,
    toUInt32(0) AS ScopeDroppedAttrCount,
    'https://opentelemetry.io/schemas/1.0.0' AS ScopeSchemaUrl,
    concat('svc-', toString(number % 30)) AS ServiceName,
    multiIf(number % 3 = 0, 'http.server.requests', number % 3 = 1, 'db.calls', 'queue.consumed') AS MetricName,
    'synthetic sum metric' AS MetricDescription,
    'count' AS MetricUnit,
    map('method', if(number % 2 = 0, 'GET', 'POST')) AS Attributes,
    ts - toIntervalSecond(10) AS StartTimeUnix,
    ts AS TimeUnix,
    toFloat64((number % 1000) + 1) AS Value,
    toUInt32(0) AS Flags,
    [] AS ``Exemplars.FilteredAttributes``,
    [] AS ``Exemplars.TimeUnix``,
    [] AS ``Exemplars.Value``,
    [] AS ``Exemplars.SpanId``,
    [] AS ``Exemplars.TraceId``,
    toInt32(2) AS AggregationTemporality,
    toUInt8(1) AS IsMonotonic
FROM
(
    SELECT
        number,
        now64(9) - toIntervalSecond(number % 604800) AS ts
    FROM numbers($Rows)
)
"@
    Invoke-CHQuery -Sql $sumSql -Silent

    Write-Host "Inserting metrics_gauge..." -ForegroundColor Green
    $gaugeSql = @"
INSERT INTO otel_metrics_gauge (
    ResourceAttributes, ResourceSchemaUrl, ScopeName, ScopeVersion, ScopeAttributes,
    ScopeDroppedAttrCount, ScopeSchemaUrl, ServiceName, MetricName, MetricDescription,
    MetricUnit, Attributes, StartTimeUnix, TimeUnix, Value, Flags,
    Exemplars.FilteredAttributes, Exemplars.TimeUnix, Exemplars.Value, Exemplars.SpanId, Exemplars.TraceId
)
SELECT
    map('service.name', concat('svc-', toString(number % 30))) AS ResourceAttributes,
    'https://opentelemetry.io/schemas/1.0.0' AS ResourceSchemaUrl,
    'opendashly-loadtest' AS ScopeName,
    '1.0.0' AS ScopeVersion,
    map('scope', 'loadtest') AS ScopeAttributes,
    toUInt32(0) AS ScopeDroppedAttrCount,
    'https://opentelemetry.io/schemas/1.0.0' AS ScopeSchemaUrl,
    concat('svc-', toString(number % 30)) AS ServiceName,
    multiIf(number % 2 = 0, 'cpu.usage', 'memory.usage') AS MetricName,
    'synthetic gauge metric' AS MetricDescription,
    if(number % 2 = 0, 'percent', 'bytes') AS MetricUnit,
    map('host', concat('host-', toString(number % 500))) AS Attributes,
    ts - toIntervalSecond(10) AS StartTimeUnix,
    ts AS TimeUnix,
    if(number % 2 = 0, toFloat64((number % 100) / 100.0), toFloat64((number % 500000) + 1000)) AS Value,
    toUInt32(0) AS Flags,
    [] AS ``Exemplars.FilteredAttributes``,
    [] AS ``Exemplars.TimeUnix``,
    [] AS ``Exemplars.Value``,
    [] AS ``Exemplars.SpanId``,
    [] AS ``Exemplars.TraceId``
FROM
(
    SELECT
        number,
        now64(9) - toIntervalSecond(number % 604800) AS ts
    FROM numbers($Rows)
)
"@
    Invoke-CHQuery -Sql $gaugeSql -Silent

    Write-Host "Insert completed." -ForegroundColor Green
}

if (-not $SkipBench) {
    Write-Host "Running benchmark queries..." -ForegroundColor Cyan
    $bench = @(
        @{ Name = "Logs latest page"; Sql = "SELECT Timestamp, SeverityText, Body, TraceId, SpanId FROM otel_logs WHERE Timestamp >= now() - INTERVAL 24 HOUR AND ServiceName = 'svc-1' ORDER BY Timestamp DESC LIMIT 101" },
        @{ Name = "Traces latest page"; Sql = "SELECT TraceId, argMin(SpanName, Timestamp) AS name, argMin(ServiceName, Timestamp) AS service, count() AS spanCount, countIf(StatusCode IN ('Error','2','STATUS_CODE_ERROR')) AS errorCount, max(Timestamp) AS lastSeen, max(Duration)/1000000 AS durationMs FROM otel_traces WHERE Timestamp >= now() - INTERVAL 24 HOUR AND ServiceName = 'svc-1' GROUP BY TraceId ORDER BY lastSeen DESC LIMIT 101" },
        @{ Name = "Throughput time series"; Sql = "SELECT toStartOfInterval(Timestamp, INTERVAL 5 MINUTE) AS bucket, count() AS request_count, countIf(StatusCode IN ('Error','2','STATUS_CODE_ERROR')) AS error_count FROM otel_traces WHERE Timestamp >= now() - INTERVAL 24 HOUR AND ServiceName = 'svc-1' AND SpanKind IN ('SERVER','SPAN_KIND_SERVER','2') GROUP BY bucket ORDER BY bucket" },
        @{ Name = "Error hotspots"; Sql = "SELECT SpanName AS endpoint, ServiceName AS service, countIf(StatusCode IN ('Error','2','STATUS_CODE_ERROR')) AS error_count, count() AS total_count, if(count() > 0, (countIf(StatusCode IN ('Error','2','STATUS_CODE_ERROR')) / count()) * 100, 0) AS error_rate FROM otel_traces WHERE Timestamp >= now() - INTERVAL 24 HOUR AND (startsWith(SpanName, 'GET ') OR startsWith(SpanName, 'POST ') OR startsWith(SpanName, 'PUT ') OR startsWith(SpanName, 'PATCH ') OR startsWith(SpanName, 'DELETE ') OR startsWith(SpanName, 'OPTIONS ') OR startsWith(SpanName, 'HEAD ')) AND SpanKind IN ('SERVER','SPAN_KIND_SERVER','2') GROUP BY endpoint, service HAVING total_count >= 5 AND error_count > 0 ORDER BY error_rate DESC, error_count DESC LIMIT 10" },
        @{ Name = "Metrics union page"; Sql = "SELECT name, unit, timestamp, value FROM (SELECT MetricName AS name, MetricUnit AS unit, TimeUnix AS timestamp, Value AS value FROM otel_metrics_sum WHERE TimeUnix >= now() - INTERVAL 24 HOUR AND ServiceName = 'svc-1' ORDER BY TimeUnix DESC LIMIT 10000 UNION ALL SELECT MetricName AS name, MetricUnit AS unit, TimeUnix AS timestamp, Value AS value FROM otel_metrics_gauge WHERE TimeUnix >= now() - INTERVAL 24 HOUR AND ServiceName = 'svc-1' ORDER BY TimeUnix DESC LIMIT 10000) ORDER BY timestamp DESC LIMIT 101" }
    )

    $results = foreach ($item in $bench) {
        Measure-CHQuery -Name $item.Name -Sql $item.Sql
    }

    Write-Host ""
    Write-Host "Benchmark results (ms):" -ForegroundColor Cyan
    $results | Sort-Object Ms | Format-Table -AutoSize
}

Write-Host "Done." -ForegroundColor Cyan
