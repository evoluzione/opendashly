<script lang="ts">
  import { onMount } from 'svelte';
  import { createUser, listUsers } from '../services/users';
  import type { User } from '../services/auth';

  let users: User[] = [];
  let loading = false;
  let error = '';

  let username = '';
  let password = '';
  let role: 'admin' | 'user' = 'user';

  async function load() {
    loading = true;
    error = '';
    try {
      users = await listUsers();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Impossibile caricare gli utenti';
    } finally {
      loading = false;
    }
  }

  async function submit() {
    if (!username || !password) {
      error = 'Username e password sono obbligatori';
      return;
    }
    loading = true;
    error = '';
    try {
      const created = await createUser({ username, password, role });
      users = [...users, created].sort((a, b) => a.username.localeCompare(b.username));
      username = '';
      password = '';
      role = 'user';
    } catch (err) {
      error = err instanceof Error ? err.message : 'Impossibile creare l\'utente';
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<section class="user-management">
  <header>
    <h2>Gestione utenti</h2>
    <p>Crea e gestisci gli accessi alla dashboard.</p>
  </header>

  <div class="panel">
    <h3>Nuovo utente</h3>
    <div class="form">
      <div>
        <label for="username">Nome utente</label>
        <input id="username" bind:value={username} placeholder="nome utente" />
      </div>
      <div>
        <label for="password">Password</label>
        <input id="password" type="password" bind:value={password} placeholder="password" />
      </div>
      <div>
        <label for="role">Ruolo</label>
        <select id="role" bind:value={role}>
          <option value="user">Utente</option>
          <option value="admin">Amministratore</option>
        </select>
      </div>
      <button on:click={submit} disabled={loading}>Crea utente</button>
    </div>
  </div>

  <div class="panel">
    <h3>Utenti</h3>
    {#if error}
      <div class="error">{error}</div>
    {/if}
    {#if loading}
      <div class="status">Caricamento...</div>
    {:else if users.length === 0}
      <div class="status">Nessun utente disponibile.</div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>Nome utente</th>
            <th>Ruolo</th>
            <th>Stato</th>
          </tr>
        </thead>
        <tbody>
          {#each users as user}
            <tr>
              <td>{user.username}</td>
              <td>{user.role}</td>
              <td>{user.isDisabled ? 'Disabilitato' : 'Attivo'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</section>

<style>
  .user-management {
    display: grid;
    gap: 20px;
  }
  header h2 {
    margin: 0 0 6px 0;
  }
  .panel {
    background: #fff;
    border-radius: 16px;
    padding: 20px;
    border: 1px solid rgba(15, 20, 25, 0.08);
  }
  .form {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 12px;
  }
  label {
    font-size: 12px;
    text-transform: uppercase;
  }
  input,
  select {
    width: 100%;
    padding: 10px 12px;
    border-radius: 10px;
    border: 1px solid rgba(15, 20, 25, 0.15);
  }
  button {
    padding: 10px 14px;
    border-radius: 10px;
    border: none;
    background: #0f172a;
    color: #fff;
    cursor: pointer;
  }
  table {
    width: 100%;
    border-collapse: collapse;
  }
  th,
  td {
    text-align: left;
    padding: 10px 6px;
    border-bottom: 1px solid rgba(15, 20, 25, 0.08);
  }
  .status {
    color: #6b6f76;
  }
  .error {
    color: #b42318;
    margin-bottom: 8px;
  }
</style>
