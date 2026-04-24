#!/usr/bin/env node

import process from "node:process";
import { setTimeout as sleep } from "node:timers/promises";

import { context, trace, SpanKind, SpanStatusCode } from "@opentelemetry/api";
import { SeverityNumber } from "@opentelemetry/api-logs";
import { Resource } from "@opentelemetry/resources";
import { NodeTracerProvider } from "@opentelemetry/sdk-trace-node";
import { BatchSpanProcessor } from "@opentelemetry/sdk-trace-base";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-http";
import { LoggerProvider, BatchLogRecordProcessor } from "@opentelemetry/sdk-logs";
import { OTLPLogExporter } from "@opentelemetry/exporter-logs-otlp-http";
import {
  MeterProvider,
  PeriodicExportingMetricReader,
  AggregationTemporality
} from "@opentelemetry/sdk-metrics";
import { OTLPMetricExporter } from "@opentelemetry/exporter-metrics-otlp-http";

const ALL_SERVICES = [
  "api-gateway",
  "auth-service",
  "catalog-service",
  "cart-service",
  "checkout-service",
  "payment-service",
  "inventory-service",
  "shipping-service",
  "notification-service"
];

const JOURNEYS = [
  {
    id: "browse",
    weight: 38,
    rootRoute: "/api/v1/catalog/search",
    method: "GET",
    rootService: "api-gateway",
    calls: [
      { caller: "api-gateway", callee: "catalog-service", method: "GET", route: "/catalog/search", baseMs: 35, jitterMs: 25 },
      { caller: "catalog-service", callee: "inventory-service", method: "GET", route: "/inventory/availability", baseMs: 22, jitterMs: 18 }
    ],
    baseMs: 80,
    jitterMs: 60,
    errorBias: 0.25
  },
  {
    id: "product-detail",
    weight: 18,
    rootRoute: "/api/v1/catalog/products/{sku}",
    method: "GET",
    rootService: "api-gateway",
    calls: [
      { caller: "api-gateway", callee: "catalog-service", method: "GET", route: "/catalog/products/{sku}", baseMs: 30, jitterMs: 20 },
      { caller: "catalog-service", callee: "inventory-service", method: "GET", route: "/inventory/stock/{sku}", baseMs: 18, jitterMs: 14 }
    ],
    baseMs: 70,
    jitterMs: 50,
    errorBias: 0.2
  },
  {
    id: "add-to-cart",
    weight: 22,
    rootRoute: "/api/v1/cart/items",
    method: "POST",
    rootService: "api-gateway",
    calls: [
      { caller: "api-gateway", callee: "auth-service", method: "GET", route: "/auth/session", baseMs: 12, jitterMs: 10 },
      { caller: "api-gateway", callee: "cart-service", method: "POST", route: "/cart/items", baseMs: 28, jitterMs: 20 },
      { caller: "cart-service", callee: "inventory-service", method: "GET", route: "/inventory/stock/{sku}", baseMs: 16, jitterMs: 12 }
    ],
    baseMs: 95,
    jitterMs: 65,
    errorBias: 0.45
  },
  {
    id: "checkout",
    weight: 16,
    rootRoute: "/api/v1/checkout",
    method: "POST",
    rootService: "api-gateway",
    calls: [
      { caller: "api-gateway", callee: "checkout-service", method: "POST", route: "/checkout", baseMs: 38, jitterMs: 24 },
      { caller: "checkout-service", callee: "payment-service", method: "POST", route: "/payments/authorize", baseMs: 85, jitterMs: 80, canFail: true },
      { caller: "checkout-service", callee: "inventory-service", method: "POST", route: "/inventory/reserve", baseMs: 30, jitterMs: 18, canFail: true },
      { caller: "checkout-service", callee: "shipping-service", method: "POST", route: "/shipping/quote", baseMs: 25, jitterMs: 16 },
      { caller: "checkout-service", callee: "notification-service", method: "POST", route: "/notifications/order-confirmed", baseMs: 10, jitterMs: 8 }
    ],
    baseMs: 220,
    jitterMs: 150,
    errorBias: 2.2
  },
  {
    id: "order-tracking",
    weight: 6,
    rootRoute: "/api/v1/orders/{orderId}",
    method: "GET",
    rootService: "api-gateway",
    calls: [
      { caller: "api-gateway", callee: "shipping-service", method: "GET", route: "/shipping/orders/{orderId}", baseMs: 25, jitterMs: 14 }
    ],
    baseMs: 55,
    jitterMs: 30,
    errorBias: 0.15
  },
  {
    id: "catalog-search-heavy",
    weight: 3,
    rootRoute: "/api/v1/catalog/search",
    method: "GET",
    rootService: "api-gateway",
    calls: [
      {
        caller: "api-gateway",
        callee: "catalog-service",
        method: "GET",
        route: "/catalog/search",
        baseMs: 90,
        jitterMs: 50,
        fatPricing: { itemsRange: [40, 140] }
      },
      { caller: "catalog-service", callee: "inventory-service", method: "GET", route: "/inventory/availability", baseMs: 22, jitterMs: 18 }
    ],
    baseMs: 180,
    jitterMs: 100,
    errorBias: 0.15
  }
];

