import { api } from './api';
import type { Device, DeviceInput, ApiResponse, PaginatedResponse } from '$lib/types';

/**
 * Device service for managing IoT devices
 * Provides methods for CRUD operations on devices
 */
export const deviceService = {
	/**
	 * Get all devices with optional pagination
	 */
	async getAll(page = 1, perPage = 20): Promise<ApiResponse<PaginatedResponse<Device>>> {
		return api.get<PaginatedResponse<Device>>(`/devices?page=${page}&perPage=${perPage}`);
	},

	/**
	 * Get a single device by ID
	 */
	async getById(id: string): Promise<ApiResponse<Device>> {
		return api.get<Device>(`/devices/${id}`);
	},

	/**
	 * Create a new device
	 */
	async create(device: DeviceInput): Promise<ApiResponse<Device>> {
		return api.post<Device>('/devices', device);
	},

	/**
	 * Update an existing device
	 */
	async update(id: string, device: Partial<DeviceInput>): Promise<ApiResponse<Device>> {
		return api.patch<Device>(`/devices/${id}`, device);
	},

	/**
	 * Delete a device
	 */
	async delete(id: string): Promise<ApiResponse<void>> {
		return api.delete<void>(`/devices/${id}`);
	},

	/**
	 * Search devices by name or type
	 */
	async search(query: string): Promise<ApiResponse<Device[]>> {
		return api.get<Device[]>(`/devices/search?q=${encodeURIComponent(query)}`);
	}
};
