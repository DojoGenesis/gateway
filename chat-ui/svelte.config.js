import adapter from '@sveltejs/adapter-static';
import { relative, sep } from 'node:path';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	compilerOptions: {
		// Enforce rune mode for all project files (not node_modules).
		runes: ({ filename }) => {
			const relativePath = relative(import.meta.dirname, filename);
			const pathSegments = relativePath.toLowerCase().split(sep);
			const isExternalLibrary = pathSegments.includes('node_modules');
			return isExternalLibrary ? undefined : true;
		}
	},
	kit: {
		// adapter-static builds a standalone SPA served by the Go Gateway at GET /chat/*.
		// fallback: 'index.html' enables client-side routing for all unmatched paths.
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false,
			strict: false
		}),
		// Base path must match the route prefix registered in server/router.go.
		paths: {
			base: '/chat'
		}
	}
};

export default config;