const DEFAULTS = {
  durationSec: 300,
  rps: 100,
  services: 9,
  errorRate: 0.035,
  hotRate: 0.2,
  maxInFlight: 500,
  metricsExportIntervalMs: 5000,
  collectorBase: "http://localhost:4318",
  tenant: "tenant-demo",
  environment: "loadtest",
  seed: null,
  traces: true,
  logs: true,
  metrics: true,
  quiet: false
};

function parseArgs(argv) {
  const args = { ...DEFAULTS };

  const alias = {
    "duration": "durationSec",
    "duration-sec": "durationSec",
    "rps": "rps",
    "services": "services",
    "error-rate": "errorRate",
    "hot-rate": "hotRate",
    "max-in-flight": "maxInFlight",
    "metrics-export-interval-ms": "metricsExportIntervalMs",
    "collector": "collectorBase",
    "tenant": "tenant",
    "env": "environment",
    "environment": "environment",
    "seed": "seed",
    "traces": "traces",
    "logs": "logs",
    "metrics": "metrics",
    "quiet": "quiet"
  };

  for (let i = 0; i < argv.length; i += 1) {
    const raw = argv[i];
    if (!raw.startsWith("--")) {
      continue;
    }

    if (raw === "--help") {
      printHelp();
      process.exit(0);
    }

    const [keyRaw, inlineValue] = raw.slice(2).split("=", 2);
    const isNegated = keyRaw.startsWith("no-");
    const key = isNegated ? keyRaw.slice(3) : keyRaw;
    const mapped = alias[key];
    if (!mapped) {
      throw new Error(`Unknown parameter: ${raw}`);
    }

    const targetType = typeof DEFAULTS[mapped];
    if (targetType === "boolean") {
      if (isNegated) {
        args[mapped] = false;
        continue;
      }
      if (inlineValue === undefined) {
        args[mapped] = true;
      } else {
        args[mapped] = parseBoolean(inlineValue);
      }
      continue;
    }

    const value = inlineValue !== undefined ? inlineValue : argv[++i];
    if (value === undefined) {
      throw new Error(`Missing value for --${keyRaw}`);
    }

    switch (mapped) {
      case "durationSec":
      case "services":
      case "maxInFlight":
      case "metricsExportIntervalMs": {
        args[mapped] = parseInt(value, 10);
        break;
      }
      case "rps":
      case "errorRate":
      case "hotRate": {
        args[mapped] = Number(value);
        break;
      }
      case "seed": {
        args[mapped] = Number(value);
        break;
      }
      default: {
        args[mapped] = value;
      }
    }
  }

  validateArgs(args);
  return args;
}

function parseBoolean(value) {
  const lowered = String(value).toLowerCase();
  if (["1", "true", "yes", "on"].includes(lowered)) {
    return true;
  }
  if (["0", "false", "no", "off"].includes(lowered)) {
    return false;
  }
  throw new Error(`Invalid boolean value: ${value}`);
}

function validateArgs(args) {
  if (!Number.isFinite(args.durationSec) || args.durationSec <= 0) {
    throw new Error("durationSec must be > 0");
  }
  if (!Number.isFinite(args.rps) || args.rps <= 0) {
    throw new Error("rps must be > 0");
  }
  if (!Number.isFinite(args.services) || args.services <= 1 || args.services > ALL_SERVICES.length) {
    throw new Error(`services must be in range [2, ${ALL_SERVICES.length}]`);
  }
  if (!Number.isFinite(args.errorRate) || args.errorRate < 0 || args.errorRate > 1) {
    throw new Error("errorRate must be in range [0,1]");
  }
  if (!Number.isFinite(args.hotRate) || args.hotRate < 0 || args.hotRate > 1) {
    throw new Error("hotRate must be in range [0,1]");
  }
  if (!Number.isFinite(args.maxInFlight) || args.maxInFlight < 1) {
    throw new Error("maxInFlight must be >= 1");
  }
  if (!Number.isFinite(args.metricsExportIntervalMs) || args.metricsExportIntervalMs < 1000) {
    throw new Error("metricsExportIntervalMs must be >= 1000");
  }
  if (!args.traces && !args.logs && !args.metrics) {
    throw new Error("At least one signal must be enabled: traces/logs/metrics");
  }
}

