export type DeviceStatus = 'online' | 'offline' | 'maintenance';

export type Device = {
	id: string;
	name: string;
	type: 'sensor' | 'actuator' | 'gateway';
	status: DeviceStatus;
	description?: string;
	lastSeen?: Date;
	metadata?: Record<string, unknown>;
};

export type DeviceInput = Omit<Device, 'id' | 'lastSeen'>;
