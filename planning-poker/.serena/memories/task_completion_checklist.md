After code changes, run the smallest relevant verification first, then broader checks as needed.
Backend-focused: make lint, make fmt, and targeted go tests (or make tests for full run).
Frontend-focused: in frontend/planning-poker-front run npm run test; run npm run build for production-safety checks when relevant.
If interfaces tied to go:generate mocks changed, run make generate.
Do not hand-edit generated mocks unless repository treats them as source.
Respect existing architecture boundaries and avoid unrelated refactors.