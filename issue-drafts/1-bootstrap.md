## Goal
Make the repository a proper Go module that builds with a current toolchain and is checked by CI. Right now there is no `go.mod`, so `go build ./...`, `go vet ./...` and `go test ./...` all fail.

## Scope
- Add `go.mod` with `module github.com/actuallyKasi/stock` and a `go` directive for a current release (1.24 or later), plus `go.sum`. The only dependency is `github.com/spf13/pflag`.
- Replace the deprecated `io/ioutil` with `io.ReadAll`.
- Run `gofmt` across the repository.
- Add `.github/workflows/ci.yml`. On push and pull_request to `master` it runs `actions/setup-go` using `go-version-file: go.mod`, then a gofmt check (`test -z "$(gofmt -l .)"`), `go build ./...`, `go vet ./...` and `go test ./...`.
- Add a `.gitignore` for the built `stock` binary.
- **Out of scope:** behaviour changes, refactoring, and switching the API provider. Later issues handle those.

## Acceptance criteria
- [ ] `go build ./...`, `go vet ./...` and `go test ./...` succeed from a clean clone.
- [ ] `gofmt -l .` prints nothing.
- [ ] No `io/ioutil` imports remain.
- [ ] The CI workflow exists and runs the four checks above.
- [ ] The CLI's flags and output are unchanged.

## Validation
No new tests are needed. The workflow's validation commands must pass.
