# Sprint 1 Baseline (Frontend Refactor)

Data baseline: 2026-03-15

## Obiettivi Sprint 1
- Congelare i flussi critici con una smoke suite E2E dedicata.
- Introdurre base unit test su servizi frontend.
- Rendere misurabile la baseline tecnica prima del refactor strutturale.

## Comandi di riferimento
- Unit test: `npm run test:unit`
- Smoke E2E: `npm run test:smoke`
- Tutti gli E2E: `npm run test:ui`

## Hotspot complessita (linee file)
Snapshot iniziale (frontend/src, file .svelte/.ts):
- `src/routes/admin/status/+page.svelte`: 1322
- `src/routes/+page.svelte`: 1314
- `src/components/TraceSpanTimeline.svelte`: 1164
- `src/components/QueryForm.svelte`: 1083
- `src/components/LogResultsTable.svelte`: 961
- `src/components/DashboardSettings.svelte`: 922
- `src/components/TraceResultsList.svelte`: 710
- `src/components/AIAssistantWidget.svelte`: 653
- `src/components/RetentionManagement.svelte`: 573
- `src/components/FilterBuilder.svelte`: 488

## Ambito smoke suite (critico, environment-agnostic)
- `tests/integration/smoke_auth_shell.spec.ts`

Copertura smoke introdotta:
- Render pagina login (controlli principali)
- Redirect da route protetta verso login senza sessione
- Render pagina first-login e form cambio password

## Criteri uscita Sprint 1
- Script smoke dedicato disponibile e documentato.
- Presenza di unit test su servizi core (query, dashboard, auth).
- Test unitari verdi in locale/CI.
