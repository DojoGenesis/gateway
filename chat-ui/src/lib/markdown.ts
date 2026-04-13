/**
 * Minimal markdown renderer — no external dependencies.
 * Handles: code blocks, inline code, bold, italic, links, headings, lists.
 * Output is sanitized HTML safe for innerHTML insertion.
 */

function escapeHtml(text: string): string {
	return text
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;')
		.replace(/'/g, '&#39;');
}

function renderInline(text: string): string {
	// Bold+italic: ***text***
	text = text.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>');
	// Bold: **text**
	text = text.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
	// Italic: *text* or _text_
	text = text.replace(/\*(.+?)\*/g, '<em>$1</em>');
	text = text.replace(/_(.+?)_/g, '<em>$1</em>');
	// Inline code: `code`
	text = text.replace(/`([^`]+)`/g, (_, code: string) => `<code>${escapeHtml(code)}</code>`);
	// Links: [label](url)
	text = text.replace(
		/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g,
		(_, label: string, url: string) =>
			`<a href="${escapeHtml(url)}" target="_blank" rel="noopener noreferrer">${escapeHtml(label)}</a>`
	);
	return text;
}

export function renderMarkdown(raw: string): string {
	const lines = raw.split('\n');
	const output: string[] = [];
	let inCodeBlock = false;
	let codeLang = '';
	let codeLines: string[] = [];
	let inList = false;

	const flushList = () => {
		if (inList) {
			output.push('</ul>');
			inList = false;
		}
	};

	for (let i = 0; i < lines.length; i++) {
		const line = lines[i];

		// Fenced code block start/end
		const fenceMatch = /^```(\w*)/.exec(line);
		if (fenceMatch && !inCodeBlock) {
			flushList();
			inCodeBlock = true;
			codeLang = fenceMatch[1] ?? '';
			codeLines = [];
			continue;
		}
		if (inCodeBlock) {
			if (line.startsWith('```')) {
				// End code block
				const langClass = codeLang ? ` class="language-${escapeHtml(codeLang)}"` : '';
				output.push(
					`<pre><code${langClass}>${escapeHtml(codeLines.join('\n'))}</code></pre>`
				);
				inCodeBlock = false;
				codeLines = [];
				codeLang = '';
			} else {
				codeLines.push(line);
			}
			continue;
		}

		// Headings
		const h3 = /^### (.+)/.exec(line);
		if (h3) { flushList(); output.push(`<h3>${renderInline(escapeHtml(h3[1]))}</h3>`); continue; }
		const h2 = /^## (.+)/.exec(line);
		if (h2) { flushList(); output.push(`<h2>${renderInline(escapeHtml(h2[1]))}</h2>`); continue; }
		const h1 = /^# (.+)/.exec(line);
		if (h1) { flushList(); output.push(`<h1>${renderInline(escapeHtml(h1[1]))}</h1>`); continue; }

		// Unordered list items
		const li = /^[-*+] (.+)/.exec(line);
		if (li) {
			if (!inList) { output.push('<ul>'); inList = true; }
			output.push(`<li>${renderInline(escapeHtml(li[1]))}</li>`);
			continue;
		}

		// Empty line → paragraph break
		if (line.trim() === '') {
			flushList();
			output.push('<br>');
			continue;
		}

		// Normal paragraph line
		flushList();
		output.push(`<p>${renderInline(escapeHtml(line))}</p>`);
	}

	// Close any open code block
	if (inCodeBlock && codeLines.length > 0) {
		const langClass = codeLang ? ` class="language-${escapeHtml(codeLang)}"` : '';
		output.push(`<pre><code${langClass}>${escapeHtml(codeLines.join('\n'))}</code></pre>`);
	}
	flushList();

	return output.join('\n');
}
