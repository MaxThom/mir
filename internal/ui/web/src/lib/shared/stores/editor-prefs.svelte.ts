const COOKIE_KEY = 'editor-prefs';
const COOKIE_MAX_AGE = 60 * 60 * 24 * 365; // 1 year

function readCookie(): { vim: boolean; json: boolean } {
	if (typeof document === 'undefined') return { vim: false, json: false };
	const match = document.cookie.match(/(?:^|;\s*)editor-prefs=([^;]*)/);
	if (!match) return { vim: false, json: false };
	try {
		return JSON.parse(decodeURIComponent(match[1]));
	} catch {
		return { vim: false, json: false };
	}
}

function writeCookie(value: { vim: boolean; json: boolean }) {
	if (typeof document === 'undefined') return;
	document.cookie = `${COOKIE_KEY}=${encodeURIComponent(JSON.stringify(value))};path=/;max-age=${COOKIE_MAX_AGE};SameSite=Lax`;
}

class EditorPrefsStore {
	vim = $state(false);
	json = $state(false);

	constructor() {
		const saved = readCookie();
		this.vim = saved.vim;
		this.json = saved.json;
	}

	setVim(value: boolean) {
		this.vim = value;
		writeCookie({ vim: this.vim, json: this.json });
	}

	setJson(value: boolean) {
		this.json = value;
		writeCookie({ vim: this.vim, json: this.json });
	}
}

export const editorPrefs = new EditorPrefsStore();
