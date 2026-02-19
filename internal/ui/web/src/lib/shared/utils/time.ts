export function relativeTime(seconds: bigint | number): string {
	const diff = Date.now() - Number(seconds) * 1000;
	const minutes = Math.floor(diff / 60000);
	if (minutes < 1) return 'just now';
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.floor(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	return `${Math.floor(hours / 24)}d ago`;
}

export function formatFullDate(seconds: bigint | number): string {
	return new Date(Number(seconds) * 1000).toLocaleString();
}
