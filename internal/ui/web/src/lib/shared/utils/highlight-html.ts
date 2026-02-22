import { getHighlighter } from './highlighter';

export async function highlightHtml(html: string): Promise<string> {
	const hl = await getHighlighter();
	const parser = new DOMParser();
	const doc = parser.parseFromString(`<body>${html}</body>`, 'text/html');
	const blocks = doc.querySelectorAll<HTMLElement>('pre code[data-lang]');
	for (const code of blocks) {
		const lang = (code.dataset.lang ?? 'bash') as 'go' | 'bash' | 'typescript';
		const text = code.textContent ?? '';
		const highlighted = hl.codeToHtml(text, {
			lang,
			themes: { light: 'github-light', dark: 'github-dark' },
			defaultColor: false
		});
		const pre = code.closest('pre');
		if (pre) {
			const tmp = document.createElement('div');
			tmp.innerHTML = highlighted;
			if (tmp.firstElementChild) pre.replaceWith(tmp.firstElementChild);
		}
	}
	return doc.body.innerHTML;
}
