/**
 * Format a connection URL for display in the UI.
 * Strips protocol (nats://, wss://, ws://) and shortens localhost to "local".
 *
 * @param url - The URL to format (e.g., "ws://localhost:9222")
 * @returns Formatted URL string (e.g., "local:9222")
 */
export function formatNatsUrl(url: string): string {
	if (!url) return 'Unknown';

	let formatted = url.replace(/^(nats\+tls|nats|wss|ws):\/\//, '');
	formatted = formatted.replace(/^localhost/, 'local');

	return formatted;
}

/**
 * Format a Grafana URL for display and linking
 * Adds http/https protocol if missing
 *
 * @param url - The Grafana URL to format
 * @returns Formatted URL with protocol
 */
export function formatGrafanaUrl(url: string): string {
	if (!url) return '';

	// Already has protocol
	if (url.startsWith('http://') || url.startsWith('https://')) {
		return url;
	}

	// Localhost gets http
	if (url.startsWith('localhost')) {
		return `http://${url}`;
	}

	// Everything else gets https
	return `https://${url}`;
}
