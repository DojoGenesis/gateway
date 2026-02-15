// Package apps provides MCP Apps host infrastructure for the Agentic Gateway.
//
// It enables visual interfaces for skills by managing:
//   - Resource serving for ui:// URIs
//   - App lifecycle management (launch, close, list)
//   - Tool call proxying with authorization
//   - Security enforcement (CSP, permissions, sandbox headers)
//
// This package provides the backend infrastructure only. Client-side rendering
// (iframe sandboxing, postMessage handling) is handled by the official
// @modelcontextprotocol/ext-apps SDK.
//
// Feature flag: MCP_APPS_ENABLED (default: false)
package apps
