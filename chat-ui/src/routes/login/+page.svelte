<script lang="ts">
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { login } from '$lib/api';
	import { saveAuth } from '$lib/stores.svelte';

	let email = $state('');
	let password = $state('');
	let errorMsg = $state('');
	let loading = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (!email.trim() || !password) return;

		errorMsg = '';
		loading = true;

		try {
			const resp = await login(email.trim(), password);
			saveAuth(resp.access_token, resp.refresh_token, resp.user_id, resp.display_name);
			goto(`${base}/`);
		} catch (err) {
			errorMsg = err instanceof Error ? err.message : 'Login failed';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Sign in — Dojo Chat</title>
</svelte:head>

<div class="login-page">
	<div class="login-card">
		<div class="login-logo">
			<h1>Dojo <span>Chat</span></h1>
			<p>Powered by the Dojo Gateway</p>
		</div>

		{#if errorMsg}
			<div class="login-error" role="alert">{errorMsg}</div>
		{/if}

		<form onsubmit={handleSubmit}>
			<div class="form-group">
				<label class="form-label" for="email">Email</label>
				<input
					id="email"
					type="email"
					class="form-input"
					placeholder="you@example.com"
					bind:value={email}
					disabled={loading}
					required
					autocomplete="email"
				/>
			</div>

			<div class="form-group">
				<label class="form-label" for="password">Password</label>
				<input
					id="password"
					type="password"
					class="form-input"
					placeholder="••••••••"
					bind:value={password}
					disabled={loading}
					required
					autocomplete="current-password"
				/>
			</div>

			<button type="submit" class="btn-primary" disabled={loading || !email.trim() || !password}>
				{loading ? 'Signing in…' : 'Sign in'}
			</button>
		</form>

		<div class="divider">or</div>

		<a href="{base}/auth/github" class="btn-github">
			<svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
				<path d="M12 0C5.374 0 0 5.373 0 12c0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23A11.509 11.509 0 0112 5.803c1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576C20.566 21.797 24 17.3 24 12c0-6.627-5.373-12-12-12z" />
			</svg>
			Continue with GitHub
		</a>
	</div>
</div>
