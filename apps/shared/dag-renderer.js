/**
 * dag-renderer.js — Zero-dependency DAG rendering for Dojo MCP Apps.
 *
 * Shared by: observability-dashboard (read-only), workflow-builder (editable).
 * Design: renders DAG nodes with state-aware styling into a container element.
 *
 * Usage:
 *   DAGRenderer.render(container, nodes, options);
 *   DAGRenderer.updateNode(container, nodeId, 'success', 245);
 *   DAGRenderer.clear(container);
 */
const DAGRenderer = (() => {
  const STATES = {
    PENDING: 'pending',
    RUNNING: 'running',
    SUCCESS: 'success',
    FAILED: 'failed',
    REPLANNING: 'replanning',
  };

  const STATE_COLORS = {
    pending: 'var(--dag-pending, #64748b)',
    running: 'var(--dag-running, #3b82f6)',
    success: 'var(--dag-success, #22c55e)',
    failed: 'var(--dag-failed, #ef4444)',
    replanning: 'var(--dag-replanning, #eab308)',
  };

  const STATE_BORDERS = {
    pending: 'var(--dag-border, #1e293b)',
    running: 'var(--dag-running, #3b82f6)',
    success: 'var(--dag-success, #22c55e)',
    failed: 'var(--dag-failed, #ef4444)',
    replanning: 'var(--dag-replanning, #eab308)',
  };

  /**
   * Render a DAG into a container.
   * @param {HTMLElement} container - Target container element
   * @param {Object} nodes - Map of nodeId -> {toolName, state, durationMs, dependencies, planId}
   * @param {Object} options - {animate: bool, showDuration: bool, onClick: fn(nodeId)}
   */
  function render(container, nodes, options = {}) {
    const { animate = true, showDuration = true, onClick = null } = options;

    container.innerHTML = '';
    container.style.display = 'flex';
    container.style.flexWrap = 'wrap';
    container.style.gap = '8px';
    container.style.alignContent = 'flex-start';

    Object.entries(nodes).forEach(([id, node]) => {
      const el = _createNodeElement(id, node, { animate, showDuration, onClick });
      el.dataset.nodeId = id;
      container.appendChild(el);
    });
  }

  /**
   * Update a single node's state without re-rendering the entire DAG.
   * @param {HTMLElement} container - Target container element
   * @param {string} nodeId - Node ID to update
   * @param {string} state - New state (from STATES)
   * @param {number} durationMs - Optional duration in ms
   */
  function updateNode(container, nodeId, state, durationMs) {
    const el = container.querySelector(`[data-node-id="${nodeId}"]`);
    if (!el) return;

    // Update state class
    Object.values(STATES).forEach(s => el.classList.remove(s));
    el.classList.add(state);

    // Update dot color
    const dot = el.querySelector('.dag-dot');
    if (dot) dot.style.background = STATE_COLORS[state] || STATE_COLORS.pending;

    // Update border
    el.style.borderColor = STATE_BORDERS[state] || STATE_BORDERS.pending;

    // Update duration
    if (durationMs !== undefined) {
      let dur = el.querySelector('.dag-duration');
      if (!dur) {
        dur = document.createElement('span');
        dur.className = 'dag-duration';
        dur.style.cssText = 'color:var(--dag-dim, #64748b);font-size:11px;margin-left:4px;';
        el.appendChild(dur);
      }
      dur.textContent = `${durationMs}ms`;
    }

    // Remove animation for completed states
    if (state === STATES.SUCCESS || state === STATES.FAILED) {
      el.style.animation = 'none';
    }

    // Add pulse for running state
    if (state === STATES.RUNNING) {
      el.style.animation = 'dag-pulse 1.5s infinite';
      el.style.boxShadow = `0 0 8px ${STATE_COLORS.running}40`;
    }
  }

  /**
   * Clear the DAG container.
   * @param {HTMLElement} container
   */
  function clear(container) {
    container.innerHTML = '';
  }

  // --- Internal helpers ---

  function _createNodeElement(id, node, options) {
    const el = document.createElement('div');
    const state = node.state || STATES.PENDING;
    el.className = `dag-node ${state}`;
    el.style.cssText = `
      display: inline-flex;
      align-items: center;
      gap: 6px;
      background: var(--dag-surface, #1e293b);
      border: 1px solid ${STATE_BORDERS[state] || STATE_BORDERS.pending};
      border-radius: 8px;
      padding: 8px 14px;
      font-size: 12px;
      font-family: var(--dag-mono, monospace);
      color: var(--dag-fg, #e2e8f0);
      cursor: ${options.onClick ? 'pointer' : 'default'};
      transition: border-color 0.2s, box-shadow 0.2s;
      ${state === STATES.PENDING ? 'opacity: 0.5;' : ''}
      ${state === STATES.RUNNING ? `animation: dag-pulse 1.5s infinite; box-shadow: 0 0 8px ${STATE_COLORS.running}40;` : ''}
    `;

    // Dot indicator
    const dot = document.createElement('span');
    dot.className = 'dag-dot';
    dot.style.cssText = `
      width: 8px; height: 8px; border-radius: 50%;
      background: ${STATE_COLORS[state] || STATE_COLORS.pending};
      flex-shrink: 0;
    `;
    el.appendChild(dot);

    // Tool name
    const name = document.createElement('span');
    name.textContent = node.toolName || id;
    el.appendChild(name);

    // Duration
    if (options.showDuration && node.durationMs) {
      const dur = document.createElement('span');
      dur.className = 'dag-duration';
      dur.style.cssText = 'color:var(--dag-dim, #64748b);font-size:11px;margin-left:4px;';
      dur.textContent = `${node.durationMs}ms`;
      el.appendChild(dur);
    }

    // Tooltip
    el.title = `Node: ${id}\nTool: ${node.toolName || '-'}\nState: ${state}\nPlan: ${node.planId || '-'}`;

    // Click handler
    if (options.onClick) {
      el.addEventListener('click', () => options.onClick(id));
    }

    return el;
  }

  // Inject keyframes if not already present
  if (typeof document !== 'undefined' && !document.getElementById('dag-renderer-styles')) {
    const style = document.createElement('style');
    style.id = 'dag-renderer-styles';
    style.textContent = '@keyframes dag-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.7; } }';
    document.head.appendChild(style);
  }

  return { render, updateNode, clear, STATES, STATE_COLORS };
})();

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
  module.exports = DAGRenderer;
}
