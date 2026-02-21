# ESB CLI

`esb-cli` は ESB 用の producer/apply CLI です。  
主なコマンドは `deploy` / `artifact generate` / `artifact apply` / `version` です。

## 前提

- Go `1.25.1`
- `mise`
- Docker / Docker Compose
- 操作対象の ESB リポジトリ（`docker-compose.*.yml` と `.esb/` を含む）

`esb` コマンドは ESB リポジトリ内で実行してください。  
リポジトリ外から実行すると `EBS repository root not found...` で終了します。

## 開発セットアップ（mise）

```bash
mise trust
mise install
mise run setup
```

`mise run setup` は `esb-cli` 開発用の Git hook（`lefthook install`）を有効化します。

cert/buildkit の準備は `esb-cli` ではなく、操作対象の ESB リポジトリ側で行ってください。

```bash
cd /path/to/esb
mise trust
mise install
mise run setup
```

## CI

GitHub Actions で `lint` / `test` / `build` を実行します（`.github/workflows/ci.yml`）。
`github.com/poruru-code/esb/pkg/...` を private module として取得するため、
GitHub App 方式（branding-tool と同様）で token を発行します。
Actions Secrets に `ESB_APP_ID` と `ESB_APP_PRIVATE_KEY` を設定してください。
GitHub App は `poruru-code/esb` を read できる必要があります。
ローカルでも同じ内容を以下で再現できます。

```bash
mise run lint
mise run test
mise run build
```

## Lint / 自動更新

```bash
# .golangci.yml を使って lint
mise run lint

# 変更を監視して esb を自動再インストール
mise run dev
```

`air` は `.air.toml` を使い、ファイル変更時に `mise run install` を再実行します。

## インストール

### 1. `go install`（推奨）

```bash
go install github.com/poruru-code/esb-cli/cmd/esb@latest
```

### 2. ソースからビルド

```bash
cd /path/to/esb-cli
go build -o ~/.local/bin/esb ./cmd/esb
```

## コマンド一覧（オプション付き）

### グローバルオプション（全コマンド共通）

- `-h, --help`: ヘルプ表示
- `-t, --template <path> ...`: SAM template パス（複数指定可）
- `-e, --env <name>`: 環境名
- `--env-file <path>`: `.env` ファイルパス

### `esb deploy`

- `-m, --mode <docker|containerd>`
- `-o, --output <dir>`
- `--manifest <path>`
- `-p, --project <name>`
- `--compose-file <file>[,<file>...]`
- `--image-uri <function>=<image-uri>[,...]`
- `--image-runtime <function>=<python|java21>[,...]`
- `--build-only`
- `--bundle-manifest`
- `--no-cache`
- `--with-deps`
- `--secret-env <path>`
- `-v, --verbose`
- `--emoji`
- `--no-emoji`
- `--force`
- `--no-save-defaults`

### `esb artifact generate`

- `-m, --mode <docker|containerd>`
- `-o, --output <dir>`
- `--manifest <path>`
- `-p, --project <name>`
- `--compose-file <file>[,<file>...]`
- `--image-uri <function>=<image-uri>[,...]`
- `--image-runtime <function>=<python|java21>[,...]`
- `--bundle-manifest`
- `--build-images`
- `--no-cache`
- `-v, --verbose`
- `--emoji`
- `--no-emoji`
- `--force`
- `--no-save-defaults`

### `esb artifact apply`

- `--artifact <path>`
- `--out <dir>`
- `--secret-env <path>`

### `esb version`

- 追加オプションなし（グローバルオプションのみ）

補足:
- TTY 実行時は不足値を対話入力で補完できます（例: `output`, `project`, `compose files`, `artifact apply` の必須値）。
- 実装と同期した詳細なヘルプスナップショットは `docs/command-reference.md` を参照してください。

## 使い方

### バージョン確認

```bash
esb version
```

### Deploy（generate + apply）

```bash
esb deploy \
  --template e2e/fixtures/template.e2e.yaml \
  --env dev \
  --mode docker \
  --verbose
```

### Artifact を生成のみ実行

```bash
esb artifact generate \
  --template e2e/fixtures/template.e2e.yaml \
  --env dev \
  --mode docker \
  --manifest .esb/artifacts/esb/dev/artifact.yml
```

### 生成済み Artifact を適用

```bash
esb artifact apply \
  --artifact .esb/artifacts/esb/dev/artifact.yml \
  --out .esb/staging/esb-dev/dev/config \
  --secret-env .env
```

## 詳細ドキュメント

- `docs/architecture.md`
- `docs/build.md`
- `docs/container-management.md`
- `docs/generator-architecture.md`
- `docs/sam-parsing-architecture.md`
- `docs/deploy-interactive-inputs.md`
- `docs/command-reference.md`
- `docs/version.md`
