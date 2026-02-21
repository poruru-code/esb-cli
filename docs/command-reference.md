<!--
Where: docs/command-reference.md
What: CLI command/option reference snapshot.
Why: Keep README-level option lists traceable to actual parser output.
-->
# コマンドリファレンス（`--help` スナップショット）

このドキュメントは、`go run ./cmd/esb ... --help` の出力を基にした実装準拠スナップショットです。

## 再生成手順

```bash
go run ./cmd/esb --help
go run ./cmd/esb deploy --help
go run ./cmd/esb artifact --help
go run ./cmd/esb artifact generate --help
go run ./cmd/esb artifact apply --help
```

## `esb --help`

```text
Usage: esb <command> [flags]

Flags:
  -h, --help                     Show context-sensitive help.
  -t, --template=TEMPLATE,...    Path to SAM template (repeatable)
  -e, --env=STRING               Environment name
      --env-file=STRING          Path to .env file

Commands:
  deploy [flags]
    Deploy functions

  artifact generate [flags]
    Generate artifacts and manifest (without apply)

  artifact apply [flags]
    Apply artifact manifest

  version [flags]
    Show version information

Run "esb <command> --help" for more information on a command.
```

## `esb deploy --help`

```text
Usage: esb deploy [flags]

Deploy functions

Flags:
  -h, --help                       Show context-sensitive help.
  -t, --template=TEMPLATE,...      Path to SAM template (repeatable)
  -e, --env=STRING                 Environment name
      --env-file=STRING            Path to .env file

  -m, --mode=STRING                Runtime mode (docker/containerd)
      --artifact-root=STRING       Artifact root directory (artifact.yml +
                                   artifacts/)
  -p, --project=STRING             Compose project name to target
      --compose-file=COMPOSE-FILE,...
                                   Compose file(s) to use (repeatable or
                                   comma-separated)
      --image-uri=IMAGE-URI,...    Image URI override for image functions
                                   (<function>=<image-uri>)
      --image-runtime=IMAGE-RUNTIME,...
                                   Runtime override for image functions
                                   (<function>=<python|java21>)
      --build-only                 Build only (skip provisioner and runtime
                                   sync)
      --bundle-manifest            Write bundle manifest (for bundling)
      --no-cache                   Do not use cache when building images
      --with-deps                  Start dependent services when running
                                   provisioner
      --secret-env=STRING          Path to secret env file for apply phase
  -v, --verbose                    Verbose output
      --emoji                      Enable emoji output (default: auto)
      --no-emoji                   Disable emoji output
      --force                      Allow environment mismatch with running
                                   gateway (skip auto-alignment)
      --no-save-defaults           Do not persist deploy defaults
```

## `esb artifact --help`

```text
Usage: esb artifact <command> [flags]

Artifact operations

Flags:
  -h, --help                     Show context-sensitive help.
  -t, --template=TEMPLATE,...    Path to SAM template (repeatable)
  -e, --env=STRING               Environment name
      --env-file=STRING          Path to .env file

Commands:
  artifact generate [flags]
    Generate artifacts and manifest (without apply)

  artifact apply [flags]
    Apply artifact manifest
```

## `esb artifact generate --help`

```text
Usage: esb artifact generate [flags]

Generate artifacts and manifest (without apply)

Flags:
  -h, --help                       Show context-sensitive help.
  -t, --template=TEMPLATE,...      Path to SAM template (repeatable)
  -e, --env=STRING                 Environment name
      --env-file=STRING            Path to .env file

  -m, --mode=STRING                Runtime mode (docker/containerd)
      --artifact-root=STRING       Artifact root directory (artifact.yml +
                                   artifacts/)
  -p, --project=STRING             Compose project name to target
      --compose-file=COMPOSE-FILE,...
                                   Compose file(s) to use (repeatable or
                                   comma-separated)
      --image-uri=IMAGE-URI,...    Image URI override for image functions
                                   (<function>=<image-uri>)
      --image-runtime=IMAGE-RUNTIME,...
                                   Runtime override for image functions
                                   (<function>=<python|java21>)
      --bundle-manifest            Write bundle manifest (for bundling)
      --build-images               Build base/function images during generate
      --no-cache                   Do not use cache when building images
  -v, --verbose                    Verbose output
      --emoji                      Enable emoji output (default: auto)
      --no-emoji                   Disable emoji output
      --force                      Allow environment mismatch with running
                                   gateway (skip auto-alignment)
      --no-save-defaults           Do not persist deploy defaults
```

## `esb artifact apply --help`

```text
Usage: esb artifact apply [flags]

Apply artifact manifest

Flags:
  -h, --help                     Show context-sensitive help.
  -t, --template=TEMPLATE,...    Path to SAM template (repeatable)
  -e, --env=STRING               Environment name
      --env-file=STRING          Path to .env file

      --artifact=STRING          Path to artifact manifest (artifact.yml)
      --out=STRING               Output config directory
      --secret-env=STRING        Path to secret env file
```
