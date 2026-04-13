import { h as head, a as attr_class, b as ensure_array_like, e as escape_html, s as stringify, c as attr } from "../../chunks/renderer.js";
import "@sveltejs/kit/internal";
import "../../chunks/exports.js";
import "../../chunks/utils.js";
import "@sveltejs/kit/internal/server";
import "../../chunks/root.js";
import "../../chunks/state.svelte.js";
import "clsx";
function html(value) {
  var html2 = String(value ?? "");
  var open = "<!---->";
  return open + html2 + "<!---->";
}
const messages = [];
const conversations = [];
const activeConversationId = { value: "" };
const models = [];
const selectedModel = { value: "" };
const isStreaming = { value: false };
function escapeHtml(text) {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
}
function renderInline(text) {
  text = text.replace(/\*\*\*(.+?)\*\*\*/g, "<strong><em>$1</em></strong>");
  text = text.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>");
  text = text.replace(/\*(.+?)\*/g, "<em>$1</em>");
  text = text.replace(/_(.+?)_/g, "<em>$1</em>");
  text = text.replace(/`([^`]+)`/g, (_, code) => `<code>${escapeHtml(code)}</code>`);
  text = text.replace(
    /\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g,
    (_, label, url) => `<a href="${escapeHtml(url)}" target="_blank" rel="noopener noreferrer">${escapeHtml(label)}</a>`
  );
  return text;
}
function renderMarkdown(raw) {
  const lines = raw.split("\n");
  const output = [];
  let inCodeBlock = false;
  let codeLang = "";
  let codeLines = [];
  let inList = false;
  const flushList = () => {
    if (inList) {
      output.push("</ul>");
      inList = false;
    }
  };
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const fenceMatch = /^```(\w*)/.exec(line);
    if (fenceMatch && !inCodeBlock) {
      flushList();
      inCodeBlock = true;
      codeLang = fenceMatch[1] ?? "";
      codeLines = [];
      continue;
    }
    if (inCodeBlock) {
      if (line.startsWith("```")) {
        const langClass = codeLang ? ` class="language-${escapeHtml(codeLang)}"` : "";
        output.push(
          `<pre><code${langClass}>${escapeHtml(codeLines.join("\n"))}</code></pre>`
        );
        inCodeBlock = false;
        codeLines = [];
        codeLang = "";
      } else {
        codeLines.push(line);
      }
      continue;
    }
    const h3 = /^### (.+)/.exec(line);
    if (h3) {
      flushList();
      output.push(`<h3>${renderInline(escapeHtml(h3[1]))}</h3>`);
      continue;
    }
    const h2 = /^## (.+)/.exec(line);
    if (h2) {
      flushList();
      output.push(`<h2>${renderInline(escapeHtml(h2[1]))}</h2>`);
      continue;
    }
    const h1 = /^# (.+)/.exec(line);
    if (h1) {
      flushList();
      output.push(`<h1>${renderInline(escapeHtml(h1[1]))}</h1>`);
      continue;
    }
    const li = /^[-*+] (.+)/.exec(line);
    if (li) {
      if (!inList) {
        output.push("<ul>");
        inList = true;
      }
      output.push(`<li>${renderInline(escapeHtml(li[1]))}</li>`);
      continue;
    }
    if (line.trim() === "") {
      flushList();
      output.push("<br>");
      continue;
    }
    flushList();
    output.push(`<p>${renderInline(escapeHtml(line))}</p>`);
  }
  if (inCodeBlock && codeLines.length > 0) {
    const langClass = codeLang ? ` class="language-${escapeHtml(codeLang)}"` : "";
    output.push(`<pre><code${langClass}>${escapeHtml(codeLines.join("\n"))}</code></pre>`);
  }
  flushList();
  return output.join("\n");
}
function _page($$renderer, $$props) {
  $$renderer.component(($$renderer2) => {
    let inputValue = "";
    function formatDate(iso) {
      try {
        const d = new Date(iso);
        const now = /* @__PURE__ */ new Date();
        const diff = now.getTime() - d.getTime();
        if (diff < 6e4) return "just now";
        if (diff < 36e5) return `${Math.floor(diff / 6e4)}m ago`;
        if (diff < 864e5) return `${Math.floor(diff / 36e5)}h ago`;
        return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
      } catch {
        return iso;
      }
    }
    head("1uha8ag", $$renderer2, ($$renderer3) => {
      $$renderer3.title(($$renderer4) => {
        $$renderer4.push(`<title>Dojo Chat</title>`);
      });
    });
    $$renderer2.push(`<div class="app-shell"><aside${attr_class("sidebar", void 0, { "collapsed": false })}><div class="sidebar-header"><span class="sidebar-title">Conversations</span> <button class="btn-icon" title="New conversation"><svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75"><line x1="8" y1="2" x2="8" y2="14"></line><line x1="2" y1="8" x2="14" y2="8"></line></svg></button></div> <div class="conversation-list">`);
    if (conversations.length === 0) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<p class="sidebar-empty">No conversations yet</p>`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<!--[-->`);
      const each_array = ensure_array_like(conversations);
      for (let $$index = 0, $$length = each_array.length; $$index < $$length; $$index++) {
        let conv = each_array[$$index];
        $$renderer2.push(`<button${attr_class("conversation-item", void 0, { "active": activeConversationId.value === conv.id })}><span class="conv-title">${escape_html(conv.title)}</span> <span class="conv-meta">${escape_html(formatDate(conv.updated_at))} · ${escape_html(conv.message_count)} msg</span></button>`);
      }
      $$renderer2.push(`<!--]-->`);
    }
    $$renderer2.push(`<!--]--></div></aside> <main class="main-panel"><header class="chat-header"><div class="header-left"><button class="btn-icon" title="Toggle sidebar"><svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75"><line x1="2" y1="4" x2="14" y2="4"></line><line x1="2" y1="8" x2="14" y2="8"></line><line x1="2" y1="12" x2="14" y2="12"></line></svg></button> <span class="app-logo">Dojo <span>Chat</span></span></div> <div class="header-right">`);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--> <button class="btn-text">Sign out</button></div></header> <div class="message-thread"><div class="thread-inner">`);
    if (messages.length === 0) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.push(`<div class="empty-state"><p class="empty-state-title">What would you like to explore?</p> <p class="empty-state-sub">Type a message below to start a conversation.</p></div>`);
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<!--[-->`);
      const each_array_1 = ensure_array_like(messages);
      for (let i = 0, $$length = each_array_1.length; i < $$length; i++) {
        let msg = each_array_1[i];
        $$renderer2.push(`<div${attr_class(`message-row ${stringify(msg.role)}`)}><div class="message-bubble">`);
        if (msg.role === "assistant") {
          $$renderer2.push("<!--[0-->");
          $$renderer2.push(`<div class="markdown-content" role="article" aria-label="Assistant message">${html(renderMarkdown(msg.content))}</div>`);
        } else {
          $$renderer2.push("<!--[-1-->");
          $$renderer2.push(`${escape_html(msg.content)}`);
        }
        $$renderer2.push(`<!--]--></div></div>`);
      }
      $$renderer2.push(`<!--]-->`);
    }
    $$renderer2.push(`<!--]--> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]--></div></div> <div class="input-bar"><div class="input-bar-inner"><div class="input-controls"><div class="input-wrapper"><textarea class="chat-textarea" placeholder="Message Dojo Chat…" rows="1"${attr("disabled", isStreaming.value, true)}>`);
    const $$body = escape_html(inputValue);
    if ($$body) {
      $$renderer2.push(`${$$body}`);
    }
    $$renderer2.push(`</textarea></div> <button class="btn-send" title="Send message"${attr("disabled", !inputValue.trim() || isStreaming.value, true)}><svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M2 14L14 8 2 2v4.5l8 1.5-8 1.5z"></path></svg></button></div> <div class="input-meta">`);
    if (models.length > 0) {
      $$renderer2.push("<!--[0-->");
      $$renderer2.select(
        {
          class: "model-select",
          value: selectedModel.value,
          disabled: isStreaming.value
        },
        ($$renderer3) => {
          $$renderer3.push(`<!--[-->`);
          const each_array_2 = ensure_array_like(models);
          for (let $$index_2 = 0, $$length = each_array_2.length; $$index_2 < $$length; $$index_2++) {
            let m = each_array_2[$$index_2];
            $$renderer3.option({ value: m.id }, ($$renderer4) => {
              $$renderer4.push(`${escape_html(m.id)}`);
            });
          }
          $$renderer3.push(`<!--]-->`);
        }
      );
    } else {
      $$renderer2.push("<!--[-1-->");
      $$renderer2.push(`<span class="input-hint">No models loaded</span>`);
    }
    $$renderer2.push(`<!--]--> <span class="input-hint">Enter to send · Shift+Enter for newline</span></div></div></div></main></div> `);
    {
      $$renderer2.push("<!--[-1-->");
    }
    $$renderer2.push(`<!--]-->`);
  });
}
export {
  _page as default
};