function printHelp() {
  console.log(`
OpenDashly ecommerce OTLP load test (Node)

Usage:
  node scripts/loadtest_otel_node.mjs [options]

Options:
  --duration, --duration-sec <sec>         Test duration in seconds (default: ${DEFAULTS.durationSec})
  --rps <n>                                Target requests per second (default: ${DEFAULTS.rps})
  --services <n>                           Number of active microservices 2..${ALL_SERVICES.length} (default: ${DEFAULTS.services})
  --error-rate <0..1>                      Base error rate (default: ${DEFAULTS.errorRate})
  --hot-rate <0..1>                        Extra traffic share for checkout hotspot (default: ${DEFAULTS.hotRate})
  --max-in-flight <n>                      In-flight request cap (default: ${DEFAULTS.maxInFlight})
  --collector <url>                        OTLP HTTP base endpoint (default: ${DEFAULTS.collectorBase})
  --metrics-export-interval-ms <ms>        Metrics export interval (default: ${DEFAULTS.metricsExportIntervalMs})
  --tenant <id>                            Tenant tag (default: ${DEFAULTS.tenant})
  --environment <name>                     Environment tag (default: ${DEFAULTS.environment})
  --seed <n>                               Deterministic seed
  --traces | --no-traces                   Enable/disable traces (default: ON)
  --logs | --no-logs                       Enable/disable logs (default: ON)
  --metrics | --no-metrics                 Enable/disable metrics (default: ON)
  --quiet | --no-quiet                     Compact output
  --help                                   Show this help

Examples:
  node scripts/loadtest_otel_node.mjs --duration 180 --rps 120
  node scripts/loadtest_otel_node.mjs --duration 300 --rps 250 --error-rate 0.06 --hot-rate 0.35
  node scripts/loadtest_otel_node.mjs --duration 120 --rps 80 --services 6 --no-metrics
`);
}

class Rng {
  constructor(seed) {
    const finalSeed = Number.isFinite(seed) ? seed : Date.now();
    this.state = (finalSeed >>> 0) || 0x9e3779b9;
  }

  next() {
    let t = this.state += 0x6d2b79f5;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  }

  int(min, max) {
    return Math.floor(this.next() * (max - min + 1)) + min;
  }

  pick(list) {
    return list[Math.floor(this.next() * list.length)];
  }
}

