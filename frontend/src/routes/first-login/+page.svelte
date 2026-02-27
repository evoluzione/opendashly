<script lang="ts">
  import { goto } from '$app/navigation';
  import { authState, changePasswordForFirstLogin } from '../../lib/stores/auth';

  let newPassword = '';

  async function submit() {
    const session = await changePasswordForFirstLogin(newPassword);
    if (!session) {
      return;
    }
    await goto('/');
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      submit();
    }
  }
</script>

<section class="login">
  <div class="login-left">
    <div class="login-left-inner">
      <div class="branding">
        <div class="logo">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5-10-5z"/>
            <path d="M2 17l10 5 10-5"/>
            <path d="M2 12l10 5 10-5"/>
          </svg>
        </div>
        <div class="brand-text">
          <h1>Opendashly</h1>
          <span>Dashboard</span>
        </div>
      </div>

      <div class="notice">
        <h2>Cambio password richiesto</h2>
        <p>Per sicurezza devi impostare una nuova password prima di continuare.</p>
      </div>
    </div>
  </div>

  <div class="login-right">
    <div class="card">
      <div class="card-header">
        <h2>Imposta una nuova password</h2>
        <p>Questo passaggio sarà necessario solo la prima volta che accedi. Assicurati di scegliere una password sicura.</p>
      </div>

      <div class="form">
        <div class="field">
          <label for="new">Nuova password</label>
          <div class="input-wrapper">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            <input
              id="new"
              type="password"
              bind:value={newPassword}
              placeholder="Inserisci la nuova password"
              on:keydown={handleKeydown}
            />
          </div>
        </div>

        {#if $authState.error}
          <div class="error">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="15" y1="9" x2="9" y2="15"/>
              <line x1="9" y1="9" x2="15" y2="15"/>
            </svg>
            <span>{$authState.error}</span>
          </div>
        {/if}

        <button class="submit-btn" on:click={submit} disabled={$authState.loading || !newPassword}>
          {#if $authState.loading}
            <span class="spinner"></span>
            Aggiornamento in corso...
          {:else}
            Aggiorna password
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="5" y1="12" x2="19" y2="12"/>
              <polyline points="12 5 19 12 12 19"/>
            </svg>
          {/if}
        </button>
      </div>
    </div>
  </div>
</section>

<style>
  .login {
    min-height: 100vh;
    display: grid;
    grid-template-columns: 1fr 1fr;
  }

  .login-left {
    background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
    padding: 60px;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    overflow: hidden;
  }

  .login-left::before {
    content: '';
    position: absolute;
    top: -50%;
    right: -50%;
    width: 100%;
    height: 100%;
    background: radial-gradient(circle, rgba(99, 102, 241, 0.15) 0%, transparent 70%);
    pointer-events: none;
  }

  .login-left-inner {
    width: min(520px, 100%);
    position: relative;
    z-index: 1;
  }

  .branding {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 40px;
  }

  .logo {
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    border-radius: 16px;
    color: white;
    box-shadow: 0 8px 24px rgba(99, 102, 241, 0.4);
  }

  .brand-text h1 {
    margin: 0;
    font-size: 24px;
    font-weight: 700;
    color: #f8fafc;
  }

  .brand-text span {
    font-size: 13px;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .notice h2 {
    margin: 0 0 8px 0;
    font-size: 30px;
    font-weight: 700;
    color: #f8fafc;
  }

  .notice p {
    margin: 0;
    font-size: 15px;
    line-height: 1.6;
    color: #94a3b8;
  }

  .login-right {
    background: linear-gradient(135deg, #f0f4f8 0%, #e2e8f0 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }

  .card {
    background: #fff;
    border-radius: 24px;
    padding: 40px;
    width: min(460px, 100%);
    box-shadow: 0 20px 60px rgba(15, 23, 42, 0.12), 0 8px 24px rgba(15, 23, 42, 0.08);
    border: 1px solid rgba(148, 163, 184, 0.2);
  }

  .card-header {
    margin-bottom: 28px;
  }

  .card-header h2 {
    margin: 0 0 8px 0;
    font-size: 28px;
    font-weight: 700;
    color: #0f172a;
    letter-spacing: -0.02em;
  }

  .card-header p {
    margin: 0;
    color: #64748b;
    font-size: 15px;
    line-height: 1.5;
  }

  .form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .field {
    display: grid;
    gap: 8px;
  }

  label {
    font-size: 13px;
    font-weight: 600;
    color: #334155;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .input-wrapper svg {
    position: absolute;
    left: 14px;
    color: #94a3b8;
    pointer-events: none;
  }

  input {
    width: 100%;
    padding: 14px 16px 14px 44px;
    border-radius: 12px;
    border: 1px solid #cbd5e1;
    background: #f8fafc;
    color: #0f172a;
    font-size: 15px;
    transition: all 0.2s;
  }

  input:focus {
    outline: none;
    border-color: #6366f1;
    background: #ffffff;
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
  }

  .submit-btn {
    width: 100%;
    padding: 14px 20px;
    border-radius: 12px;
    border: none;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: #fff;
    font-weight: 600;
    font-size: 15px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    transition: all 0.2s;
  }

  .submit-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 8px 20px rgba(99, 102, 241, 0.35);
  }

  .submit-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .error {
    display: flex;
    align-items: center;
    gap: 8px;
    color: #dc2626;
    font-size: 14px;
    padding: 12px 14px;
    background: #fef2f2;
    border: 1px solid #fecaca;
    border-radius: 12px;
  }

  @media (max-width: 980px) {
    .login {
      grid-template-columns: 1fr;
    }

    .login-left {
      padding: 40px 24px;
    }

    .notice h2 {
      font-size: 26px;
    }

    .login-right {
      padding: 24px;
    }

    .card {
      padding: 28px;
      border-radius: 20px;
    }
  }
</style>
