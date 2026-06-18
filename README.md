# kubeseal-tui

`kubeseal-tui` is a Go CLI for generating Kubernetes SealedSecret manifests. It ports the behavior of the original `generate-kubeseal.sh` helper into a Cobra command with both prompt-based and flag-driven flows.

## Requirements

- `kubeseal` available on `PATH`
- Access to the Sealed Secrets controller certificate through your current Kubernetes context, unless your `kubeseal` setup already supplies one

## Install

From source:

```sh
go install github.com/porthorian/kubeseal-tui@latest
```

With Homebrew after a release has been published:

```sh
brew tap porthorian/tap
brew install --cask kubeseal-tui
```

## Usage

Run without data or target flags to use the prompt flow:

```sh
kubeseal-tui
```

Use non-interactive mode by passing at least one `--data`, `--data-file`, or `--target` flag:

```sh
kubeseal-tui \
  --name grafana-admin \
  --data admin-user=admin \
  --data-file admin-password=./password.txt \
  --target monitoring=./k8/apps/grafana
```

Multiple targets write one file per namespace/output directory:

```sh
kubeseal-tui \
  --name app-secret \
  --data token=secret-value \
  --target prod=./clusters/prod/app \
  --target staging=./clusters/staging/app
```

Output files are always named:

```text
sealed-<secret-name>.yaml
```

Relative output directories resolve from the current working directory. Existing files are rejected in non-interactive mode unless `--force` is set. In prompt mode, each overwrite is confirmed individually.

## Flags

```text
--name <secret-name>               Secret metadata.name (required in non-interactive mode)
--type <secret-type>               Secret type (default: Opaque)
--data <KEY=VALUE>                 Add plaintext data entry (repeatable)
--data-file <KEY=FILEPATH>         Add data entry from file contents (repeatable)
--target <NAMESPACE=OUTPUT_DIR>    Output target (repeatable)
--controller-namespace <ns>        Sealed Secrets controller namespace (default: sealed-secrets)
--controller-name <name>           Sealed Secrets controller name (default: sealed-secrets)
--force                            Overwrite existing outputs in non-interactive mode
-h, --help                         Show help
```

## Release

Releases are tag-driven through GoReleaser.

Required repository secret:

```text
TAP_GITHUB_TOKEN
```

The token must have write access to `porthorian/homebrew-tap` so GoReleaser can update `Casks/kubeseal-tui.rb`.

Release flow:

```sh
git tag v0.1.0
git push origin v0.1.0
```

Local checks:

```sh
go test ./...
go run github.com/goreleaser/goreleaser/v2@latest check
go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean
```
