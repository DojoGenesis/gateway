---
name: kata-broken
version: 0.1.0
contract: kata-ai/v1
kind: structured
input_schema: contracts/kata-ai/v1/kata-test-echo/input.schema.json
output_schema: contracts/kata-ai/v1/does-not-exist/output.schema.json
---
Broken on purpose: the output schema is missing.
