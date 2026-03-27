import devtoolsJson from 'vite-plugin-devtools-json';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';
import { sveltekit } from '@sveltejs/kit/vite';
import { resolve } from 'path';

const suppressUnusedExternalImport = {
	onwarn(warning: { code?: string }, warn: (w: { code?: string }) => void) {
		// False positives: imports used inside $effect are stripped in SSR compilation
		if (warning.code === 'UNUSED_EXTERNAL_IMPORT') return;
		warn(warning);
	}
};

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), devtoolsJson()],

	environments: {
		ssr: {
			build: {
				rollupOptions: suppressUnusedExternalImport
			}
		}
	},

	build: {
		chunkSizeWarningLimit: 1000,
		rollupOptions: suppressUnusedExternalImport
	},

	resolve: {
		alias: {
			'@mir/sdk': resolve(__dirname, '../../../pkgs/web/src/index.ts')
		}
	},

	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:3021',
				changeOrigin: true
			}
		}
	},

	test: {
		expect: { requireAssertions: true },
		projects: [
			{
				extends: './vite.config.ts',
				test: {
					name: 'client',
					browser: {
						enabled: true,
						provider: playwright(),
						instances: [{ browser: 'chromium', headless: true }]
					},
					include: ['src/**/*.svelte.{test,spec}.{js,ts}'],
					exclude: ['src/lib/server/**']
				}
			},

			{
				extends: './vite.config.ts',
				test: {
					name: 'server',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
