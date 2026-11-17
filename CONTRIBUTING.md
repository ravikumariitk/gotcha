# Contributing

Thanks for helping improve Gotcha.

## Before opening a pull request

1. Explain the motivation and the behavior change.
2. Add or update tests for the change.
3. Run:

   ```bash
   gofmt -w .
   go test ./...
   go test -race ./...
   go vet ./...
   ```

4. Update the README or changelog when the public API or security behavior
   changes.

Keep pull requests focused, avoid committing secrets or generated media, and
do not change a published release tag. Security issues should follow
`SECURITY.md` instead of being filed publicly.
