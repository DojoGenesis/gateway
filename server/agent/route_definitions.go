package agent

// DefaultRouteDefinitions returns the built-in set of handler-oriented route
// definitions for the SemanticRouter. Each route maps to a specific processing
// commitment (handler), not a semantic label.
//
// Route categories are designed for embedding separability:
//   - DIRECT_RESPONSE: identity/capability language ("who", "hello", "what can you do")
//   - FAST_INFERENCE: lookup/definition patterns ("what is", "how many", "convert")
//   - DEEP_INFERENCE: imperative construction + technical tokens ("write", "debug", "refactor")
//   - SPECIALIST_DISPATCH: domain-data + retrieval imperative ("pull", "search", "look up")
//   - ORCHESTRATED_PLAN: compound conjunctions + multi-object sequences ("then", "and create")
func DefaultRouteDefinitions() []RouteDefinition {
	return []RouteDefinition{
		{
			Name:          "direct_response",
			Handler:       "template",
			ProviderAlias: "",
			Fallback:      "llm-fast",
			Threshold:     0.70,
			Utterances: []string{
				"Hi there",
				"Hello, can you help me?",
				"What can you do?",
				"Good morning",
				"Who are you?",
				"Tell me your name",
				"What version are you?",
				"Thanks, bye",
				"What are your capabilities?",
				"How do you work?",
			},
		},
		{
			Name:          "fast_inference",
			Handler:       "llm-fast",
			ProviderAlias: "llm-fast",
			Fallback:      "llm-reasoning",
			Threshold:     0.60,
			Utterances: []string{
				"What is the capital of France?",
				"Convert 42 miles to kilometers",
				"What does idempotent mean?",
				"What year was Python released?",
				"How many bytes in a gigabyte?",
				"Explain what an API is in one sentence",
				"What's the difference between TCP and UDP?",
				"Who invented the World Wide Web?",
				"What time zone is Tokyo in?",
				"What is a p-value?",
			},
		},
		{
			Name:          "deep_inference",
			Handler:       "llm-reasoning",
			ProviderAlias: "llm-reasoning",
			Fallback:      "llm-fast",
			Threshold:     0.58,
			Utterances: []string{
				"Write a Go HTTP handler that validates JWT tokens and returns 401 on failure",
				"Why is my recursive function hitting a stack overflow? Here's the code",
				"Explain how transformer attention works, including the math",
				"Compare microservices vs monolith architecture for a team of 5 engineers",
				"Refactor this function to use the repository pattern",
				"Debug this SQL query that's returning duplicate rows",
				"Write unit tests for this authentication module",
				"Explain the CAP theorem and when eventual consistency is acceptable",
				"Review this API design and identify security issues",
				"Rewrite this Python script in Go, preserving the same logic",
			},
		},
		{
			Name:          "specialist_dispatch",
			Handler:       "specialist",
			ProviderAlias: "llm-reasoning",
			Fallback:      "llm-reasoning",
			Threshold:     0.62,
			Utterances: []string{
				"What were our sales numbers last quarter?",
				"Search the codebase for all usages of the deprecated auth function",
				"Pull the latest equity data for Madison tract 0054",
				"Run the Atlas pipeline for the new census vintage",
				"Check the status of the gateway health endpoint",
				"What does our deployment runbook say about rollbacks?",
				"Summarize the last 10 customer support tickets",
				"Look up the BLS unemployment rate for Dane County",
				"Find all open GitHub issues tagged bug",
				"What's the current token usage across all tenants this week?",
			},
		},
		{
			Name:          "orchestrated_plan",
			Handler:       "orchestrate",
			ProviderAlias: "llm-reasoning",
			Fallback:      "llm-reasoning",
			Threshold:     0.60,
			Utterances: []string{
				"Analyze our Q1 data, write a summary, and create an HTML report",
				"Set up a new Go service with tests, a Dockerfile, and a deployment config",
				"Research three competing products, compare them, and recommend one",
				"Refactor the auth module, update the tests, and write a migration guide",
				"Pull the latest census data, rerun the Atlas pipeline, and email the results",
				"Audit all 83 skills for quality issues and produce a triage list",
				"Build a landing page, write copy for it, and deploy it to staging",
				"Review the codebase, identify tech debt, and create GitHub issues for each item",
				"Summarize last week's commits, update the changelog, and draft a release note",
				"Find all broken links in the docs, fix them, and submit a PR",
			},
		},
	}
}