function weightedPick(rng, weightedItems) {
  const total = weightedItems.reduce((acc, item) => acc + item.weight, 0);
  let target = rng.next() * total;
  for (const item of weightedItems) {
    target -= item.weight;
    if (target <= 0) {
      return item;
    }
  }
  return weightedItems[weightedItems.length - 1];
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function buildCollectorUrls(base) {
  const normalized = base.endsWith("/") ? base.slice(0, -1) : base;
  const alreadySignalPath = /\/v1\/(traces|logs|metrics)$/.test(normalized);
  if (alreadySignalPath) {
    return {
      traces: normalized.replace(/\/v1\/(logs|metrics)$/, "/v1/traces"),
      logs: normalized.replace(/\/v1\/(traces|metrics)$/, "/v1/logs"),
      metrics: normalized.replace(/\/v1\/(traces|logs)$/, "/v1/metrics")
    };
  }
  return {
    traces: `${normalized}/v1/traces`,
    logs: `${normalized}/v1/logs`,
    metrics: `${normalized}/v1/metrics`
  };
}

function createServiceTelemetry(serviceName, cfg, urls) {
  const resource = new Resource({
    "service.name": serviceName,
    "service.namespace": "ecommerce",
    "service.version": "1.0.0",
    "deployment.environment": cfg.environment,
    "host.name": `loadgen-${cfg.tenant}`
  });

  const service = {
    serviceName,
    tracerProvider: null,
    tracer: null,
    loggerProvider: null,
    logger: null,
    meterProvider: null,
    meter: null,
    metricReader: null,
    metrics: null,
    runtime: {
      cpuUsage: 0.12,
      memoryUsageBytes: 300_000_000 + Math.floor(Math.random() * 400_000_000)
    }
  };

  if (cfg.traces) {
    const traceExporter = new OTLPTraceExporter({ url: urls.traces });
    service.tracerProvider = new NodeTracerProvider({ resource });
    service.tracerProvider.addSpanProcessor(new BatchSpanProcessor(traceExporter));
    service.tracer = service.tracerProvider.getTracer("opendashly-loadtest", "1.0.0");
  }

  if (cfg.logs) {
    const logExporter = new OTLPLogExporter({ url: urls.logs });
    service.loggerProvider = new LoggerProvider({ resource });
    service.loggerProvider.addLogRecordProcessor(new BatchLogRecordProcessor(logExporter));
    service.logger = service.loggerProvider.getLogger("opendashly-loadtest", "1.0.0");
  }

  if (cfg.metrics) {
    const metricExporter = new OTLPMetricExporter({
      url: urls.metrics,
      temporalityPreference: AggregationTemporality.DELTA
    });

    service.metricReader = new PeriodicExportingMetricReader({
      exporter: metricExporter,
      exportIntervalMillis: cfg.metricsExportIntervalMs
    });

    service.meterProvider = new MeterProvider({
      resource,
      readers: [service.metricReader]
    });

    service.meter = service.meterProvider.getMeter("opendashly-loadtest", "1.0.0");

    const requestCounter = service.meter.createCounter("http.server.requests", {
      description: "Number of incoming ecommerce requests",
      unit: "count"
    });
    const errorCounter = service.meter.createCounter("http.server.errors", {
      description: "Number of incoming ecommerce failed requests",
      unit: "count"
    });
    const durationHistogram = service.meter.createHistogram("http.server.duration", {
      description: "Server-side request duration",
      unit: "ms"
    });
    const businessCounter = service.meter.createCounter("ecommerce.business.events", {
      description: "Business counters for checkout/cart flow",
      unit: "count"
    });

    const cpuGauge = service.meter.createObservableGauge("service.cpu.usage", {
      description: "Synthetic CPU usage",
      unit: "1"
    });
    cpuGauge.addCallback((observableResult) => {
      observableResult.observe(clamp(service.runtime.cpuUsage, 0.01, 0.95), { service: serviceName });
    });

    const memGauge = service.meter.createObservableGauge("service.memory.usage", {
      description: "Synthetic memory usage",
      unit: "By"
    });
    memGauge.addCallback((observableResult) => {
      observableResult.observe(Math.max(120_000_000, service.runtime.memoryUsageBytes), { service: serviceName });
    });

    service.metrics = {
      requestCounter,
      errorCounter,
      durationHistogram,
      businessCounter
    };
  }

  return service;
}

function makeDataset(rng) {
  const userId = `usr-${rng.int(100000, 999999)}`;
  const sessionId = `sess-${rng.int(100000, 999999)}-${rng.int(1000, 9999)}`;
  const sku = `SKU-${rng.int(1000, 9999)}`;
  const orderId = `ORD-${rng.int(100000, 999999)}`;
  const cartId = `CRT-${rng.int(100000, 999999)}`;
  return { userId, sessionId, sku, orderId, cartId };
}

function resolveRouteTemplate(routeTemplate, data) {
  return routeTemplate
    .replace("{sku}", data.sku)
    .replace("{orderId}", data.orderId);
}

function jitterMs(rng, base, jitter) {
  return Math.max(3, Math.round(base + (rng.next() * 2 - 1) * jitter));
}

function statusTextFromCode(code) {
  if (code >= 500) {
    return "ERROR";
  }
  if (code >= 400) {
    return "WARN";
  }
  return "INFO";
}

function messageForJourney(journeyId, code, data) {
  switch (journeyId) {
    case "browse":
      return code >= 400
        ? `Catalog lookup degraded for sku=${data.sku}`
        : `Catalog listing returned for session=${data.sessionId}`;
    case "product-detail":
      return code >= 400
        ? `Product detail failed for sku=${data.sku}`
        : `Product detail rendered for sku=${data.sku}`;
    case "add-to-cart":
      return code >= 400
        ? `Add to cart failed for cart=${data.cartId} sku=${data.sku}`
        : `Cart updated cart=${data.cartId} sku=${data.sku}`;
    case "checkout":
      return code >= 400
        ? `Checkout failed order=${data.orderId}`
        : `Checkout completed order=${data.orderId}`;
    default:
      return code >= 400
        ? `Order tracking failed order=${data.orderId}`
        : `Order tracking delivered order=${data.orderId}`;
  }
}

function chooseJourney(rng, hotRate, activeServices) {
  const base = JOURNEYS
    .map((j) => ({ ...j }))
    .filter((j) => j.calls.every((c) => activeServices.has(c.caller) && activeServices.has(c.callee)) && activeServices.has(j.rootService));

  for (const item of base) {
    if (item.id === "checkout") {
      item.weight = item.weight * (1 + hotRate * 2.5);
    }
  }

  return weightedPick(rng, base);
}

function spanName(method, route) {
  return `${method.toUpperCase()} ${route}`;
}

function emitFatPricingSpans(service, parentCtx, serverStartMs, serverDurationMs, rng, itemsRange) {
  const [minItems, maxItems] = itemsRange;
  const itemCount = rng.int(minItems, maxItems);
  const windowMs = Math.max(1, serverDurationMs - 2);
  const sku = `SKU-${rng.int(1000, 9999)}`;

  for (let i = 0; i < itemCount; i += 1) {
    const slotStart = serverStartMs + 1 + Math.floor((i / itemCount) * windowMs);
    const calcDuration = rng.int(0, 2);

    const calcSpan = service.tracer.startSpan(
      "CalculateFinalPrice",
      {
        kind: SpanKind.INTERNAL,
        attributes: {
          "pricing.item_index": i,
          "pricing.sku": `${sku}-${i}`
        },
        startTime: slotStart
      },
      parentCtx
    );
    const calcCtx = trace.setSpan(parentCtx, calcSpan);

    const findSpan = service.tracer.startSpan(
      "FindPrice",
      {
        kind: SpanKind.INTERNAL,
        attributes: { "pricing.step": "find" },
        startTime: slotStart
      },
      calcCtx
    );
    findSpan.end(slotStart + rng.int(0, 1));

    const strategiesSpan = service.tracer.startSpan(
      "GetPossibleStrategies",
      {
        kind: SpanKind.INTERNAL,
        attributes: { "pricing.step": "strategies" },
        startTime: slotStart
      },
      calcCtx
    );
    strategiesSpan.end(slotStart + rng.int(0, 1));

    if (rng.next() < 0.35) {
      const lowestSpan = service.tracer.startSpan(
        "CalculateLowestPrice30Days",
        {
          kind: SpanKind.INTERNAL,
          attributes: { "pricing.step": "lowest30d" },
          startTime: slotStart
        },
        calcCtx
      );
      lowestSpan.end(slotStart + rng.int(0, 1));
    }

    calcSpan.end(slotStart + calcDuration);
  }
}

function emitLog(service, ctx, severity, body, attributes) {
  if (!service.logger) {
    return;
  }
  const severityMap = {
    INFO: SeverityNumber.INFO,
    WARN: SeverityNumber.WARN,
    ERROR: SeverityNumber.ERROR,
    DEBUG: SeverityNumber.DEBUG
  };

  service.logger.emit({
    timestamp: Date.now(),
    severityText: severity,
    severityNumber: severityMap[severity] ?? SeverityNumber.INFO,
    body,
    attributes,
    context: ctx
  });
}

function recordServerMetric(service, payload) {
  if (!service.metrics) {
    return;
  }

  const baseAttrs = {
    "http.method": payload.method,
    "http.route": payload.route,
    "http.status_code": String(payload.httpStatus),
    "ecommerce.journey": payload.journey,
    "ecommerce.tenant": payload.tenant
  };

  service.metrics.requestCounter.add(1, baseAttrs);
  service.metrics.durationHistogram.record(payload.durationMs, baseAttrs);
  if (payload.isError) {
    service.metrics.errorCounter.add(1, baseAttrs);
  }

  if (payload.journey === "checkout") {
    service.metrics.businessCounter.add(1, {
      event: payload.isError ? "orders_failed" : "orders_created",
      "ecommerce.tenant": payload.tenant
    });
  }

  if (payload.journey === "add-to-cart" && payload.isError) {
    service.metrics.businessCounter.add(1, {
      event: "cart_abandonment",
      "ecommerce.tenant": payload.tenant
    });
  }
}

async function simulateRequest(runtime, reqNo) {
  const { cfg, rng, servicesByName, activeServices } = runtime;
  const data = makeDataset(rng);
  const journey = chooseJourney(rng, cfg.hotRate, activeServices);
  const correlationId = `corr-${reqNo}-${rng.int(10000, 99999)}`;

  const rootRoute = resolveRouteTemplate(journey.rootRoute, data);
  const rootDuration = jitterMs(rng, journey.baseMs, journey.jitterMs);
  const localErrorRate = clamp(cfg.errorRate * journey.errorBias, 0, 0.95);
  const shouldFail = rng.next() < localErrorRate;

  let failCallIndex = -1;
  let failStatus = 500;
  if (shouldFail && journey.calls.length > 0) {
    const eligible = journey.calls
      .map((call, index) => ({ call, index }))
      .filter((item) => item.call.canFail);
    if (eligible.length > 0) {
      const target = rng.pick(eligible);
      failCallIndex = target.index;
      failStatus = target.call.route.includes("/payments/") ? 402 : 409;
    } else {
      failCallIndex = rng.int(0, journey.calls.length - 1);
      failStatus = 503;
    }
  }

  const gateway = servicesByName.get(journey.rootService);
  const rootStart = Date.now();

  let rootSpan = null;
  let rootCtx = context.active();

  if (cfg.traces) {
    rootSpan = gateway.tracer.startSpan(
      spanName(journey.method, rootRoute),
      {
        kind: SpanKind.SERVER,
        attributes: {
          "http.method": journey.method,
          "http.route": rootRoute,
          "http.target": rootRoute,
          "http.user_agent": "synthetic-browser/1.0",
          "enduser.id": data.userId,
          "session.id": data.sessionId,
          "ecommerce.journey": journey.id,
          "ecommerce.cart.id": data.cartId,
          "ecommerce.order.id": data.orderId,
          "ecommerce.tenant": cfg.tenant
        },
        startTime: rootStart
      }
    );
    rootCtx = trace.setSpan(rootCtx, rootSpan);
  }

  const rootLogAttrs = {
    tenant: cfg.tenant,
    env: cfg.environment,
    journey: journey.id,
    route: rootRoute,
    method: journey.method,
    user_id: data.userId,
    session_id: data.sessionId,
    cart_id: data.cartId,
    order_id: data.orderId,
    sku: data.sku,
    correlation_id: correlationId
  };

  emitLog(gateway, rootCtx, "INFO", `Request accepted ${journey.method} ${rootRoute}`, rootLogAttrs);

  for (let i = 0; i < journey.calls.length; i += 1) {
    const call = journey.calls[i];
    const caller = servicesByName.get(call.caller);
    const callee = servicesByName.get(call.callee);
    const route = resolveRouteTemplate(call.route, data);

    const isErrorCall = shouldFail && i === failCallIndex;
    const httpStatus = isErrorCall ? failStatus : 200;
    const serverDuration = jitterMs(rng, call.baseMs, call.jitterMs);
    const clientDuration = Math.max(serverDuration + rng.int(2, 16), 5);
    const callStart = Date.now();

    let clientSpan = null;
    let clientCtx = rootCtx;

    if (cfg.traces) {
      clientSpan = caller.tracer.startSpan(
        `HTTP ${call.method} ${route}`,
        {
          kind: SpanKind.CLIENT,
          attributes: {
            "http.method": call.method,
            "http.route": route,
            "http.target": route,
            "net.peer.name": call.callee,
            "rpc.system": "http",
            "ecommerce.journey": journey.id,
            "ecommerce.tenant": cfg.tenant
          },
          startTime: callStart
        },
        rootCtx
      );
      clientCtx = trace.setSpan(rootCtx, clientSpan);
    }

    emitLog(caller, clientCtx, "DEBUG", `Calling ${call.callee} ${call.method} ${route}`, {
      ...rootLogAttrs,
      source_service: call.caller,
      target_service: call.callee,
      route,
      method: call.method
    });

    let serverSpan = null;
    let serverCtx = clientCtx;

    if (cfg.traces) {
      serverSpan = callee.tracer.startSpan(
        spanName(call.method, route),
        {
          kind: SpanKind.SERVER,
          attributes: {
            "http.method": call.method,
            "http.route": route,
            "http.target": route,
            "http.status_code": httpStatus,
            "enduser.id": data.userId,
            "session.id": data.sessionId,
            "ecommerce.journey": journey.id,
            "ecommerce.tenant": cfg.tenant,
            "peer.service": call.caller
          },
          startTime: callStart + 1
        },
        clientCtx
      );

      if (isErrorCall) {
        serverSpan.addEvent("exception", {
          "exception.type": failStatus === 402 ? "PaymentDeclined" : "DownstreamFailure",
          "exception.message": `Synthetic failure on ${call.callee}`,
          "error.type": "synthetic"
        });
      }

      serverCtx = trace.setSpan(clientCtx, serverSpan);

      if (call.fatPricing) {
        emitFatPricingSpans(callee, serverCtx, callStart + 1, serverDuration, rng, call.fatPricing.itemsRange);
      }
    }

    const severity = statusTextFromCode(httpStatus);
    emitLog(callee, serverCtx, severity, messageForJourney(journey.id, httpStatus, data), {
      ...rootLogAttrs,
      service: call.callee,
      route,
      method: call.method,
      status_code: String(httpStatus)
    });

    recordServerMetric(callee, {
      method: call.method,
      route,
      httpStatus,
      durationMs: serverDuration,
      isError: isErrorCall,
      journey: journey.id,
      tenant: cfg.tenant
    });

    callee.runtime.cpuUsage += (rng.next() - 0.5) * 0.03;
    callee.runtime.memoryUsageBytes += Math.floor((rng.next() - 0.45) * 4_000_000);

    if (cfg.traces) {
      if (isErrorCall) {
        serverSpan.setStatus({ code: SpanStatusCode.ERROR, message: `HTTP ${httpStatus}` });
      } else {
        serverSpan.setStatus({ code: SpanStatusCode.OK });
      }
      serverSpan.end(callStart + serverDuration);

      if (isErrorCall) {
        clientSpan.setStatus({ code: SpanStatusCode.ERROR, message: `HTTP ${httpStatus}` });
      } else {
        clientSpan.setStatus({ code: SpanStatusCode.OK });
      }
      clientSpan.end(callStart + clientDuration);
    }
  }

  const rootHttpStatus = shouldFail ? failStatus : 200;
  const rootSeverity = statusTextFromCode(rootHttpStatus);

  emitLog(gateway, rootCtx, rootSeverity, messageForJourney(journey.id, rootHttpStatus, data), {
    ...rootLogAttrs,
    service: journey.rootService,
    status_code: String(rootHttpStatus),
    latency_ms: String(rootDuration)
  });

  recordServerMetric(gateway, {
    method: journey.method,
    route: rootRoute,
    httpStatus: rootHttpStatus,
    durationMs: rootDuration,
    isError: shouldFail,
    journey: journey.id,
    tenant: cfg.tenant
  });

  gateway.runtime.cpuUsage += (rng.next() - 0.45) * 0.04;
  gateway.runtime.memoryUsageBytes += Math.floor((rng.next() - 0.42) * 6_000_000);

  if (cfg.traces && rootSpan) {
    rootSpan.setAttribute("http.status_code", rootHttpStatus);
    if (shouldFail) {
      rootSpan.setStatus({ code: SpanStatusCode.ERROR, message: `HTTP ${rootHttpStatus}` });
      rootSpan.addEvent("exception", {
        "exception.type": "RequestFailure",
        "exception.message": `Synthetic ${journey.id} failed with ${rootHttpStatus}`
      });
    } else {
      rootSpan.setStatus({ code: SpanStatusCode.OK });
    }
    rootSpan.end(rootStart + rootDuration);
  }

  return {
    journeyId: journey.id,
    isError: shouldFail,
    rootDuration,
    rootHttpStatus
  };
}

async function flushAndShutdown(servicesByName, cfg) {
  const tasks = [];

  for (const service of servicesByName.values()) {
    if (cfg.traces && service.tracerProvider) {
      tasks.push(service.tracerProvider.forceFlush().catch(() => {}));
      tasks.push(service.tracerProvider.shutdown().catch(() => {}));
    }
    if (cfg.logs && service.loggerProvider) {
      tasks.push(service.loggerProvider.forceFlush().catch(() => {}));
      tasks.push(service.loggerProvider.shutdown().catch(() => {}));
    }
    if (cfg.metrics && service.meterProvider) {
      tasks.push(service.meterProvider.forceFlush().catch(() => {}));
      tasks.push(service.meterProvider.shutdown().catch(() => {}));
    }
  }

  await Promise.all(tasks);
}

function printHeader(cfg, activeServices, urls) {
  console.log("== OpenDashly OTLP ecommerce load test ==");
  console.log(`durationSec=${cfg.durationSec} rps=${cfg.rps} services=${activeServices.size} maxInFlight=${cfg.maxInFlight}`);
  console.log(`signals traces=${cfg.traces} logs=${cfg.logs} metrics=${cfg.metrics}`);
  console.log(`collector traces=${urls.traces} logs=${urls.logs} metrics=${urls.metrics}`);
  console.log(`errorRate=${cfg.errorRate} hotRate=${cfg.hotRate} tenant=${cfg.tenant} env=${cfg.environment}`);
  console.log("");
}

function printSummary(stats, startedAt) {
  const elapsedSec = Math.max((Date.now() - startedAt) / 1000, 0.001);
  const avgRps = stats.total / elapsedSec;
  const errorRate = stats.total > 0 ? (stats.errors / stats.total) * 100 : 0;

  const sorted = [...stats.durations].sort((a, b) => a - b);
  const p = (q) => {
    if (sorted.length === 0) {
      return 0;
    }
    const idx = Math.min(sorted.length - 1, Math.floor(q * sorted.length));
    return sorted[idx];
  };

  console.log("\n== Summary ==");
  console.log(`Total requests: ${stats.total}`);
  console.log(`Errors: ${stats.errors} (${errorRate.toFixed(2)}%)`);
  console.log(`Achieved avg RPS: ${avgRps.toFixed(2)}`);
  console.log(`Root latency ms p50=${p(0.5).toFixed(2)} p95=${p(0.95).toFixed(2)} p99=${p(0.99).toFixed(2)}`);
  console.log("Journeys:");
  for (const [journey, count] of Object.entries(stats.byJourney)) {
    console.log(`  - ${journey}: ${count}`);
  }
}

async function run() {
  const cfg = parseArgs(process.argv.slice(2));
  const rng = new Rng(cfg.seed);
  const urls = buildCollectorUrls(cfg.collectorBase);

  const activeServiceNames = ALL_SERVICES.slice(0, cfg.services);
  const activeServices = new Set(activeServiceNames);

  const servicesByName = new Map();
  for (const serviceName of activeServiceNames) {
    servicesByName.set(serviceName, createServiceTelemetry(serviceName, cfg, urls));
  }

  printHeader(cfg, activeServices, urls);

  const stats = {
    total: 0,
    errors: 0,
    durations: [],
    byJourney: {}
  };

  const startedAt = Date.now();
  const endAt = startedAt + cfg.durationSec * 1000;
  let tokens = 0;
  let lastRefillMs = startedAt;
  let reqNo = 0;

  const inFlight = new Set();

  while (Date.now() < endAt) {
    const now = Date.now();
    const deltaSec = (now - lastRefillMs) / 1000;
    lastRefillMs = now;
    tokens += deltaSec * cfg.rps;

    const maxBurst = cfg.rps * 1.5;
    if (tokens > maxBurst) {
      tokens = maxBurst;
    }

    let launched = false;
    while (tokens >= 1 && inFlight.size < cfg.maxInFlight) {
      tokens -= 1;
      launched = true;
      reqNo += 1;

      const promise = simulateRequest(
        {
          cfg,
          rng,
          servicesByName,
          activeServices
        },
        reqNo
      )
        .then((result) => {
          stats.total += 1;
          if (result.isError) {
            stats.errors += 1;
          }
          stats.durations.push(result.rootDuration);
          stats.byJourney[result.journeyId] = (stats.byJourney[result.journeyId] || 0) + 1;
        })
        .catch((err) => {
          stats.total += 1;
          stats.errors += 1;
          if (!cfg.quiet) {
            console.error(`request ${reqNo} failed: ${err.message}`);
          }
        })
        .finally(() => {
          inFlight.delete(promise);
        });

      inFlight.add(promise);
    }

    if (!launched || inFlight.size >= cfg.maxInFlight) {
      await sleep(5);
    }
  }

  await Promise.all(inFlight);
  await sleep(Math.max(1000, cfg.metricsExportIntervalMs + 250));
  await flushAndShutdown(servicesByName, cfg);

  printSummary(stats, startedAt);
}

run().catch((err) => {
  console.error("Load test failed:", err.message);
  process.exitCode = 1;
});
