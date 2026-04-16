<script lang="ts">
  import { goto } from '$app/navigation';
  import { authState, changePasswordForFirstLogin } from '../../lib/stores/auth';
  import { locale, t } from '$lib/i18n';

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
          <img src="/sidebar-mark.svg" alt="OpenDashly" />
        </div>
        <div class="brand-text">
          <h1>Opendashly</h1>
          <span>{t($locale, 'common.dashboard')}</span>
        </div>
      </div>

      <div class="notice">
        <h2>{t($locale, 'firstLogin.requiredTitle')}</h2>
        <p>{t($locale, 'firstLogin.requiredText')}</p>
      </div>
    </div>
  </div>

  <div class="login-right">
    <div class="card">
      <div class="card-header">
        <h2>{t($locale, 'firstLogin.setNewPassword')}</h2>
        <p>{t($locale, 'firstLogin.oneTimeStep')}</p>
      </div>

      <div class="form">
        <div class="field">
          <label for="new">{t($locale, 'firstLogin.newPassword')}</label>
          <div class="input-wrapper">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            <input
              id="new"
              type="password"
              bind:value={newPassword}
              placeholder={t($locale, 'firstLogin.newPasswordPlaceholder')}
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
            {t($locale, 'firstLogin.updating')}
          {:else}
            {t($locale, 'firstLogin.updatePassword')}
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
    background: linear-gradient(135deg, var(--color-slate-950) 0%, var(--color-slate-900) 100%);
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
    background: radial-gradient(circle, rgba(var(--rgb-primary-600), 0.15) 0%, transparent 70%);
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
    border-radius: 16px;
    box-shadow: 0 8px 24px rgba(var(--rgb-primary-600), 0.4);
    overflow: hidden;
  }

  .logo img {
    width: 100%;
    height: 100%;
    display: block;
    object-fit: cover;
  }

  .brand-text h1 {
    margin: 0;
    font-size: 24px;
    font-weight: 700;
    color: var(--color-slate-50);
  }

  .brand-text span {
    font-size: 13px;
    color: var(--color-slate-500);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .notice h2 {
    margin: 0 0 8px 0;
    font-size: 30px;
    font-weight: 700;
    color: var(--color-slate-50);
  }

  .notice p {
    margin: 0;
    font-size: 15px;
    line-height: 1.6;
    color: var(--color-slate-400);
  }

  .login-right {
    background: linear-gradient(135deg, #f0f4f8 0%, var(--color-slate-200) 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }

  .card {
    background: var(--color-white);
    border-radius: 24px;
    padding: 40px;
    width: min(460px, 100%);
    box-shadow: 0 20px 60px rgba(var(--rgb-slate-950), 0.12), 0 8px 24px rgba(var(--rgb-slate-950), 0.08);
    border: 1px solid rgba(148, 163, 184, 0.2);
  }

  .card-header {
    margin-bottom: 28px;
  }

  .card-header h2 {
    margin: 0 0 8px 0;
    font-size: 28px;
    font-weight: 700;
    color: var(--color-slate-950);
    letter-spacing: -0.02em;
  }

  .card-header p {
    margin: 0;
    color: var(--color-slate-500);
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
    color: var(--color-slate-700);
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
    color: var(--color-slate-400);
    pointer-events: none;
  }

  input {
    width: 100%;
    padding: 14px 16px 14px 44px;
    border-radius: 12px;
    border: 1px solid var(--color-slate-300);
    background: var(--color-slate-50);
    color: var(--color-slate-950);
    font-size: 15px;
    transition: all 0.2s;
  }

  input:focus {
    outline: none;
    border-color: var(--color-primary-600);
    background: var(--color-white);
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.1);
  }

  .submit-btn {
    width: 100%;
    padding: 14px 20px;
    border-radius: 12px;
    border: none;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: var(--color-white);
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
    box-shadow: 0 8px 20px rgba(var(--rgb-primary-600), 0.35);
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
    color: var(--color-danger-600);
    font-size: 14px;
    padding: 12px 14px;
    background: var(--color-danger-50);
    border: 1px solid var(--color-danger-75);
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
