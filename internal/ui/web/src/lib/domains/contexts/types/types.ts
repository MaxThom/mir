export type Context = {
	name: string;
	target: string; // NATS URL
	webTarget?: string; // explicit WebSocket URL; overrides the derived ws[s]://host:9222
	grafana: string; // Grafana URL
	secured: boolean; // true if context requires a password to retrieve credentials
};
