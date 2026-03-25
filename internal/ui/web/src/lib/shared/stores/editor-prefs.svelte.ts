const COOKIE_KEY = 'editor-prefs';
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1 year

export type GlobalTimeFilter =
	| { mode: 'relative'; minutes: number }
	| { mode: 'absolute'; start: Date; end: Date };

type CookieData = { vim: boolean; json: boolean; utc: boolean; refreshInterval: number; timeMinutes: number };

function readCookie(): CookieData {
	if (typeof document === 'undefined') return { vim: false, json: false, utc: false, refreshInterval: 10, timeMinutes: 60 };
	const match = document.cookie.match(/(?:^|;\s*)editor-prefs=([^;]*)/);
	if (!match) return { vim: false, json: false, utc: false, refreshInterval: 10, timeMinutes: 60 };
	try {
		return JSON.parse(decodeURIComponent(match[1]));
	} catch {
		return { vim: false, json: false, utc: false, refreshInterval: 10, timeMinutes: 60 };
	}
}

function writeCookie(value: CookieData) {
	if (typeof document === 'undefined') return;
	document.cookie = `${COOKIE_KEY}=${encodeURIComponent(JSON.stringify(value))};path=/;max-age=${COOKIE_MAX_AGE};SameSite=Lax`;
}

class EditorPrefsStore {
	vim = $state(false);
	json = $state(false);
	utc = $state(false);
	refreshInterval = $state(10);
	timeMinutes = $state(60);
	// Live time filter — supports both relative presets and absolute ranges.
	// Only the `timeMinutes` (relative) part is persisted to cookie/server.
	timeFilter = $state<GlobalTimeFilter>({ mode: 'relative', minutes: 60 });

	constructor() {
		const saved = readCookie();
		this.vim = saved.vim;
		this.json = saved.json;
		this.utc = saved.utc ?? false;
		this.refreshInterval = saved.refreshInterval ?? 10;
		this.timeMinutes = saved.timeMinutes ?? 60;
		this.timeFilter = { mode: 'relative', minutes: this.timeMinutes };
	}

	private _write() {
		writeCookie({ vim: this.vim, json: this.json, utc: this.utc, refreshInterval: this.refreshInterval, timeMinutes: this.timeMinutes });
	}

	setVim(value: boolean) {
		this.vim = value;
		this._write();
	}

	setJson(value: boolean) {
		this.json = value;
		this._write();
	}

	setUtc(value: boolean) {
		this.utc = value;
		this._write();
	}

	setRefreshInterval(value: number) {
		this.refreshInterval = value;
		this._write();
	}

	setTimeMinutes(value: number) {
		this.timeMinutes = value;
		this.timeFilter = { mode: 'relative', minutes: value };
		this._write();
	}

	setTimeFilter(filter: GlobalTimeFilter) {
		this.timeFilter = filter;
		if (filter.mode === 'relative') {
			this.timeMinutes = filter.minutes;
			this._write();
		}
	}
}

export const editorPrefs = new EditorPrefsStore();
