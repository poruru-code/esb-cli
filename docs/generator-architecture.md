<!--
Where: docs/generator-architecture.md
What: Template generator architecture for deploy artifacts.
Why: Explain parse/stage/render boundaries and safe extension paths.
-->
# Generator アーキテクチャ

## 概要
`internal/infra/templategen` は、SAM テンプレートから deploy 用アーティファクトを生成します。

- `functions.yml`
- `routing.yml`
- `resources.yml`
- 各関数の build context / Dockerfile

## 役割分担

- 入口: `internal/infra/templategen/generate.go`
- SAM 解析: `internal/infra/sam/template_parser.go`
- 関数 staging: `internal/infra/templategen/stage.go`
- layer staging: `internal/infra/templategen/stage_layers.go`
- Java runtime 補助: `internal/infra/templategen/stage_java_runtime.go`
- manifest 出力: `internal/infra/templategen/bundle_manifest.go`

## パイプライン

```mermaid
flowchart TD
    A[template.yaml] --> B[ParseSAMTemplate]
    B --> C[FunctionSpec / ResourcesSpec]
    C --> D[applyImageSourceOverrides]
    D --> E[resolveImageFunctionRuntime]
    E --> F[stageFunction]
    F --> G[RenderDockerfile]
    G --> H[write functions.yml / routing.yml / resources.yml]
```

## 生成上のルール

- `ImageSource` を持つ関数も Dockerfile を生成し、`FROM <ImageUri>` で hooks 注入イメージを再ビルドする
- 関数名は `template.ApplyImageNames` で正規化する
- image source は template 既定値に CLI override を上書きして確定する
- image runtime は `python3.12` / `java21` に正規化して生成に渡す
- warnings は `stderr` 系出力へ集約する
- 出力先は `<output>/<env>` 配下で完結する

## 拡張プレイブック

### 1. 新しい関数属性を扱う
1. `internal/infra/sam/template_functions_*.go` で抽出
2. `internal/domain/template/types.go` に必要な型を追加
3. `generate.go` / renderer に反映
4. テスト:
   - `internal/infra/sam/template_functions_test.go`
   - `internal/infra/templategen/generate_test.go`

### 2. 新しい生成ファイルを追加
1. `generate.go` に書き出しを追加
2. `pkg/artifactcore/merge.go` と `internal/usecase/deploy/runtime_config.go` への反映要否を確認
3. テスト:
   - `internal/infra/templategen/generate_test.go`
   - `internal/usecase/deploy/runtime_config_test.go`

### 3. Java runtime hooks staging を変更
1. `stage_java_runtime.go` を更新
2. `runtime-hooks/java/{wrapper,agent}` の JAR 必須コピー契約を維持
3. テスト:
   - `internal/infra/templategen/generate_test.go`

## 変更時の最小テスト

```bash
go test ./internal/infra/sam ./internal/infra/templategen -count=1
```
