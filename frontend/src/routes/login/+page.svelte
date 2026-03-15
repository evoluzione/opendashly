<script lang="ts">
  import { goto } from '$app/navigation';
  import { authState, loginUser } from '../../lib/stores/auth';
  import { locale, t } from '$lib/i18n';
  let username = '';
  let password = '';

  async function submit() {
    const session = await loginUser(username, password);
    if (!session) {
      return;
    }
    if (session.mustChangePassword) {
      await goto('/first-login');
    } else {
      await goto('/');
    }
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
          <span>{t($locale, 'common.dashboard')}</span>
        </div>
      </div>
      
      <div class="features">
        <div class="feature">
          <div class="feature-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
          </div>
          <div>
            <strong>Log</strong>
            <p>{t($locale, 'login.logExplore')}</p>
          </div>
        </div>
        <div class="feature">
          <div class="feature-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="20" x2="18" y2="10"/>
              <line x1="12" y1="20" x2="12" y2="4"/>
              <line x1="6" y1="20" x2="6" y2="14"/>
            </svg>
          </div>
          <div>
            <strong>{t($locale, 'home.metrics')}</strong>
            <p>{t($locale, 'login.metricsView')}</p>
          </div>
        </div>
        <div class="feature">
          <div class="feature-icon">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
            </svg>
          </div>
          <div>
            <strong>{t($locale, 'home.traces')}</strong>
            <p>{t($locale, 'login.tracesAnalyze')}</p>
          </div>
        </div>
      </div>
    </div>
  </div>
  
  <div class="login-right">
    <div class="card">
      <div class="card-header">
        <h2>{t($locale, 'login.welcomeBack')}</h2>
        <p>{t($locale, 'login.enterCredentials')}</p>
      </div>

      <div class="form">
        <div class="field">
          <label for="username">{t($locale, 'login.username')}</label>
          <div class="input-wrapper">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <input 
              id="username" 
              bind:value={username} 
              placeholder={t($locale, 'login.usernamePlaceholder')}
              on:keydown={handleKeydown}
            />
          </div>
        </div>
        
        <div class="field">
          <label for="password">{t($locale, 'login.password')}</label>
          <div class="input-wrapper">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            <input 
              id="password" 
              type="password" 
              bind:value={password} 
              placeholder={t($locale, 'login.passwordPlaceholder')}
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

        <button class="submit-btn" on:click={submit} disabled={$authState.loading || !username || !password}>
          {#if $authState.loading}
            <span class="spinner"></span>
            {t($locale, 'login.signingIn')}
          {:else}
            {t($locale, 'login.signIn')}
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
    position: relative;
  }
  
  .login-left {
    background: linear-gradient(135deg, var(--color-slate-950) 0%, var(--color-slate-900) 100%);
    padding: 60px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    position: relative;
    overflow: hidden;
  }

  .login-left-inner {
    width: min(520px, 100%);
    display: flex;
    flex-direction: column;
    gap: 32px;
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
  
  .login-left::after {
    content: '';
    position: absolute;
    bottom: -30%;
    left: -30%;
    width: 80%;
    height: 80%;
    background: radial-gradient(circle, rgba(139, 92, 246, 0.1) 0%, transparent 70%);
    pointer-events: none;
  }
  
  .branding {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 60px;
    position: relative;
    z-index: 1;
  }
  
  .logo {
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    border-radius: 16px;
    color: white;
    box-shadow: 0 8px 24px rgba(var(--rgb-primary-600), 0.4);
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
  
  .features {
    display: flex;
    flex-direction: column;
    gap: 24px;
    position: relative;
    z-index: 1;
  }
  
  .feature {
    display: flex;
    align-items: flex-start;
    gap: 16px;
  }
  
  .feature-icon {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    color: #a5b4fc;
    flex-shrink: 0;
  }
  
  .feature strong {
    display: block;
    color: var(--color-slate-100);
    font-size: 15px;
    margin-bottom: 4px;
  }
  
  .feature p {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 13px;
    line-height: 1.5;
  }
  
  .login-right {
    background: linear-gradient(135deg, var(--color-slate-50) 0%, var(--color-slate-100) 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;
  }
  
  .card {
    background: white;
    border-radius: 24px;
    padding: 48px;
    width: 100%;
    max-width: 420px;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.1);
    border: 1px solid rgba(var(--rgb-slate-950), 0.06);
  }

  .card-header {
    text-align: center;
    margin-bottom: 32px;
  }
  
  .card-header h2 {
    margin: 0 0 8px 0;
    font-size: 28px;
    font-weight: 700;
    color: var(--color-slate-950);
  }
  
  .card-header p {
    margin: 0;
    color: var(--color-slate-500);
    font-size: 15px;
  }
  
  .form {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  
  .field {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  label {
    font-size: 13px;
    font-weight: 600;
    color: #374151;
  }
  
  .input-wrapper {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    border-radius: 12px;
    border: 2px solid #e5e7eb;
    background: #f9fafb;
    transition: all 0.2s ease;
  }
  
  .input-wrapper:focus-within {
    border-color: var(--color-primary-600);
    background: white;
    box-shadow: 0 0 0 4px rgba(var(--rgb-primary-600), 0.1);
  }
  
  .input-wrapper svg {
    color: #9ca3af;
    flex-shrink: 0;
  }
  
  .input-wrapper:focus-within svg {
    color: var(--color-primary-600);
  }
  
  input {
    flex: 1;
    border: none;
    background: transparent;
    font-size: 15px;
    color: var(--color-slate-950);
    outline: none;
  }
  
  input::placeholder {
    color: #9ca3af;
  }
  
  .error {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    background: var(--color-danger-50);
    border: 1px solid var(--color-danger-75);
    border-radius: 10px;
    color: var(--color-danger-600);
    font-size: 14px;
  }
  
  .submit-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: 8px;
    padding: 16px 24px;
    border-radius: 12px;
    border: none;
    background: linear-gradient(135deg, var(--color-primary-600) 0%, var(--color-primary-500) 100%);
    color: white;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
    box-shadow: 0 4px 14px rgba(var(--rgb-primary-600), 0.4);
  }
  
  .submit-btn:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 8px 20px rgba(var(--rgb-primary-600), 0.5);
  }
  
  .submit-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    transform: none;
  }
  
  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255, 255, 255, 0.3);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  @media (max-width: 900px) {
    .login {
      grid-template-columns: 1fr;
    }
    
    .login-left {
      display: none;
    }
    
    .login-right {
      padding: 24px;
    }
    
    .card {
      padding: 32px;
    }
  }
</style>
