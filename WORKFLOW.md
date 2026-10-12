---
validation:
  - test -z "$(gofmt -l .)"
  - go build ./...
  - go vet ./...
  - go test ./...
required_checks: []
merge_method: squash
max_turns: 5
timeout_seconds: 3600
concurrency: 1
---
Implement issue #{{.Issue.Number}}: {{.Issue.Title}}.
Follow existing repository conventions. Keep the change scoped to this issue.
Never call the real Finnhub or Massive APIs from tests; use net/http/httptest.
