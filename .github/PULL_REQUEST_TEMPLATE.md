<!-- Keep it short and factual. The reviewer knows the codebase. -->

## What changed

<!-- One paragraph. The behaviour that is different, not the files. -->

## Why

<!-- The problem it solves, or the issue it closes. -->

Fixes #

## How it was checked

<!-- Commands and manual steps. At minimum `make check`. -->

```sh
make check
```

## Checklist

- [ ] `make check` passes (gofmt, vet, tests)
- [ ] New behaviour is covered by a test, or the PR says why it cannot be
- [ ] User-visible strings are in `locales/en.json` and `locales/zh-Hans.json`
- [ ] `docs/functional-spec.md` or `docs/design-system.md` is updated when a rule or a surface changed
- [ ] No credentials, private paths or personal data in the diff
