# Sprint 3 - Store Domain Split

Data: 2026-03-15

## Obiettivi
- Separare la logica di dominio dagli store Svelte mantenendo API pubbliche invariate.
- Ridurre complessita interna e aumentare testabilita di merge/polling.

## Modifiche
- Nuovi moduli domain:
  - `src/lib/stores/query.domain.ts`
  - `src/lib/stores/dashboard.domain.ts`
- Store aggiornati:
  - `src/lib/stores/query.ts`
  - `src/lib/stores/dashboard.ts`
- Nuovi test unitari domain:
  - `tests/unit/stores/query.domain.spec.ts`
  - `tests/unit/stores/dashboard.domain.spec.ts`

## Compatibilita
- Nessun breaking change sull'API esportata dagli store.
- Componenti/route continuano a importare gli stessi simboli.

## Verifica
- `npm run test:unit`
- `npm run test:smoke`
