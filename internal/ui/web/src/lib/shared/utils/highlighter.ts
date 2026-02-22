import { getSingletonHighlighter } from 'shiki';

export function getHighlighter() {
	return getSingletonHighlighter({
		themes: ['github-light', 'github-dark'],
		langs: ['go', 'bash', 'typescript', 'json']
	});
}
