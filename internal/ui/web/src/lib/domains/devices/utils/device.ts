export type EditLabels = { key: string; value: string }[];

export function setDeviceLabels(
	originalLabels: Record<string, string>,
	editLabels: EditLabels
): Record<string, string> {
	const newLabels: Record<string, string> = {};

	for (const { key, value } of editLabels.filter((l) => l.key.trim())) {
		newLabels[key.trim()] = value;
	}
	for (const key of Object.keys(originalLabels)) {
		if (!(key in newLabels)) newLabels[key] = 'null';
	}

	return newLabels;
}
