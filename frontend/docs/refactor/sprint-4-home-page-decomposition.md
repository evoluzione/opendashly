# Sprint 4 - Home Page Decomposition

Data: 2026-03-15

## Obiettivi
- Ridurre la complessita di `src/routes/+page.svelte` senza cambiare il comportamento UI.
- Estrarre logica non-visuale in modulo riusabile e testabile.

## Modifiche
- Nuovo modulo: `src/routes/home-page.logic.ts`
  - gestione range dashboard e validazioni
  - utility temporali
  - builder request query (run, cambio pagina, cambio page size)
  - gestione cursori paginazione
  - conteggio filtri attivi
- Route aggiornata: `src/routes/+page.svelte`
  - delega la logica al nuovo modulo helper
  - mantiene invariata API degli store e rendering dei componenti
- Nuovi test unit: `tests/unit/routes/home-page.logic.spec.ts`

## Verifica
- `npm run test:unit`
- `npm run test:smoke`
- `npm run build`

## Compatibilita
- Nessuna modifica ai contratti pubblici dei componenti consumati da `+page.svelte`.
- Nessuna modifica alle route.
