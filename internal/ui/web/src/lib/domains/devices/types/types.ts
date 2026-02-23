export type Context = {
	name: string;
	target: string; // NATS URL
	grafana: string; // Grafana URL
};

export type Descriptor = {
	name: string;
	labels: Record<string, string>;
	template: string;
	error: string;
};

export type ResponseEntry = {
	status: number;
	error: string;
	payload: Uint8Array;
};
