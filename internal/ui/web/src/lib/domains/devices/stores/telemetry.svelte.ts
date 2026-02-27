import type { Mir } from '@mir/sdk';
import { DeviceTarget } from '@mir/sdk';
import type { TelemetryGroup, QueryData } from '@mir/sdk';

class TelemetryStore {
	measurements = $state<TelemetryGroup[]>([]);
	isLoading = $state(false);
	error = $state<string | null>(null);

	isQuerying = $state(false);
	queryError = $state<string | null>(null);
	queryData = $state<QueryData | null>(null);

	async loadMeasurements(mir: Mir, deviceId: string) {
		this.isLoading = true;
		this.error = null;

		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir.client().listTelemetry().request(target);
			this.measurements = result;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load telemetry';
		} finally {
			this.isLoading = false;
		}
	}

	async queryMeasurement(
		mir: Mir,
		deviceId: string,
		measurement: string,
		fields: string[],
		start: Date,
		end: Date
	) {
		this.isQuerying = true;
		this.queryError = null;

		try {
			const target = new DeviceTarget({ ids: [deviceId] });
			const result = await mir
				.client()
				.queryTelemetry()
				.request(target, measurement, fields, start, end);
			this.queryData = result;
		} catch (err) {
			this.queryError = err instanceof Error ? err.message : 'Failed to query telemetry';
			this.queryData = null;
		} finally {
			this.isQuerying = false;
		}
	}

	reset() {
		this.measurements = [];
		this.queryData = null;
		this.queryError = null;
		this.error = null;
	}
}

export const telemetryStore = new TelemetryStore();
