# ESB CLI

`esb-cli` は ESB 用の producer/apply CLI です。  
主なコマンドは `deploy` / `artifact generate` / `artifact apply` / `version` です。

## 前提

- Go `1.25.1`
- Docker / Docker Compose
- 操作対象の ESB リポジトリ（`docker-compose.*.yml` と `.esb/` を含む）

`esb` コマンドは ESB リポジトリ内で実行してください。  
リポジトリ外から実行すると `EBS repository root not found...` で終了します。

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
- `docs/version.md`
