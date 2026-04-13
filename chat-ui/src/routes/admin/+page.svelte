<script lang="ts">
	import {
		adminListUsers,
		adminDeactivateUser,
		adminActivateUser,
		adminGetProviders,
		adminGetCosts,
		adminGetMCPStatus,
		adminGetHealth,
		type AdminUser,
		type ProviderStatus,
		type CostSummary,
		type MCPStatus,
		type HealthStatus
	} from '$lib/api';

	// ── Access control ────────────────────────────────────────────────────────
	function isAdmin(): boolean {
		try {
			const token = localStorage.getItem('access_token');
			if (!token) return false;
			const payload = JSON.parse(atob(token.split('.')[1]));
			return payload.role === 'admin';
		} catch {
			return false;
		}
	}

	// ── State ─────────────────────────────────────────────────────────────────
	let authorized = $state(false);
	let loadingUsers = $state(true);
	let loadingProviders = $state(true);
	let loadingCosts = $state(true);
	let loadingHealth = $state(true);
	let loadingMCP = $state(true);

	let users = $state<AdminUser[]>([]);
	let providers = $state<ProviderStatus[]>([]);
	let costs = $state<CostSummary | null>(null);
	let health = $state<HealthStatus | null>(null);
	let mcpServers = $state<MCPStatus[]>([]);

	let errorMsg = $state('');
	let togglingUser = $state<string | null>(null);

	// ── Provider auto-refresh ─────────────────────────────────────────────────
	let providerTimer: ReturnType<typeof setInterval> | null = null;

	// ── Data fetching ─────────────────────────────────────────────────────────
	async function fetchUsers() {
		loadingUsers = true;
		try {
			users = await adminListUsers();
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Failed to load users';
		} finally {
			loadingUsers = false;
		}
	}

	async function fetchProviders() {
		loadingProviders = true;
		try {
			providers = await adminGetProviders();
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Failed to load providers';
		} finally {
			loadingProviders = false;
		}
	}

	async function fetchCosts() {
		loadingCosts = true;
		try {
			costs = await adminGetCosts();
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Failed to load costs';
		} finally {
			loadingCosts = false;
		}
	}

	async function fetchHealth() {
		loadingHealth = true;
		try {
			health = await adminGetHealth();
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Failed to load health';
		} finally {
			loadingHealth = false;
		}
	}

	async function fetchMCP() {
		loadingMCP = true;
		try {
			mcpServers = await adminGetMCPStatus();
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Failed to load MCP status';
		} finally {
			loadingMCP = false;
		}
	}

	// ── User toggle ───────────────────────────────────────────────────────────
	async function toggleUser(user: AdminUser) {
		togglingUser = user.id;
		try {
			if (user.is_active) {
				await adminDeactivateUser(user.id);
			} else {
				await adminActivateUser(user.id);
			}
			await fetchUsers();
		} catch (e) {
			errorMsg = e instanceof Error ? e.message : 'Failed to update user';
		} finally {
			togglingUser = null;
		}
	}

	// ── Helpers ───────────────────────────────────────────────────────────────
	function providerStatusClass(status: string): string {
		if (status === 'healthy') return 'badge-green';
		if (status === 'degraded') return 'badge-yellow';
		return 'badge-red';
	}

	function healthDot(status: string): string {
		if (status === 'ok' || status === 'healthy') return 'dot-green';
		return 'dot-red';
	}

	function formatCost(n: number): string {
		return `$${n.toFixed(4)}`;
	}

	function formatTokens(n: number): string {
		if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
		if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
		return `${n}`;
	}

	function formatDate(s: string): string {
		try {
			return new Date(s).toLocaleDateString();
		} catch {
			return s;
		}
	}

	// ── Lifecycle ─────────────────────────────────────────────────────────────
	$effect(() => {
		authorized = isAdmin();
		if (!authorized) return;

		fetchUsers();
		fetchProviders();
		fetchCosts();
		fetchHealth();
		fetchMCP();

		providerTimer = setInterval(fetchProviders, 30_000);

		return () => {
			if (providerTimer !== null) clearInterval(providerTimer);
		};
	});

	// Dismiss error after 5s
	$effect(() => {
		if (!errorMsg) return;
		const t = setTimeout(() => { errorMsg = ''; }, 5000);
		return () => clearTimeout(t);
	});
