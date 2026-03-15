# Sprint 2 - Service Layer Stabilization

Data: 2026-03-15

## Obiettivi
- Standardizzare gestione errori HTTP nel client API.
- Introdurre timeout configurabile e retry controllato per richieste idempotenti.
- Separare trasformazioni dati dal trasporto HTTP nei servizi principali.

## Modifiche principali
- `src/services/api.ts`
  - nuova classe `ApiError` con `status`, `path`, `method`.
  - supporto `timeoutMs`, `retries`, `retryDelayMs`, `retryOnStatuses`.
  - retry solo su metodi idempotenti (`GET`, `HEAD`, `OPTIONS`).
- Nuovi mapper dedicati:
  - `src/services/query.mapper.ts`
  - `src/services/auth.mapper.ts`
  - `src/services/dashboard.mapper.ts`
- Servizi aggiornati per usare i mapper:
  - `src/services/query.ts`
  - `src/services/auth.ts`
  - `src/services/dashboard.ts`

## Verifiche previste
- `npm run test:unit`

## Note di compatibilita
- Le firme pubbliche dei servizi restano invariate.
- La gestione errori mantiene `message` per compatibilita con gli store esistenti.
