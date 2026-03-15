<script lang="ts">
  import { onMount } from "svelte";
  import { createUser, listUsers } from "../services/users";
  import type { User } from "../services/auth";
  import { locale, t } from "../lib/i18n";

  let users: User[] = [];
  let loading = false;
  let error = "";

  let username = "";
  let password = "";
  let role: "admin" | "user" = "user";

  async function load() {
    loading = true;
    error = "";
    try {
      users = await listUsers();
    } catch (err) {
      error =
        err instanceof Error ? err.message : t($locale, "users.loadError");
    } finally {
      loading = false;
    }
  }

  async function submit() {
    if (!username || !password) {
      error = t($locale, "users.required");
      return;
    }
    loading = true;
    error = "";
    try {
      const created = await createUser({ username, password, role });
      users = [...users, created].sort((a, b) =>
        a.username.localeCompare(b.username),
      );
      username = "";
      password = "";
      role = "user";
    } catch (err) {
      error =
        err instanceof Error ? err.message : t($locale, "users.createError");
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
    <h2>{t($locale, "users.title")}</h2>
    <p>{t($locale, "users.subtitle")}</p>
  </header>

  <div class="panel">
    <h3>{t($locale, "users.newUser")}</h3>
    <div class="form">
      <div>
        <label for="username">{t($locale, "users.username")}</label>
        <input id="username" bind:value={username} placeholder={t($locale, "users.username")} />
      </div>
      <div>
        <label for="password">{t($locale, "users.password")}</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          placeholder={t($locale, "users.password")}
        />
      </div>
      <div>
        <label for="role">{t($locale, "users.role")}</label>
        <select id="role" bind:value={role}>
          <option value="user">{t($locale, "users.roleUser")}</option>
          <option value="admin">{t($locale, "users.roleAdmin")}</option>
        </select>
      </div>
      <button on:click={submit} disabled={loading}>{t($locale, "users.create")}</button>
    </div>
  </div>

  <div class="panel">
    <h3>{t($locale, "users.listTitle")}</h3>
    {#if error}
      <div class="error">{error}</div>
    {/if}
    {#if loading}
      <div class="status">{t($locale, "common.loading")}</div>
    {:else if users.length === 0}
      <div class="status">{t($locale, "users.noUsers")}</div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>{t($locale, "users.username")}</th>
            <th>{t($locale, "users.role")}</th>
            <th>{t($locale, "users.status")}</th>
          </tr>
        </thead>
        <tbody>
          {#each users as user}
            <tr>
              <td>{user.username}</td>
              <td>{user.role}</td>
              <td>{user.isDisabled ? t($locale, "users.disabled") : t($locale, "users.active")}</td>
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
    font-size: 22px;
    color: #0f172a;
  }
  header p {
    margin: 0;
    color: #64748b;
    font-size: 14px;
  }
  .panel {
    background: #fff;
    border-radius: 16px;
    padding: 20px;
    border: 1px solid rgba(15, 23, 42, 0.06);
    box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
  }
  .form {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 12px;
  }
  label {
    display: block;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
    margin-bottom: 6px;
  }
  input,
  select {
    width: 100%;
    padding: 12px 14px;
    border-radius: 10px;
    border: 1px solid #e2e8f0;
    background: #f8fafc;
    font-size: 14px;
    color: #0f172a;
    transition: all 0.2s ease;
  }
  input:focus,
  select:focus {
    outline: none;
    border-color: #6366f1;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.12);
    background: #fff;
  }
  button {
    width: fit-content;
    height: fit-content;
    padding: 12px 24px;
    align-self: flex-end;
    justify-self: end;
    border-radius: 10px;
    border: none;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: #fff;
    cursor: pointer;
    font-weight: 600;
    transition: all 0.2s ease;
    box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
  }
  button:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 6px 16px rgba(99, 102, 241, 0.4);
  }
  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
  }
  th,
  td {
    text-align: left;
    padding: 10px 6px;
    border-bottom: 1px solid rgba(15, 23, 42, 0.06);
  }
  th {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #64748b;
  }
  .status {
    color: #6b6f76;
  }
  .error {
    color: #b42318;
    margin-bottom: 8px;
  }
</style>
