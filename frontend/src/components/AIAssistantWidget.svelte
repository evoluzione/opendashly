<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import ConfirmModal from "./common/ConfirmModal.svelte";
  import {
    getAssistantSession,
    resetAssistantSession,
    saveAssistantSession,
    sendAssistantMessage,
    type AssistantMessage,
    type AssistantSuggestion,
    type AssistantContext,
    type AssistantLink,
    type AssistantRow,
  } from "../services/assistant";
  import { downloadLink, linkUrl, DOWNLOAD_LIMIT } from "../services/assistantLinks";
  import { locale, t } from "../lib/i18n";

  type ChatMessage = AssistantMessage & {
    id: string;
    loading?: boolean;
    steps?: string[];
    stepsShown?: number;
    suggestions?: AssistantSuggestion[];
    lang?: "it" | "en";
    links?: AssistantLink[];
    rows?: AssistantRow[];
  };

  const STEP_DELAY_MS = 300;
  const TYPE_FRAME_MS = 16;
  const TYPE_FRAMES = 60;

  let open = false;
  let expanded = false;
  let messagesEl: HTMLDivElement | null = null;
  let inputEl: HTMLTextAreaElement | null = null;
  let historyLoaded = false;
  let prompt = "";
  // sending: waiting for the backend. playing: an answer is being replayed;
  // a new message fast-forwards it instead of being ignored.
  let sending = false;
  let playing: Promise<void> | null = null;
  let fastForward = false;
  let messages: ChatMessage[] = [];
  let error = "";
  let confirmResetOpen = false;
  let destroyed = false;
  let dockBottomPx = 20;
  let layoutObserver: MutationObserver | null = null;
  let resizeHandler: (() => void) | null = null;

  onMount(async () => {
    await loadHistory();
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
    destroyed = true;
    if (resizeHandler) {
      window.removeEventListener("resize", resizeHandler);
      resizeHandler = null;
    }
    if (layoutObserver) {
      layoutObserver.disconnect();
      layoutObserver = null;
    }
  });

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
          context: m.context,
          suggestions: Array.isArray(m.suggestions) ? m.suggestions : undefined,
          links: Array.isArray(m.links) ? m.links : undefined,
          rows: Array.isArray(m.rows) ? m.rows : undefined,
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
      .map((m) => ({ role: m.role, content: m.content, context: m.context, suggestions: m.suggestions, links: m.links, rows: m.rows }));
    try {
      await saveAssistantSession({ messages: compact });
    } catch {
      // Keep UI reactive even if persistence fails.
    }
  }

  function toggleOpen() {
    open = !open;
    updateDockOffset();
    if (open) {
      // Show the latest messages, not the oldest ones.
      void scrollToBottom();
      void focusInput();
    }
  }

  async function focusInput() {
    await tick();
    inputEl?.focus();
  }

  // The context of the latest answer lets the backend resolve follow-ups.
  function lastContext(): AssistantContext | undefined {
    for (let i = messages.length - 1; i >= 0; i -= 1) {
      if (messages[i].role === "assistant" && !messages[i].loading) {
        return messages[i].context;
      }
    }
    return undefined;
  }

  function toggleExpanded() {
    expanded = !expanded;
    void scrollToBottom();
  }

  function handleWindowKeydown(event: KeyboardEvent) {
    if (event.key === "Escape" && expanded) {
      expanded = false;
    }
  }

  function prefersReducedMotion(): boolean {
    return typeof window !== "undefined" && window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  }

  function wait(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  function patchMessage(id: string, fields: Partial<ChatMessage>) {
    messages = messages.map((m) => (m.id === id ? { ...m, ...fields } : m));
  }

  async function scrollToBottom() {
    await tick();
    if (messagesEl) {
      messagesEl.scrollTop = messagesEl.scrollHeight;
    }
  }

  // Replays the steps the diagnosis really performed, then types the answer,
  // so the user can follow what was read and decided before the result.
  async function playResponse(id: string, steps: string[], answer: string, suggestions: AssistantSuggestion[], context?: AssistantContext, lang?: "it" | "en", links?: AssistantLink[], rows?: AssistantRow[]) {
    const animate = !prefersReducedMotion();
    patchMessage(id, { steps, stepsShown: 0, content: "", lang });
    for (let i = 1; i <= steps.length && animate && !destroyed && !fastForward; i += 1) {
      patchMessage(id, { stepsShown: i });
      await scrollToBottom();
      await wait(STEP_DELAY_MS);
    }
    patchMessage(id, { stepsShown: steps.length });
    const chunk = Math.max(3, Math.ceil(answer.length / TYPE_FRAMES));
    for (let n = chunk; n < answer.length && animate && !destroyed && !fastForward; n += chunk) {
      patchMessage(id, { content: answer.slice(0, n) });
      await scrollToBottom();
      await wait(TYPE_FRAME_MS);
    }
    patchMessage(id, { content: answer, loading: false, suggestions, context, links, rows });
    await scrollToBottom();
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

  function welcomeSuggestions(): AssistantSuggestion[] {
    return [
      { label: t($locale, "assistant.quickHour"), prompt: t($locale, "assistant.quickHourPrompt") },
      { label: t($locale, "assistant.quickDay"), prompt: t($locale, "assistant.quickDayPrompt") },
      { label: t($locale, "assistant.quickToday"), prompt: t($locale, "assistant.quickTodayPrompt") },
      { label: t($locale, "assistant.quickHelp"), prompt: t($locale, "assistant.quickHelp") },
    ];
  }

  // downloading holds the key of the link being downloaded, to show progress.
  let downloading = "";

  async function download(messageId: string, index: number, link: AssistantLink) {
    const key = `${messageId}:${index}`;
    if (downloading) return;
    downloading = key;
    error = "";
    try {
      await downloadLink(link);
    } catch {
      error = t($locale, "assistant.downloadError");
    } finally {
      downloading = "";
    }
  }

  function pick(suggestion: AssistantSuggestion) {
    void sendMessage(suggestion.label, suggestion.prompt);
  }

  // Follow-ups are offered on the latest answer only: they run on its context.
  $: isLast = (index: number) => index === messages.length - 1 && !sending;
  const inline = (s?: AssistantSuggestion[]) => (s ?? []).filter((x) => x.kind === "inline");
  const reruns = (s?: AssistantSuggestion[]) => (s ?? []).filter((x) => x.kind === "rerun");
  const others = (s?: AssistantSuggestion[]) => (s ?? []).filter((x) => x.kind !== "inline" && x.kind !== "rerun");

  // rowActions adds "Analizza" and "Apri ↗" to each item of the answer's
  // numbered list: the markdown comes in as HTML, so the rows are found in the
  // DOM once it is rendered. Row i matches the i-th item of the first list.
  function rowActions(node: HTMLElement, params: { rows?: AssistantRow[]; lang: Parameters<typeof t>[0]; content: string }) {
    function apply({ rows, lang }: typeof params) {
      node.querySelectorAll(".row-actions").forEach((el) => el.remove());
      if (!rows?.length) return;
      const items = node.querySelector("ol")?.querySelectorAll(":scope > li") ?? [];
      items.forEach((li, i) => {
        const row = rows[i];
        if (!row) return;
        const box = document.createElement("span");
        box.className = "row-actions";
        const analyze = document.createElement("button");
        analyze.type = "button";
        analyze.textContent = t(lang, "assistant.analyzeRow");
        analyze.title = row.prompt;
        analyze.addEventListener("click", () => void sendMessage(row.prompt));
        box.append(analyze);
        if (row.link) {
          const a = document.createElement("a");
          a.href = linkUrl(row.link);
          a.target = "_blank";
          a.rel = "noopener";
          a.textContent = `${t(lang, "assistant.openRow")} ↗`;
          a.title = t(lang, "assistant.openInSearch");
          box.append(a);
        }
        li.append(box);
      });
    }
    apply(params);
    return { update: apply };
  }

  async function sendMessage(display = prompt.trim(), request = display) {
    if (!display || sending) return;
    if (playing) {
      fastForward = true;
      await playing;
    }
    fastForward = false;

    const context = lastContext();
    sending = true;
    error = "";

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: "user",
      content: display,
    };
    const placeholder: ChatMessage = {
      id: `assistant-${Date.now()}`,
      role: "assistant",
      content: "",
      loading: true,
      steps: [t($locale, "assistant.working")],
      stepsShown: 1,
    };

    messages = [...messages, userMessage, placeholder];
    prompt = "";
    void scrollToBottom();

    let current: Promise<void> | null = null;
    try {
      const response = await sendAssistantMessage({
        prompt: request,
        locale: $locale,
        context,
      });
      sending = false;
      current = playResponse(placeholder.id, response.steps ?? [], response.answer, response.suggestions ?? [], response.context, response.lang, response.links, response.rows);
      playing = current;
      await current;
      await persistHistory();
    } catch (err) {
      error = err instanceof Error ? err.message : t($locale, "assistant.errorGeneric");
      patchMessage(placeholder.id, {
        loading: false,
        steps: undefined,
        content: t($locale, "assistant.errorReply"),
      });
      await persistHistory();
    } finally {
      sending = false;
      if (playing === current) {
        playing = null;
      }
      void focusInput();
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

<svelte:window on:keydown={handleWindowKeydown} />

<div class="assistant-widget" style={`--assistant-bottom: ${dockBottomPx}px;`}>
    {#if open && expanded}
      <button type="button" class="backdrop" aria-label={t($locale, "assistant.collapse")} on:click={toggleExpanded}></button>
    {/if}
    {#if open}
      <section class="panel" class:expanded role="dialog" aria-label={t($locale, "assistant.dialogLabel")}>
        <header class="panel-header">
          <div>
            <h3>{t($locale, "assistant.title")}</h3>
            <p>{t($locale, "assistant.subtitle")}</p>
          </div>
          <div class="header-actions">
            <button type="button" class="icon-btn" on:click={askResetConversation} title={t($locale, "assistant.resetChat")}>
              ⟲
            </button>
            <button
              type="button"
              class="icon-btn"
              on:click={toggleExpanded}
              title={expanded ? t($locale, "assistant.collapse") : t($locale, "assistant.expand")}
              aria-label={expanded ? t($locale, "assistant.collapse") : t($locale, "assistant.expand")}
            >
              {expanded ? "⤡" : "⤢"}
            </button>
            <button type="button" class="icon-btn" on:click={toggleOpen} title={t($locale, "assistant.close")}>
              ✕
            </button>
          </div>
        </header>

        <div class="messages" bind:this={messagesEl}>
          {#if !historyLoaded}
            <div class="status">{t($locale, "assistant.loadingChat")}</div>
          {:else}
            <article class="message">
              <div class="md-content"><p>{t($locale, "assistant.welcome")}</p></div>
              {#if messages.length === 0}
                <div class="suggestions">
                  {#each welcomeSuggestions() as suggestion}
                    <button type="button" disabled={sending} on:click={() => pick(suggestion)}>{suggestion.label}</button>
                  {/each}
                </div>
              {/if}
            </article>
            {#each messages as message, index (message.id)}
              <article class="message" class:user={message.role === "user"}>
                {#if message.steps?.length}
                  {#if message.loading && !message.content}
                    <ol class="steps" aria-live="polite">
                      {#each message.steps.slice(0, message.stepsShown ?? 0) as step, i}
                        <li class:done={i < (message.stepsShown ?? 0) - 1}>
                          <span class="step-icon" aria-hidden="true"></span>{step}
                        </li>
                      {/each}
                    </ol>
                  {:else}
                    <details class="steps-summary">
                      <summary>{t(message.lang ?? $locale, "assistant.stepsDone", { count: message.steps.length })}</summary>
                      <ol class="steps">
                        {#each message.steps as step}
                          <li class="done"><span class="step-icon" aria-hidden="true"></span>{step}</li>
                        {/each}
                      </ol>
                    </details>
                  {/if}
                {/if}
                {#if message.role === "assistant"}
                  {#if message.content}
                    <div
                      class="md-content"
                      use:rowActions={{ rows: message.loading ? undefined : message.rows, lang: message.lang ?? $locale, content: message.content }}
                    >{@html renderAssistantMarkdown(message.content)}</div>
                  {/if}
                {:else}
                  <p>{message.content}</p>
                {/if}
                {#if message.loading && message.content}
                  <span class="caret" aria-hidden="true"></span>
                {/if}
                {#if !message.loading && ((isLast(index) && inline(message.suggestions).length) || message.links?.length)}
                  {@const lang = message.lang ?? $locale}
                  <div class="bubble-footer">
                    {#if isLast(index)}
                      {#each inline(message.suggestions) as suggestion}
                        <button type="button" class="inline-action" title={suggestion.prompt} on:click={() => pick(suggestion)}>{suggestion.label}</button>
                      {/each}
                    {/if}
                    {#if message.links?.length}
                      <details class="more">
                        <summary title={t(lang, "assistant.more")} aria-label={t(lang, "assistant.more")}>⋯</summary>
                        <div class="more-menu">
                          {#each message.links as link, i}
                            <a href={linkUrl(link)} target="_blank" rel="noopener" title={t(lang, "assistant.openInSearch")}
                              >{t(lang, "assistant.openLink")} {link.label.toLowerCase()} ↗</a>
                            <button type="button" disabled={downloading !== ""} on:click={() => download(message.id, i, link)}>
                              {downloading === `${message.id}:${i}`
                                ? "…"
                                : link.kind === "trace"
                                  ? t(lang, "assistant.downloadJson")
                                  : t(lang, "assistant.downloadCsv", { count: DOWNLOAD_LIMIT })}
                            </button>
                          {/each}
                        </div>
                      </details>
                    {/if}
                  </div>
                {/if}
              </article>
              {#if isLast(index) && !message.loading && (reruns(message.suggestions).length || others(message.suggestions).length)}
                {@const lang = message.lang ?? $locale}
                <div class="next">
                  {#if reruns(message.suggestions).length}
                    <div class="chips">
                      <span class="chips-lead">↻ {t(lang, "assistant.rerunFor")}</span>
                      {#each reruns(message.suggestions) as suggestion}
                        <button type="button" title={suggestion.prompt} on:click={() => pick(suggestion)}>{suggestion.label}</button>
                      {/each}
                    </div>
                  {/if}
                  {#if others(message.suggestions).length}
                    <div class="chips">
                      {#each others(message.suggestions) as suggestion}
                        <button type="button" title={suggestion.prompt} on:click={() => pick(suggestion)}>{suggestion.label}</button>
                      {/each}
                    </div>
                  {/if}
                </div>
              {/if}
            {/each}
          {/if}
        </div>

        <div class="composer">
          <textarea
            bind:this={inputEl}
            bind:value={prompt}
            rows="3"
            placeholder={t($locale, "assistant.placeholder")}
            on:keydown={handlePromptKeydown}
          ></textarea>
          <div class="composer-footer">
            {#if error}
              <span class="error">{error}</span>
            {:else}
              <span class="hint">{t($locale, "assistant.hintEnter")}</span>
            {/if}
            <button type="button" class="send-btn" on:click={() => sendMessage()} disabled={sending || !prompt.trim()}>
              {#if sending}{t($locale, "assistant.sending")}{:else}{t($locale, "assistant.send")}{/if}
            </button>
          </div>
        </div>
      </section>
    {/if}

    <button
      class="fab"
      class:pulse={!open}
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
        <circle cx="9" cy="10" r="3.2" />
        <path d="M3 21c0-3.3 2.7-5.8 6-5.8s6 2.5 6 5.8" />
        <path d="M14 3h6.5A1.5 1.5 0 0 1 22 4.5v4A1.5 1.5 0 0 1 20.5 10H18l-2.5 2v-2H14a1.5 1.5 0 0 1-1.5-1.5v-4A1.5 1.5 0 0 1 14 3z" />
      </svg>
    </button>
  </div>

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
    --brand-indigo: var(--color-primary-600);
    --brand-violet: var(--color-primary-500);
    --brand-slate-900: var(--color-slate-950);
    --brand-slate-800: var(--color-slate-900);
    --brand-slate-600: var(--color-slate-600);
    --brand-slate-500: var(--color-slate-500);
    --brand-slate-200: var(--color-slate-200);
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
    color: var(--color-white);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 0;
    cursor: pointer;
    box-shadow:
      0 10px 26px rgba(var(--rgb-primary-600), 0.42),
      inset 0 1px 0 rgba(255, 255, 255, 0.2);
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }

  .fab {
    position: relative;
  }

  .fab.pulse::after {
    content: "";
    position: absolute;
    inset: 0;
    border-radius: inherit;
    box-shadow: 0 0 0 0 rgba(var(--rgb-primary-600), 0.55);
    animation: fab-pulse 2s ease-out infinite;
    pointer-events: none;
  }

  @keyframes fab-pulse {
    0% {
      box-shadow: 0 0 0 0 rgba(var(--rgb-primary-600), 0.55);
    }
    70%,
    100% {
      box-shadow: 0 0 0 16px rgba(var(--rgb-primary-600), 0);
    }
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
      0 14px 30px rgba(var(--rgb-primary-600), 0.5),
      inset 0 1px 0 rgba(255, 255, 255, 0.24);
  }

  .panel {
    width: min(720px, calc(100vw - 32px));
    height: min(80vh, 800px);
    background: var(--color-white);
    border: 1px solid rgba(var(--rgb-primary-600), 0.22);
    border-radius: 16px;
    box-shadow:
      0 14px 36px rgba(30, 41, 59, 0.24),
      0 4px 12px rgba(var(--rgb-primary-600), 0.12);
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
    border-bottom: 1px solid rgba(var(--rgb-primary-600), 0.2);
    background: linear-gradient(135deg, rgba(var(--rgb-primary-600), 0.12) 0%, rgba(139, 92, 246, 0.1) 100%);
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
    border: 1px solid rgba(var(--rgb-primary-600), 0.28);
    border-radius: 8px;
    background: var(--color-white);
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
    background: linear-gradient(180deg, var(--color-slate-50) 0%, var(--color-slate-100) 100%);
  }

  .status {
    margin: auto;
    color: var(--brand-slate-500);
    font-size: 13px;
    text-align: center;
  }

  .bubble-footer {
    display: flex;
    align-items: center;
    gap: 4px 12px;
    flex-wrap: wrap;
    margin-top: 8px;
    padding-top: 6px;
    border-top: 1px solid var(--brand-slate-200);
    font-size: 12px;
  }

  .inline-action,
  .md-content :global(.row-actions button),
  .md-content :global(.row-actions a) {
    border: none;
    background: none;
    padding: 0;
    color: var(--brand-indigo);
    font: inherit;
    font-size: 12px;
    font-weight: 600;
    text-decoration: none;
    cursor: pointer;
  }

  .inline-action:hover,
  .md-content :global(.row-actions button:hover),
  .md-content :global(.row-actions a:hover) {
    text-decoration: underline;
  }

  /* Row actions float over the right end of the row being pointed at, so
     they never take room in the text; on touch screens they sit inline. */
  .md-content :global(ol > li) {
    position: relative;
  }

  .md-content :global(.row-actions) {
    position: absolute;
    top: -4px;
    right: 0;
    display: none;
    gap: 10px;
    padding: 2px 8px;
    background: var(--color-white);
    border: 1px solid var(--brand-slate-200);
    border-radius: 6px;
    box-shadow: 0 2px 8px rgba(30, 41, 59, 0.12);
  }

  .md-content :global(li:hover > .row-actions),
  .md-content :global(li:focus-within > .row-actions) {
    display: inline-flex;
  }

  @media (hover: none) {
    .md-content :global(.row-actions) {
      position: static;
      display: inline-flex;
      margin-left: 8px;
      padding: 0;
      border: none;
      box-shadow: none;
      background: none;
    }
  }

  .more {
    position: relative;
    margin-left: auto;
  }

  .more summary {
    list-style: none;
    cursor: pointer;
    padding: 0 6px;
    border-radius: 6px;
    color: var(--brand-slate-500);
    font-size: 16px;
    line-height: 20px;
  }

  .more summary::-webkit-details-marker {
    display: none;
  }

  .more summary:hover,
  .more[open] summary {
    background: var(--color-slate-100);
    color: var(--brand-slate-900);
  }

  .more-menu {
    position: absolute;
    right: 0;
    bottom: calc(100% + 4px);
    z-index: 2;
    display: grid;
    grid-template-columns: auto auto;
    gap: 6px 14px;
    padding: 8px 10px;
    white-space: nowrap;
    background: var(--color-white);
    border: 1px solid var(--brand-slate-200);
    border-radius: 8px;
    box-shadow: 0 6px 18px rgba(30, 41, 59, 0.14);
  }

  .more-menu a,
  .more-menu button {
    border: none;
    background: none;
    padding: 0;
    font: inherit;
    font-size: 12px;
    text-align: left;
    color: var(--brand-indigo);
    text-decoration: none;
    cursor: pointer;
  }

  .more-menu button {
    color: var(--brand-slate-600);
  }

  .more-menu a:hover,
  .more-menu button:hover:not(:disabled) {
    text-decoration: underline;
  }

  .more-menu button:disabled {
    cursor: progress;
  }

  .next {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-width: 88%;
    margin: -2px 0 4px 4px;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
  }

  .chips-lead {
    font-size: 12px;
    color: var(--brand-slate-500);
  }

  .chips button {
    border: 1px solid var(--color-slate-300);
    background: var(--color-white);
    color: var(--brand-slate-800);
    border-radius: 999px;
    padding: 3px 10px;
    font-size: 12px;
    cursor: pointer;
  }

  .chips button:hover {
    border-color: rgba(var(--rgb-primary-600), 0.5);
    color: var(--brand-indigo);
  }

  .suggestions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 10px;
  }

  .suggestions button {
    border: 1px solid rgba(var(--rgb-primary-600), 0.35);
    background: var(--color-white);
    color: var(--brand-indigo);
    border-radius: 999px;
    padding: 5px 12px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  .suggestions button:hover:not(:disabled) {
    background: rgba(var(--rgb-primary-600), 0.08);
  }

  .suggestions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .steps {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    font-size: 12px;
    color: var(--brand-slate-600);
  }

  .steps li {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    animation: step-in 0.25s ease-out;
  }

  .step-icon {
    flex: 0 0 auto;
    width: 12px;
    height: 12px;
    margin-top: 2px;
    border-radius: 999px;
    border: 2px solid rgba(var(--rgb-primary-600), 0.25);
    border-top-color: var(--brand-indigo);
    animation: spin 0.8s linear infinite;
  }

  .steps li.done .step-icon {
    border: none;
    animation: none;
    background: var(--color-success-500, #22c55e);
    mask: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 12 12'%3E%3Cpath d='M2.5 6.2 5 8.5l4.5-5' fill='none' stroke='black' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E") center / contain no-repeat;
  }

  .steps-summary {
    margin-bottom: 8px;
    font-size: 12px;
    color: var(--brand-slate-500);
  }

  .steps-summary summary {
    cursor: pointer;
    margin-bottom: 6px;
  }

  .caret {
    display: inline-block;
    width: 7px;
    height: 14px;
    margin-top: 4px;
    background: var(--brand-indigo);
    animation: blink 1s infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  @keyframes step-in {
    from {
      opacity: 0;
      transform: translateY(3px);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .fab.pulse::after,
    .steps li,
    .step-icon,
    .caret {
      animation: none;
    }
  }

  .panel.expanded {
    position: fixed;
    top: 24px;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    width: min(1280px, calc(100vw - 48px));
    height: auto;
    z-index: 1;
  }

  .panel.expanded .messages {
    padding: 16px 20px;
  }

  .backdrop {
    position: fixed;
    inset: 0;
    border: none;
    padding: 0;
    background: rgba(15, 23, 42, 0.45);
    cursor: default;
  }

  .message {
    max-width: 88%;
    padding: 10px 12px;
    border: 1px solid var(--brand-slate-200);
    border-radius: 12px;
    background: var(--color-white);
    color: var(--brand-slate-900);
    font-size: 14px;
    line-height: 1.45;
  }

  .message.user {
    margin-left: auto;
    background: linear-gradient(135deg, rgba(var(--rgb-primary-600), 0.16) 0%, rgba(139, 92, 246, 0.14) 100%);
    border-color: rgba(var(--rgb-primary-600), 0.35);
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
    background: rgba(var(--rgb-slate-950), 0.06);
    border: 1px solid rgba(var(--rgb-slate-950), 0.12);
    border-radius: 5px;
    padding: 1px 4px;
    font-size: 12px;
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
    border-top: 1px solid rgba(var(--rgb-primary-600), 0.2);
    background: var(--color-white);
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  textarea {
    width: 100%;
    min-height: 72px;
    resize: vertical;
    border: 1px solid rgba(var(--rgb-primary-600), 0.28);
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
    box-shadow: 0 0 0 3px rgba(var(--rgb-primary-600), 0.18);
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
    color: var(--color-danger-700);
  }

  .send-btn {
    border: 1px solid var(--color-primary-700);
    background: linear-gradient(135deg, var(--brand-indigo) 0%, var(--brand-violet) 100%);
    color: var(--color-white);
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
      height: 80vh;
    }

    .panel.expanded {
      top: 0;
      bottom: 0;
      width: 100vw;
      border-radius: 0;
    }
  }
</style>