</script>

{#if !authorized}
	<div class="access-denied">
		<div class="access-denied-card">
			<div class="access-icon">&#x26D4;</div>
			<h2>Access Denied</h2>
			<p>Admin role required to view this page.</p>
			<a href="/" class="btn-back">Back to Chat</a>
		</div>
	</div>
{:else}
	<div class="admin-shell">

		<!-- Header -->
		<header class="admin-header">
			<div class="admin-header-left">
				<a href="/" class="admin-logo">Dojo<span>Gateway</span></a>
				<span class="admin-badge">Admin Dashboard</span>
			</div>
			<div class="admin-header-right">
				{#if health}
					<span class="health-row">
						<span class="dot {healthDot(health.status)}"></span>
						<span class="health-label">{health.status}</span>
						<span class="health-meta">v{health.version} &middot; up {health.uptime}</span>
					</span>
				{:else if loadingHealth}
					<span class="health-loading">Loading health&hellip;</span>
				{/if}
				<a href="/" class="btn-text">Back to Chat</a>
			</div>
		</header>

		<!-- Main grid -->
		<main class="admin-main">

			<!-- Users panel -->
			<section class="panel panel-users">
				<div class="panel-header">
					<h2 class="panel-title">Users</h2>
					{#if !loadingUsers}
						<span class="panel-count">{users.length}</span>
					{/if}
				</div>
				{#if loadingUsers}
					<div class="panel-loading">Loading users&hellip;</div>
				{:else if users.length === 0}
					<div class="panel-empty">No users found.</div>
				{:else}
					<div class="table-wrap">
						<table class="data-table">
							<thead>
								<tr>
									<th>Email</th>
									<th>Display Name</th>
									<th>Status</th>
									<th>Created</th>
									<th>Action</th>
								</tr>
							</thead>
							<tbody>
								{#each users as user, i}
									<tr class={i % 2 === 0 ? 'row-even' : 'row-odd'}>
										<td class="td-email">{user.email}</td>
										<td>{user.display_name || '—'}</td>
										<td>
											<span class="status-pill {user.is_active ? 'pill-green' : 'pill-muted'}">
												{user.is_active ? 'Active' : 'Inactive'}
											</span>
										</td>
										<td class="td-meta">{formatDate(user.created_at)}</td>
										<td>
											<button
												class="btn-toggle {user.is_active ? 'btn-deactivate' : 'btn-activate'}"
												onclick={() => toggleUser(user)}
												disabled={togglingUser === user.id}
											>
												{togglingUser === user.id
													? '...'
													: user.is_active
														? 'Deactivate'
														: 'Activate'}
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- Providers panel -->
			<section class="panel panel-providers">
				<div class="panel-header">
					<h2 class="panel-title">Providers</h2>
					<span class="panel-refresh-note">auto-refresh 30s</span>
				</div>
				{#if loadingProviders}
					<div class="panel-loading">Loading providers&hellip;</div>
				{:else if providers.length === 0}
					<div class="panel-empty">No providers found.</div>
				{:else}
					<div class="table-wrap">
						<table class="data-table">
							<thead>
								<tr>
									<th>Provider</th>
									<th>Status</th>
									<th>Latency</th>
									<th>Last Checked</th>
								</tr>
							</thead>
							<tbody>
								{#each providers as p, i}
									<tr class={i % 2 === 0 ? 'row-even' : 'row-odd'}>
										<td class="td-name">{p.name}</td>
										<td>
											<span class="badge {providerStatusClass(p.status)}">{p.status}</span>
										</td>
										<td class="td-meta">{p.latency_ms}ms</td>
										<td class="td-meta">{formatDate(p.last_checked)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- MCP panel -->
			<section class="panel panel-mcp">
				<div class="panel-header">
					<h2 class="panel-title">MCP Servers</h2>
				</div>
				{#if loadingMCP}
					<div class="panel-loading">Loading MCP status&hellip;</div>
				{:else if mcpServers.length === 0}
					<div class="panel-empty">No MCP servers found.</div>
				{:else}
					<div class="table-wrap">
						<table class="data-table">
							<thead>
								<tr>
									<th>Server</th>
									<th>Status</th>
									<th>Tools</th>
								</tr>
							</thead>
							<tbody>
								{#each mcpServers as s, i}
									<tr class={i % 2 === 0 ? 'row-even' : 'row-odd'}>
										<td class="td-name">{s.name}</td>
										<td>
											<span class="badge {providerStatusClass(s.status)}">{s.status}</span>
										</td>
										<td class="td-meta">{s.tools_count}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>

			<!-- Costs panel -->
			<section class="panel panel-costs">
				<div class="panel-header">
					<h2 class="panel-title">Cost Summary</h2>
				</div>
				{#if loadingCosts}
					<div class="panel-loading">Loading costs&hellip;</div>
				{:else if !costs}
					<div class="panel-empty">No cost data available.</div>
				{:else}
					<div class="costs-layout">
						<!-- Totals -->
						<div class="cost-totals">
							<div class="cost-stat">
								<span class="cost-stat-label">Total Tokens</span>
								<span class="cost-stat-value">{formatTokens(costs.total_tokens)}</span>
							</div>
							<div class="cost-stat">
								<span class="cost-stat-label">Estimated Cost</span>
								<span class="cost-stat-value accent">{formatCost(costs.total_cost)}</span>
							</div>
						</div>

						<div class="costs-breakdown">
							<!-- By provider -->
							<div class="breakdown-block">
								<h3 class="breakdown-title">By Provider</h3>
								{#if Object.keys(costs.by_provider).length === 0}
									<p class="panel-empty">No data.</p>
								{:else}
									<table class="data-table breakdown-table">
										<thead>
											<tr><th>Provider</th><th>Cost</th></tr>
										</thead>
										<tbody>
											{#each Object.entries(costs.by_provider) as [name, val], i}
												<tr class={i % 2 === 0 ? 'row-even' : 'row-odd'}>
													<td>{name}</td>
													<td class="td-meta">{formatCost(val)}</td>
												</tr>
											{/each}
										</tbody>
									</table>
								{/if}
							</div>

							<!-- By user -->
							<div class="breakdown-block">
								<h3 class="breakdown-title">By User</h3>
								{#if Object.keys(costs.by_user).length === 0}
									<p class="panel-empty">No data.</p>
								{:else}
									<table class="data-table breakdown-table">
										<thead>
											<tr><th>User</th><th>Cost</th></tr>
										</thead>
										<tbody>
											{#each Object.entries(costs.by_user) as [name, val], i}
												<tr class={i % 2 === 0 ? 'row-even' : 'row-odd'}>
													<td class="td-email">{name}</td>
													<td class="td-meta">{formatCost(val)}</td>
												</tr>
											{/each}
										</tbody>
									</table>
								{/if}
							</div>
						</div>
					</div>
				{/if}
			</section>

		</main>
	</div>

	{#if errorMsg}
		<div class="error-toast">{errorMsg}</div>
	{/if}
{/if}

<style>
	/* ── Shell ─────────────────────────────────────────────────────────────── */
	.admin-shell {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		background: var(--color-bg);
		color: var(--color-text);
		font-family: var(--font-sans);
		overflow-y: auto;
	}

	/* ── Header ────────────────────────────────────────────────────────────── */
	.admin-header {
		height: var(--header-height);
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 20px;
		border-bottom: 1px solid var(--color-border);
		background: var(--color-bg);
		position: sticky;
		top: 0;
		z-index: 10;
	}

	.admin-header-left {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.admin-logo {
		font-size: 16px;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: -0.02em;
		text-decoration: none;
	}
	.admin-logo span {
		color: var(--color-accent);
	}

	.admin-badge {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--color-accent);
		background: var(--color-accent-muted);
		padding: 2px 8px;
		border-radius: var(--radius-sm);
	}

	.admin-header-right {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.health-row {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.dot-green { background: var(--color-success); }
	.dot-red   { background: var(--color-error); }

	.health-label {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text);
	}

	.health-meta {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.health-loading {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.btn-text {
		font-size: 13px;
		color: var(--color-text-secondary);
		padding: 5px 10px;
		border-radius: var(--radius-sm);
		transition: background var(--transition), color var(--transition);
		text-decoration: none;
		cursor: pointer;
	}
	.btn-text:hover {
		background: var(--color-bg-hover);
		color: var(--color-text);
		text-decoration: none;
	}

	/* ── Main grid ─────────────────────────────────────────────────────────── */
	.admin-main {
		flex: 1;
		display: grid;
		grid-template-columns: 1fr 1fr;
		grid-template-rows: auto auto;
		gap: 16px;
		padding: 20px;
		align-items: start;
	}

	.panel-users    { grid-column: 1; grid-row: 1; }
	.panel-providers { grid-column: 2; grid-row: 1; }
	.panel-mcp      { grid-column: 2; grid-row: 2; }
	.panel-costs    { grid-column: 1; grid-row: 2; }

	/* ── Panel ─────────────────────────────────────────────────────────────── */
	.panel {
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		overflow: hidden;
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 14px 16px 12px;
		border-bottom: 1px solid var(--color-border);
	}

	.panel-title {
		font-size: 13px;
		font-weight: 600;
		color: var(--color-text-secondary);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		margin: 0;
	}

	.panel-count {
		font-size: 12px;
		font-weight: 600;
		color: var(--color-accent);
		background: var(--color-accent-muted);
		padding: 1px 7px;
		border-radius: 10px;
	}

	.panel-refresh-note {
		font-size: 11px;
		color: var(--color-text-muted);
	}

	.panel-loading,
	.panel-empty {
		padding: 24px 16px;
		font-size: 13px;
		color: var(--color-text-muted);
		text-align: center;
	}

	/* ── Table ─────────────────────────────────────────────────────────────── */
	.table-wrap {
		overflow-x: auto;
	}

	.data-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 13px;
	}

	.data-table th {
		padding: 8px 12px;
		text-align: left;
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--color-text-muted);
		border-bottom: 1px solid var(--color-border);
		white-space: nowrap;
	}

	.data-table td {
		padding: 9px 12px;
		color: var(--color-text);
		vertical-align: middle;
	}

	.row-even { background: transparent; }
	.row-odd  { background: rgba(255,255,255,0.02); }

	.data-table tr:hover td {
		background: var(--color-bg-hover);
	}

	.td-email {
		font-size: 12px;
		color: var(--color-text-secondary);
		max-width: 180px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.td-meta {
		font-size: 12px;
		color: var(--color-text-muted);
		white-space: nowrap;
	}

	.td-name {
		font-weight: 500;
	}

	/* ── Badges ────────────────────────────────────────────────────────────── */
	.badge {
		display: inline-block;
		padding: 2px 8px;
		border-radius: 10px;
		font-size: 11px;
		font-weight: 600;
		text-transform: capitalize;
		letter-spacing: 0.03em;
	}

	.badge-green  { background: rgba(59,165,92,0.2);  color: var(--color-success); }
	.badge-yellow { background: rgba(250,166,26,0.2); color: var(--color-warning); }
	.badge-red    { background: rgba(237,66,69,0.2);  color: var(--color-error); }

	.status-pill {
		display: inline-block;
		padding: 2px 8px;
		border-radius: 10px;
		font-size: 11px;
		font-weight: 600;
	}

	.pill-green { background: rgba(59,165,92,0.18); color: var(--color-success); }
	.pill-muted { background: var(--color-bg-tertiary); color: var(--color-text-muted); }

	/* ── Toggle buttons ────────────────────────────────────────────────────── */
	.btn-toggle {
		font-size: 12px;
		font-weight: 500;
		padding: 4px 10px;
		border-radius: var(--radius-sm);
		cursor: pointer;
		transition: opacity var(--transition), background var(--transition);
		border: 1px solid transparent;
	}
	.btn-toggle:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-deactivate {
		background: rgba(237,66,69,0.12);
		color: var(--color-error);
		border-color: rgba(237,66,69,0.25);
	}
	.btn-deactivate:hover:not(:disabled) {
		background: rgba(237,66,69,0.22);
	}

	.btn-activate {
		background: rgba(59,165,92,0.12);
		color: var(--color-success);
		border-color: rgba(59,165,92,0.25);
	}
	.btn-activate:hover:not(:disabled) {
		background: rgba(59,165,92,0.22);
	}

	/* ── Costs ─────────────────────────────────────────────────────────────── */
	.costs-layout {
		padding: 16px;
		display: flex;
		flex-direction: column;
		gap: 16px;
	}

	.cost-totals {
		display: flex;
		gap: 16px;
	}

	.cost-stat {
		flex: 1;
		background: var(--color-bg-tertiary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: 14px 16px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.cost-stat-label {
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--color-text-muted);
	}

	.cost-stat-value {
		font-size: 22px;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: -0.02em;
	}

	.cost-stat-value.accent {
		color: var(--color-accent);
	}

	.costs-breakdown {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 16px;
	}

	.breakdown-block {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.breakdown-title {
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
		color: var(--color-text-muted);
		margin: 0;
	}

	.breakdown-table {
		background: var(--color-bg-tertiary);
		border-radius: var(--radius-md);
		overflow: hidden;
	}

	/* ── Access denied ─────────────────────────────────────────────────────── */
	.access-denied {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--color-bg);
	}

	.access-denied-card {
		text-align: center;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		padding: 48px 40px;
		box-shadow: var(--shadow-md);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
	}

	.access-icon {
		font-size: 40px;
		line-height: 1;
	}

	.access-denied-card h2 {
		font-size: 20px;
		font-weight: 700;
		color: var(--color-text);
		margin: 0;
	}

	.access-denied-card p {
		font-size: 14px;
		color: var(--color-text-secondary);
		margin: 0;
	}

	.btn-back {
		margin-top: 8px;
		display: inline-block;
		padding: 9px 20px;
		background: var(--color-accent);
		color: #fff;
		font-size: 14px;
		font-weight: 600;
		border-radius: var(--radius-md);
		text-decoration: none;
		transition: background var(--transition);
	}
	.btn-back:hover {
		background: var(--color-accent-hover);
		text-decoration: none;
	}

	/* ── Error toast ───────────────────────────────────────────────────────── */
	.error-toast {
		position: fixed;
		bottom: 24px;
		left: 50%;
		transform: translateX(-50%);
		background: var(--color-error);
		color: #fff;
		padding: 10px 20px;
		border-radius: var(--radius-md);
		font-size: 13px;
		z-index: 100;
		box-shadow: var(--shadow-md);
		animation: slide-up 0.2s ease;
		white-space: nowrap;
	}

	@keyframes slide-up {
		from { opacity: 0; transform: translateX(-50%) translateY(8px); }
		to   { opacity: 1; transform: translateX(-50%) translateY(0); }
	}

	/* ── Responsive ────────────────────────────────────────────────────────── */
	@media (max-width: 900px) {
		.admin-main {
			grid-template-columns: 1fr;
		}
		.panel-users,
		.panel-providers,
		.panel-mcp,
		.panel-costs {
			grid-column: 1;
			grid-row: auto;
		}
		.costs-breakdown {
			grid-template-columns: 1fr;
		}
	}
</style>
