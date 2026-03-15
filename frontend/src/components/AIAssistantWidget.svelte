<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import ConfirmModal from "./common/ConfirmModal.svelte";
  import {
    getAIAssistantAvailability,
    getAssistantSession,
    resetAssistantSession,
    saveAssistantSession,
    sendAssistantMessage,
    type AssistantMessage,
  } from "../services/assistant";
  import { locale, t } from "../lib/i18n";

  type ChatMessage = AssistantMessage & {
    id: string;
    loading?: boolean;
  };

  function getThinkingStages() {
    return [
      t($locale, "assistant.stageAnalyzing"),
      t($locale, "assistant.stageQuerying"),
      t($locale, "assistant.stagePreparing"),
    ];
  }

  let open = false;
  let aiEnabled = false;
  let historyLoaded = false;
  let prompt = "";
  let sending = false;
  let messages: ChatMessage[] = [];
  let error = "";
  let confirmResetOpen = false;
  let thinkingTimer: ReturnType<typeof setInterval> | null = null;
  let stageIndex = 0;
  let dockBottomPx = 20;
  let layoutObserver: MutationObserver | null = null;
  let resizeHandler: (() => void) | null = null;

  onMount(async () => {
    await Promise.all([loadAvailability(), loadHistory()]);
    updateDockOffset();

    resizeHandler = () => updateDockOffset();
    window.addEventListener("resize", resizeHandler);

    layoutObserver = new MutationObserver(() => {
      updateDockOffset();
    });
    layoutObserver.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ["class", "style"],
    });
  });

  onDestroy(() => {
    if (thinkingTimer) {
      clearInterval(thinkingTimer);
      thinkingTimer = null;
    }
    if (resizeHandler) {
      window.removeEventListener("resize", resizeHandler);
      resizeHandler = null;
    }
    if (layoutObserver) {
      layoutObserver.disconnect();
      layoutObserver = null;
    }
  });

  async function loadAvailability() {
    try {
      const res = await getAIAssistantAvailability();
      aiEnabled = !!res.enabled;
    } catch {
      aiEnabled = false;
    }
  }

  async function loadHistory() {
    historyLoaded = false;
    try {
      const session = await getAssistantSession();
      messages = (session.messages ?? [])
        .filter((m) => m && (m.role === "user" || m.role === "assistant"))
        .map((m, i) => ({
          id: `m-${i}-${Date.now()}`,
          role: m.role,
          content: String(m.content ?? ""),
        }));
    } catch {
      messages = [];
    } finally {
      historyLoaded = true;
    }
  }

  async function persistHistory() {
    const compact = messages
      .filter((m) => !m.loading)
      .map((m) => ({ role: m.role, content: m.content }));
    try {
      await saveAssistantSession({ messages: compact });
    } catch {
      // Keep UI reactive even if persistence fails.
    }
  }

  function toggleOpen() {
    open = !open;
    updateDockOffset();
  }

  function startThinkingAnimation(placeholderId: string) {
    const stages = getThinkingStages();
    stageIndex = 0;
    thinkingTimer = setInterval(() => {
      stageIndex = (stageIndex + 1) % stages.length;
      messages = messages.map((m) =>
        m.id === placeholderId
          ? { ...m, content: stages[stageIndex] }
          : m,
      );
    }, 1800);
  }

  function stopThinkingAnimation() {
    if (thinkingTimer) {
      clearInterval(thinkingTimer);
      thinkingTimer = null;
    }
  }

  function askResetConversation() {
    confirmResetOpen = true;
  }

  function cancelResetConversation() {
    confirmResetOpen = false;
  }

  async function confirmResetConversation() {
    confirmResetOpen = false;
    messages = [];
    prompt = "";
    error = "";
    try {
      await resetAssistantSession();
    } catch {
      // Ignore reset failures.
    }
  }

  async function sendMessage() {
    const trimmed = prompt.trim();
    if (!trimmed || sending || !aiEnabled) return;

    sending = true;
    error = "";

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: "user",
      content: trimmed,
    };
    const placeholder: ChatMessage = {
      id: `assistant-loading-${Date.now()}`,
      role: "assistant",
      content: getThinkingStages()[0],
      loading: true,
    };

    messages = [...messages, userMessage, placeholder];
    prompt = "";
    startThinkingAnimation(placeholder.id);

    try {
      const historyForApi: AssistantMessage[] = messages
        .filter((m) => !m.loading)
        .map((m) => ({ role: m.role, content: m.content }));

      const response = await sendAssistantMessage({
        prompt: trimmed,
        messages: historyForApi,
      });
      stopThinkingAnimation();
      messages = messages.map((m) =>
        m.id === placeholder.id
          ? {
              id: `assistant-${Date.now()}`,
              role: "assistant",
              content: response.answer,
            }
          : m,
      );
      await persistHistory();
    } catch (err) {
      stopThinkingAnimation();
      error = err instanceof Error ? err.message : t($locale, "assistant.errorGeneric");
      messages = messages.map((m) =>
        m.id === placeholder.id
          ? {
              id: `assistant-error-${Date.now()}`,
              role: "assistant",
              content: t($locale, "assistant.errorReply"),
            }
          : m,
      );
      await persistHistory();
    } finally {
      sending = false;
    }
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      void sendMessage();
    }
  }

  function escapeHtml(value: string): string {
    return value
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function renderInlineMarkdown(value: string): string {
    const escaped = escapeHtml(value);
    return escaped
      .replace(/`([^`]+)`/g, "<code>$1</code>")
      .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  }

  function renderAssistantMarkdown(value: string): string {
    const lines = value.replace(/\r/g, "").split("\n");
    const blocks: string[] = [];
    let inUl = false;
    let inOl = false;

    const closeLists = () => {
      if (inUl) {
        blocks.push("</ul>");
        inUl = false;
      }
      if (inOl) {
        blocks.push("</ol>");
        inOl = false;
      }
    };

    for (const rawLine of lines) {
      const line = rawLine.trim();

      if (!line) {
        closeLists();
        continue;
      }

      const headingMatch = line.match(/^(#{1,3})\s+(.*)$/);
      if (headingMatch) {
        closeLists();
        const level = headingMatch[1].length + 2;
        blocks.push(`<h${level}>${renderInlineMarkdown(headingMatch[2])}</h${level}>`);
        continue;
      }

      const orderedMatch = line.match(/^(\d+)\.\s+(.*)$/);
      if (orderedMatch) {
        if (inUl) {
          blocks.push("</ul>");
          inUl = false;
        }
        if (!inOl) {
          blocks.push("<ol>");
          inOl = true;
        }
        blocks.push(`<li>${renderInlineMarkdown(orderedMatch[2])}</li>`);
        continue;
      }

      const unorderedMatch = line.match(/^[-*]\s+(.*)$/);
      if (unorderedMatch) {
        if (inOl) {
          blocks.push("</ol>");
          inOl = false;
        }
        if (!inUl) {
          blocks.push("<ul>");
          inUl = true;
        }
        blocks.push(`<li>${renderInlineMarkdown(unorderedMatch[1])}</li>`);
        continue;
      }

      closeLists();
      blocks.push(`<p>${renderInlineMarkdown(line)}</p>`);
    }

    closeLists();
    return blocks.join("");
  }

  function updateDockOffset() {
    const base = window.innerWidth <= 700 ? 12 : 20;
    const executeBtn = document.querySelector(".btn-execute") as HTMLElement | null;
    if (!executeBtn) {
      dockBottomPx = base;
      return;
    }

    const styles = getComputedStyle(executeBtn);
    if (styles.display === "none" || styles.visibility === "hidden") {
      dockBottomPx = base;
      return;
    }

    const rect = executeBtn.getBoundingClientRect();
    const visible =
      rect.width > 0 &&
      rect.height > 0 &&
      rect.bottom > 0 &&
      rect.top < window.innerHeight;

    if (!visible) {
      dockBottomPx = base;
      return;
    }

    dockBottomPx = base + Math.round(rect.height + 16);
  }
</script>

{#if aiEnabled}
  <div class="assistant-widget" style={`--assistant-bottom: ${dockBottomPx}px;`}>
    {#if open}
      <section class="panel" role="dialog" aria-label={t($locale, "assistant.dialogLabel")}>
        <header class="panel-header">
          <div>
            <h3>{t($locale, "assistant.title")}</h3>
            <p>{t($locale, "assistant.subtitle")}</p>
          </div>
          <div class="header-actions">
            <button type="button" class="icon-btn" on:click={askResetConversation} title={t($locale, "assistant.resetChat")}>
              ⟲
            </button>
            <button type="button" class="icon-btn" on:click={toggleOpen} title={t($locale, "assistant.close")}>
              ✕
            </button>
          </div>
        </header>

        <div class="messages">
          {#if !historyLoaded}
            <div class="status">{t($locale, "assistant.loadingChat")}</div>
          {:else if messages.length === 0}
            <div class="empty">{t($locale, "assistant.empty")}</div>
          {:else}
            {#each messages as message}
              <article class="message" class:user={message.role === "user"}>
                {#if message.role === "assistant" && !message.loading}
                  <div class="md-content">{@html renderAssistantMarkdown(message.content)}</div>
                {:else}
                  <p>{message.content}</p>
                {/if}
                {#if message.loading}
                  <div class="typing-dots" aria-hidden="true">
                    <span></span><span></span><span></span>
                  </div>
                {/if}
              </article>
            {/each}
          {/if}
        </div>

        <div class="composer">
          <textarea
            bind:value={prompt}
            rows="3"
            placeholder={t($locale, "assistant.placeholder")}
            disabled={sending}
            on:keydown={handlePromptKeydown}
          ></textarea>
          <div class="composer-footer">
            {#if error}
              <span class="error">{error}</span>
            {:else}
              <span class="hint">{t($locale, "assistant.hintEnter")}</span>
            {/if}
            <button type="button" class="send-btn" on:click={sendMessage} disabled={sending || !prompt.trim()}>
              {#if sending}{t($locale, "assistant.sending")}{:else}{t($locale, "assistant.send")}{/if}
            </button>
          </div>
        </div>
      </section>
    {/if}

    <button
      class="fab"
      type="button"
      on:click={toggleOpen}
      aria-label={open ? t($locale, "assistant.closeFab") : t($locale, "assistant.open")}
      title={open ? t($locale, "assistant.closeFab") : t($locale, "assistant.open")}
    >
      <svg
        width="22"
        height="22"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5L12 3z" />
        <path d="M19 13l1 3 3 1-3 1-1 3-1-3-3-1 3-1 1-3z" />
      </svg>
    </button>
  </div>
{/if}

<ConfirmModal
  open={confirmResetOpen}
  title={t($locale, "assistant.resetConversationTitle")}
  message={t($locale, "assistant.resetConversationMessage")}
  confirmLabel={t($locale, "assistant.reset")}
  cancelLabel={t($locale, "common.cancel")}
  variant="warning"
  on:confirm={confirmResetConversation}
  on:cancel={cancelResetConversation}
/>

<style>
  .assistant-widget {
    --brand-indigo: #6366f1;
    --brand-violet: #8b5cf6;
    --brand-slate-900: #0f172a;
    --brand-slate-800: #1e293b;
    --brand-slate-600: #475569;
    --brand-slate-500: #64748b;
    --brand-slate-200: #e2e8f0;
    position: fixed;
    right: 20px;
    bottom: var(--assistant-bottom, 20px);
    z-index: 1200;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 12px;
    pointer-events: none;
  }

  .assistant-widget :global(*) {
    pointer-events: auto;
  }

  .fab {
    width: 58px;
    height: 58px;
    padding: 0;
    border: none;
    border-radius: 999px;
    background: linear-gradient(135deg, var(--brand-indigo) 0%, var(--brand-violet) 100%);
    color: #fff;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 0;
    cursor: pointer;
    box-shadow:
      0 10px 26px rgba(99, 102, 241, 0.42),
      inset 0 1px 0 rgba(255, 255, 255, 0.2);
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }

  .fab svg {
    display: block;
    width: 22px;
    height: 22px;
    flex: 0 0 auto;
  }

  .fab:hover {
    transform: translateY(-1px);
    box-shadow:
      0 14px 30px rgba(99, 102, 241, 0.5),
      inset 0 1px 0 rgba(255, 255, 255, 0.24);
  }

  .panel {
    width: min(420px, calc(100vw - 32px));
    max-height: min(72vh, 680px);
    background: #fff;
    border: 1px solid rgba(99, 102, 241, 0.22);
    border-radius: 16px;
    box-shadow:
      0 14px 36px rgba(30, 41, 59, 0.24),
      0 4px 12px rgba(99, 102, 241, 0.12);
    display: grid;
    grid-template-rows: auto 1fr auto;
    overflow: hidden;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
    padding: 12px 14px;
    border-bottom: 1px solid rgba(99, 102, 241, 0.2);
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.12) 0%, rgba(139, 92, 246, 0.1) 100%);
  }

  .panel-header h3 {
    margin: 0;
    font-size: 14px;
    color: var(--brand-slate-900);
  }

  .panel-header p {
    margin: 2px 0 0;
    font-size: 12px;
    color: var(--brand-slate-600);
  }

  .header-actions {
    display: inline-flex;
    gap: 6px;
  }

  .icon-btn {
    width: 28px;
    height: 28px;
    border: 1px solid rgba(99, 102, 241, 0.28);
    border-radius: 8px;
    background: #fff;
    color: #4c1d95;
    cursor: pointer;
    font-size: 14px;
  }

  .messages {
    padding: 12px;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
  }

  .status,
  .empty {
    margin: auto;
    color: var(--brand-slate-500);
    font-size: 13px;
    text-align: center;
  }

  .message {
    max-width: 88%;
    padding: 9px 10px;
    border: 1px solid var(--brand-slate-200);
    border-radius: 10px;
    background: #fff;
    color: var(--brand-slate-900);
    font-size: 13px;
    line-height: 1.45;
  }

  .message.user {
    margin-left: auto;
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.16) 0%, rgba(139, 92, 246, 0.14) 100%);
    border-color: rgba(99, 102, 241, 0.35);
    color: #312e81;
  }

  .message p {
    margin: 0;
    white-space: pre-wrap;
  }

  .md-content {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .md-content :global(h3),
  .md-content :global(h4),
  .md-content :global(h5) {
    margin: 0;
    font-size: 13px;
    color: var(--brand-slate-900);
    font-weight: 700;
  }

  .md-content :global(p) {
    margin: 0;
    color: inherit;
    white-space: pre-wrap;
  }

  .md-content :global(ol),
  .md-content :global(ul) {
    margin: 0;
    padding-left: 18px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .md-content :global(li) {
    line-height: 1.45;
  }

  .md-content :global(code) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    background: rgba(15, 23, 42, 0.06);
    border: 1px solid rgba(15, 23, 42, 0.12);
    border-radius: 5px;
    padding: 1px 4px;
    font-size: 12px;
  }

  .typing-dots {
    display: inline-flex;
    gap: 4px;
    margin-top: 6px;
  }

  .typing-dots span {
    width: 6px;
    height: 6px;
    border-radius: 999px;
    background: #94a3b8;
    animation: blink 1s infinite ease-in-out;
  }

  .typing-dots span:nth-child(2) {
    animation-delay: 0.2s;
  }

  .typing-dots span:nth-child(3) {
    animation-delay: 0.4s;
  }

  @keyframes blink {
    0%,
    80%,
    100% {
      opacity: 0.25;
      transform: translateY(0);
    }
    40% {
      opacity: 1;
      transform: translateY(-2px);
    }
  }

  .composer {
    border-top: 1px solid rgba(99, 102, 241, 0.2);
    background: #fff;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  textarea {
    width: 100%;
    min-height: 72px;
    resize: vertical;
    border: 1px solid rgba(99, 102, 241, 0.28);
    border-radius: 10px;
    padding: 10px 11px;
    font-size: 13px;
    font-family: inherit;
    color: var(--brand-slate-900);
    line-height: 1.45;
  }

  textarea:focus {
    outline: none;
    border-color: var(--brand-indigo);
    box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.18);
  }

  .composer-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .hint {
    font-size: 11px;
    color: var(--brand-slate-500);
  }

  .error {
    font-size: 11px;
    color: #b91c1c;
  }

  .send-btn {
    border: 1px solid #4f46e5;
    background: linear-gradient(135deg, var(--brand-indigo) 0%, var(--brand-violet) 100%);
    color: #fff;
    border-radius: 9px;
    padding: 8px 12px;
    font-size: 12px;
    font-weight: 700;
    cursor: pointer;
  }

  .send-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  @media (max-width: 700px) {
    .assistant-widget {
      right: 12px;
    }

    .panel {
      width: calc(100vw - 24px);
      max-height: 78vh;
    }
  }
</style>
