import { createHighlighter, type Highlighter } from 'shiki';

let promise: Promise<Highlighter> | null = null;

export function getHighlighter(): Promise<Highlighter> {
	if (!promise) {
		promise = createHighlighter({
			themes: ['github-light', 'github-dark'],
			langs: ['go', 'bash', 'typescript']
		});
	}
	return promise;
}
