/**
 * Format a NATS URL for display in the UI
 * Strips protocol and shortens localhost references
 *
 * @param url - The NATS URL to format (e.g., "nats://localhost:4222")
 * @returns Formatted URL string (e.g., "local:4222")
 */
export function formatNatsUrl(url: string): string {
	if (!url) return 'Unknown';

	// Remove nats:// protocol
	let formatted = url.replace(/^nats:\/\//, '');

	// Shorten localhost to local
	formatted = formatted.replace(/^localhost:/, 'local:');

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
