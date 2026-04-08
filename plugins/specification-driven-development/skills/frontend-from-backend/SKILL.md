---
name: frontend-from-backend
description: Write production-ready frontend specifications deeply grounded in existing backend architecture. Prevents integration issues by starting with backend reality, not frontend imagination.
triggers:
  - "write a frontend spec grounded in the backend"
  - "derive the frontend from existing backend APIs"
  - "spec the UI based on what the backend actually provides"
metadata:
  version: "1.0"
  tool_dependencies:
    - file_system
    - bash
  portable: true
  tier: 1
  agents:
    - implementation-agent
    - strategic-agent
---

# Frontend From Backend Skill

**Version:** 1.0
**Author:** Tres Pies Design
**Purpose:** Write high-quality frontend specifications that are deeply integrated with an existing backend, ensuring seamless development and preventing integration friction.

---

## I. The Philosophy: Grounding Before Building

Frontend development in a full-stack application does not happen in a vacuum. The most common source of bugs, delays, and rework is a disconnect between the frontend implementation and the backend reality. This skill is built on a simple but powerful principle: **grounding before building**.

By deeply understanding the existing backend architecture, APIs, and data models *before* writing a single line of frontend specification, we prevent entire classes of integration problems. What we design is not just beautiful, but buildable.

---

## II. When to Use This Skill

- When planning a new frontend feature that interacts with an existing backend
- When writing specifications for a UI redesign with a backend component
- When commissioning frontend work to an autonomous implementation agent
- When you feel a disconnect between the frontend vision and the backend reality
- At the beginning of any major frontend development cycle

---

## III. The 5-Step Workflow

### Step 1: Deep Backend Analysis

**Goal:** Achieve comprehensive understanding of the existing backend architecture.

1. **Read key backend files:** Entry point, route registration, handlers, middleware
2. **Document APIs:** Map all relevant endpoints with methods, auth requirements, request bodies, success/error responses
3. **Identify data models:** Understand the shapes of data the backend returns
4. **Map integration points:** For each frontend feature area, identify the specific backend endpoint it will consume

**Output:** A Backend Integration Map (table format).

**Backend Integration Map Template:**

| Feature Area | Endpoint | Method | Auth | Request Body | Success Response | Error Response |
|-------------|----------|--------|------|-------------|-----------------|----------------|
| User List | `/api/v1/users` | GET | Bearer | - | `{ users: User[] }` | `{ error: string }` |
| Create User | `/api/v1/users` | POST | Bearer | `{ name: string }` | `{ id: string }` | `{ error: string, code: number }` |

### Step 2: Comprehensive Feature Specification

**Goal:** Write a production-ready specification grounded in backend reality.

1. **Executive summary** and problem statement
2. **Goals, non-goals, and user stories**
3. **Technical architecture** — How frontend and backend will interact
4. **UI/UX interaction flows** — With specific API calls mapped to each interaction
5. **API contracts** — Request/response examples for every call
6. **Security considerations** — Auth flow, token handling, CORS

### Step 3: Component Architecture

**Goal:** Design the component tree with state shapes derived from backend data models.

For each component:
- **Purpose** — What it renders and why
- **Props interface** — TypeScript interface
- **Internal state** — TypeScript interface, derived from backend response types
- **API calls** — Which endpoints it consumes
- **Loading state** — What renders during fetch
- **Error state** — What renders on failure
- **Empty state** — What renders when data is empty

**State Shape Derivation:**

```typescript
// Backend returns:
interface ApiUserResponse {
  id: string;
  name: string;
  email: string;
  created_at: string;
}

// Frontend state derived from backend:
interface UserState {
  users: ApiUserResponse[];
  loading: boolean;
  error: string | null;
}
```

### Step 4: Integration Guide

**Goal:** Create a practical guide for wiring frontend to backend.

1. **Authentication flow** — How the frontend handles auth (token storage, refresh, logout)
2. **API client setup** — Base URL, headers, interceptors
3. **Streaming architecture** — SSE/WebSocket connections if applicable
4. **Error handling patterns** — How different HTTP status codes map to UI states
5. **Code examples** — Actual fetch/axios calls with error handling

### Step 5: Audit and Deliver

**Goal:** Verify completeness and save.

1. **Verify every frontend API call** references a real backend endpoint
2. **Flag Backend Prerequisites** — If the frontend needs an endpoint that doesn't exist, mark it explicitly
3. **Check state shapes** match backend response types
4. **Save** as `[version]_frontend_spec_[feature].md`

---

## IV. The "Backend Prerequisite" Pattern

When the frontend needs functionality the backend doesn't yet provide:

```markdown
### Backend Prerequisite: [Feature Name]

**Needed by:** [Component/Feature that needs it]
**Proposed endpoint:** `[METHOD] /api/v1/[resource]`
**Request:** `{ field: type }`
**Response:** `{ field: type }`
**Reason:** [Why the frontend needs this and why it doesn't exist yet]
**Priority:** [Blocking | Nice-to-have]
```

This makes the gap explicit. Never silently invent backend endpoints in a frontend spec.

---

## V. Best Practices

- **The backend is the source of truth.** If there is a discrepancy between the frontend design and the backend API, the frontend design must adapt.
- **No backend changes is the ideal.** Design the frontend to work with the existing backend. Only propose backend changes as a last resort.
- **Over-document the integration.** Too much detail on frontend-backend wiring is always better than too little.
- **Derive state from data models.** Frontend state shapes should be direct derivations of backend response types, not independent inventions.
- **Complete this process before writing code.** This is a pre-development activity.

---

## VI. Quality Checklist

- [ ] Have you read the backend entry point and relevant handlers?
- [ ] Have you documented every API endpoint the frontend will consume?
- [ ] Does every frontend component document its loading, error, and empty states?
- [ ] Are all TypeScript interfaces derived from actual backend response types?
- [ ] Are Backend Prerequisites explicitly flagged (not silently invented)?
- [ ] Does the spec include an authentication flow description?
- [ ] Does the spec include error handling patterns for all HTTP status codes?
