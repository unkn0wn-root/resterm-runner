# resterm-runner

A headless CLI runner for [resterm](https://github.com/unkn0wn-root/resterm).

Requests created in resterm can be executed directly by `resterm-runner` against any configured environment. This enables automated validation of API contracts, regression testing across deployments, and environment-level response comparison as part of a standard build or release process.

## Install

Download a prebuilt binary from the [releases](https://github.com/unkn0wn-root/resterm-runner/releases) page, or build from source:

```sh
go install github.com/unkn0wn-root/resterm-runner@latest
```

## Usage

Run a single named request:

```sh
resterm-runner --file api.http --request login
```

Run all requests in a file against a specific environment:

```sh
resterm-runner --file api.http --env production --all
```

Run requests matching a tag:

```sh
resterm-runner --file api.http --tag smoke --env staging
```

Compare the same request across two environments and generate a JUnit report:

```sh
resterm-runner --file api.http \
  --compare "dev,staging" \
  --compare-base staging \
  --report-junit results.xml
```

Use a positional argument instead of `--file`:

```sh
resterm-runner api.http --all
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--file` | | Path to `.http`/`.rest` file |
| `--env` | | Environment name |
| `--env-file` | | Path to environment file |
| `--request` | | Run a specific named request |
| `--workflow` | | Run a named workflow |
| `--tag` | | Run requests matching a tag |
| `--all` | `false` | Run all requests in the file |
| `--compare` | | Compare environments (comma or space separated) |
| `--compare-base` | | Baseline environment for comparison |
| `--timeout` | `30s` | Request timeout |
| `--insecure` | `false` | Skip TLS certificate verification |
| `--follow` | `true` | Follow redirects |
| `--proxy` | | HTTP proxy URL |
| `--recursive` | `false` | Recursively scan workspace for request files |
| `--workspace` | | Workspace directory to scan |
| `--report-json` | | Write a JSON report to the given path |
| `--report-junit` | | Write a JUnit XML report to the given path |
| `--artifact-dir` | | Write run artifacts to the given directory |
| `--state-dir` | | Directory for persisted runner state |
| `--persist-globals` | `false` | Persist runtime globals and file variables between runs |
| `--persist-auth` | `false` | Persist authentication caches between runs |
| `--history` | `false` | Persist request history |
| `--profile` | `false` | Profile request execution |
| `--version` | | Print version and exit |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | All requests passed |
| 1 | One or more requests failed |
| 2 | Usage or configuration error |

## License

See [LICENSE](LICENSE) for details.
