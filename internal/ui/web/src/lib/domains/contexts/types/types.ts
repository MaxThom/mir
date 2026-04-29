export type Context = {
	name: string;
	target: string; // WebSocket URL (ws:// or wss://), resolved by the backend
	grafana: string; // Grafana URL
	secured: boolean; // true if context requires a password to retrieve credentials
};
