<!--
Where: docs/deploy-interactive-inputs.md
What: Interactive input behavior for deploy/artifact commands.
Why: Keep prompt UX and fallback rules explicit for maintenance and test design.
-->
# Deploy / Artifact の対話入力仕様

## 1. 目的
TTY 実行時に不足値や衝突値をその場で修正できるようにするため、
`internal/command` で対話入力（`interaction.Prompter`）を行います。

このドキュメントは、入力の優先順位と prompt の発火条件を実装準拠で定義します。

## 2. 対話入力の前提

- TTY 判定: `interaction.IsTerminal(os.Stdin)`
- 入力インターフェース: `internal/infra/interaction.Prompter`
  - `Input(title, suggestions)`
  - `Select(title, options)`
  - `SelectValue(title, options)`

non-TTY または `Prompter=nil` の場合は、既定値採用かエラー返却で処理します。

## 3. `esb deploy` の入力順序

`resolveDeployInputs`（`internal/command/deploy_inputs_resolve.go`）は Confirm で `Edit` が選ばれると再解決ループします。

1. repo root 解決
2. running stack 解決（複数時は選択）
3. compose project 解決
4. env 解決
5. mode 解決
6. artifact root 解決
7. template 解決
8. template ごとの parameters/image runtime 解決（output は artifact id から自動決定）
9. compose files 解決
10. 最終確認（Proceed/Edit）

## 4. prompt 一覧（deploy）

### 4.1 Stack 選択

- 実装: `internal/command/deploy_stack.go`
- ラベル: `stack (project=..., env=...)`
- 無効選択時: warning を出して再入力

### 4.2 Project 入力

- 実装: `internal/command/deploy_inputs_project.go`
- 発火条件: project source が `default` のときのみ
- 既定値: 推論 project（Edit 再実行で `last.Project` 優先）

### 4.3 Environment 入力 / mismatch 解決

- 実装: `internal/command/deploy_inputs_env_mode.go`, `internal/command/deploy_runtime_env.go`
- mismatch prompt には `running/current env`, `source`, `project`, `template` を表示

### 4.4 Mode 入力 / conflict 解決

- 実装: `internal/command/deploy_inputs_env_mode.go`
- mode 未指定: `resolveDeployMode` で選択
- 推論 mode と `--mode` 衝突時:
  - TTY: `resolveDeployModeConflict` で `SelectValue`
  - non-TTY: warning を出して推論 mode を採用
- Edit 再実行: `last.Mode` を default 候補として先頭提示

### 4.5 Template パス入力

- 実装: `internal/command/deploy_template_resolve.go`
- 履歴 + 候補 + 手入力を組み合わせて解決

### 4.6 Artifact Root 入力

- 実装: `internal/command/deploy_inputs_output.go`
- `--artifact-root` 未指定時に解決
- TTY: prompt 表示（既定値: `<repo>/artifacts/<project-env>` または previous）
- non-TTY: 既定レイアウトを自動採用

### 4.7 Compose files 入力

- 実装: `internal/command/deploy_inputs_compose.go`
- 対話 prompt なし（常時自動）
- `--compose-file` 指定時のみ明示値を使用
- 未指定時は `auto`（running project 由来/mode 既定で compose files を解決）

### 4.8 Template Parameters 入力

- 実装: `internal/command/deploy_template_prompt.go`
- 表示:
  - default あり: `[Default: ...]`
  - previous あり: `[Previous: ...]`
  - String 無既定: `[Optional: empty allowed]`
  - それ以外: `[Required]`
- `AllowedValues` がある場合は候補提示 + 値検証

### 4.9 Image Runtime 入力

- 実装: `internal/command/deploy_image_runtime_prompt.go`
- タイトル: `Runtime for image function <fn> (image: <effective image uri>, default: <runtime>)`
- `effective image uri` は template 既定 + `--image-uri` override の合成結果

### 4.10 最終確認

- 実装: `internal/command/deploy_summary.go`
- `Proceed` / `Edit` を `SelectValue` で選択

## 5. `esb artifact apply` の対話入力

- 実装: `internal/command/artifact.go`
- TTY + prompter 有効時、`--artifact` と `--out` が空なら必須入力として prompt
- 空入力は warning を出して再入力
- non-TTY では core 側エラーに委譲

## 6. テスト観点（主要）

- `internal/command/deploy_running_projects_test.go`: stack 選択再入力
- `internal/command/deploy_inputs_flow_test.go`: project/artifact-root/compose prompt
- `internal/command/deploy_inputs_mode_test.go`: mode conflict 選択・再試行
- `internal/command/deploy_template_prompt_test.go`: parameter 表示/AllowedValues 検証
- `internal/command/deploy_image_runtime_prompt_test.go`: image runtime prompt 文脈
- `internal/command/artifact_test.go`: artifact apply 必須入力補完
