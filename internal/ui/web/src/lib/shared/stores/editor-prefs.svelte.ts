const COOKIE_KEY = 'editor-prefs';
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1 year

function readCookie(): { vim: boolean; json: boolean; utc: boolean; refreshInterval: number } {
	if (typeof document === 'undefined') return { vim: false, json: false, utc: false, refreshInterval: 10 };
	const match = document.cookie.match(/(?:^|;\s*)editor-prefs=([^;]*)/);
	if (!match) return { vim: false, json: false, utc: false, refreshInterval: 10 };
	try {
		return JSON.parse(decodeURIComponent(match[1]));
	} catch {
		return { vim: false, json: false, utc: false, refreshInterval: 10 };
	}
}

function writeCookie(value: { vim: boolean; json: boolean; utc: boolean; refreshInterval: number }) {
	if (typeof document === 'undefined') return;
	document.cookie = `${COOKIE_KEY}=${encodeURIComponent(JSON.stringify(value))};path=/;max-age=${COOKIE_MAX_AGE};SameSite=Lax`;
}

class EditorPrefsStore {
	vim = $state(false);
	json = $state(false);
	utc = $state(false);
	refreshInterval = $state(10);

	constructor() {
		const saved = readCookie();
		this.vim = saved.vim;
		this.json = saved.json;
		this.utc = saved.utc ?? false;
		this.refreshInterval = saved.refreshInterval ?? 10;
	}

	setVim(value: boolean) {
		this.vim = value;
		writeCookie({ vim: this.vim, json: this.json, utc: this.utc, refreshInterval: this.refreshInterval });
	}

	setJson(value: boolean) {
		this.json = value;
		writeCookie({ vim: this.vim, json: this.json, utc: this.utc, refreshInterval: this.refreshInterval });
	}

	setUtc(value: boolean) {
		this.utc = value;
		writeCookie({ vim: this.vim, json: this.json, utc: this.utc, refreshInterval: this.refreshInterval });
	}

	setRefreshInterval(value: number) {
		this.refreshInterval = value;
		writeCookie({ vim: this.vim, json: this.json, utc: this.utc, refreshInterval: this.refreshInterval });
	}
}

export const editorPrefs = new EditorPrefsStore();
