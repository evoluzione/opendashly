# Sprint 5 - Warning Cleanup

Data: 2026-03-15

## Obiettivi
- Ridurre warning frontend non bloccanti emersi in build.
- Migliorare accessibilita nei componenti modali/autocomplete.
- Eliminare warning su prop e selettori CSS inutilizzati.

## Modifiche principali
- A11y modal/backdrop:
  - `src/components/TraceResultsList.svelte`
  - `src/components/common/ConfirmModal.svelte`
- A11y opzioni autocomplete:
  - `src/components/common/Autocomplete.svelte`
- Utilizzo prop `lastUpdatedLabel` per evitare export inutilizzati:
  - `src/components/LogResultsTable.svelte`
  - `src/components/TraceResultsList.svelte`
- Rimozione selettori CSS non usati:
  - `src/routes/+page.svelte`
  - `src/routes/settings/ai/+page.svelte`
  - `src/components/Sidebar.svelte`
- Fix variabile non dichiarata in Sidebar:
  - `src/components/Sidebar.svelte`

## Verifica
- `npm run test:unit`
- `npm run test:smoke`
- `npm run build`

## Note
- Restano warning legati al mismatch tra runtime Svelte e import interni SvelteKit (`untrack`, `fork`, `settled`), non introdotti da questo sprint.
