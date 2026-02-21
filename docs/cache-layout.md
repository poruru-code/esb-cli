<!--
Where: docs/cache-layout.md
What: Cache layout used by CLI build/deploy pipeline.
Why: Keep staging paths and cleanup rules explicit.
-->
# キャッシュ構成

ステータス: 実装済み（プロジェクトスコープの staging キャッシュ）

## 概要
このドキュメントは、本基盤の deploy/build staging データのキャッシュ構成を定義します。
グローバル設定は **リポジトリルート配下の `.<brand>`** に保持し、deploy のマージ結果と
staging アーティファクトも同じルート配下に配置します。staging のパスは
compose project と env から生成される単一キー（`<compose_project-env>`）で決まり、
ハッシュはパスに使用しません。

## 目的
- グローバルで再利用可能な資産は `<repo_root>/.<brand>` に保持する。
- deploy マージ結果をプロジェクト/環境単位で保存し、プロジェクト間の混入を防ぐ。
- クリーンアップをリポジトリルート配下で完結できるようにする。
- staging ディレクトリ名にハッシュを使わない。

## 目的外
- buildx の内部キャッシュ（BuildKit のストレージ実装/保存場所）は仕様対象外とする。

## 旧挙動（参考）
- グローバル設定は `~/.<brand>/config.yaml` にある。
- staging キャッシュは `~/.<brand>/.cache/staging/<project-hash>/<env>/...` 配下にあった。
- ハッシュは compose project + env から生成され、env もサブディレクトリとして入るため、
  構成が冗長で目視しづらかった。

## 現行挙動（最新）
### グローバル（リポジトリルート配下）
- `<repo_root>/.<brand>/config.yaml` は最近のテンプレートとデフォルト入力を保持する。
- `<repo_root>/.<brand>/certs` / `<repo_root>/.<brand>/wireguard` / `<repo_root>/.<brand>/buildkitd.toml` を使用する。
- 旧 `~/.<brand>` は互換性なしで参照しない。

### プロジェクトスコープ（新しいデフォルト）
リポジトリルートの `.<brand>` をキャッシュルートとして使う：

```
<repo_root>/.<brand>/
  staging/
    <compose_project_env>/
      config/
        functions.yml
        routing.yml
        resources.yml
        .deploy.lock
      services/
      pyproject.toml
```

注記:
- `compose_project_env` は `compose_project` と `env` から生成されるキーです。
- `compose_project` が未指定の場合は `esb` を使用します。
- `env` はデプロイ環境（例: dev, staging）。空の場合は `default`。
- `env` は小文字に正規化されます。
- `compose_project` がすでに `-<env>` で終わる場合は重複付与しません。
- `services/` と `pyproject.toml` はビルド用 staging アーティファクトとして同一パスに配置されます。
- buildx のキャッシュは export/import（`type=local`）を使用せず、buildx builder（BuildKit）内部キャッシュに任せる。

## パスと内容（表）
### グローバルキャッシュ
| パス | 内容 | 目的/備考 |
| --- | --- | --- |
| `<repo_root>/.<brand>/config.yaml` | 最近使ったテンプレートやデフォルト入力 | グローバル設定 |
| `<repo_root>/.<brand>/certs/` | ルート CA などの証明書 | 共有資産 |
| `<repo_root>/.<brand>/wireguard/` | WireGuard 設定/鍵 | 共有資産 |
| `<repo_root>/.<brand>/buildkitd.toml` | buildkitd 設定 | 共有資産 |

### プロジェクトキャッシュ（リポジトリルート配下）
| パス | 内容 | 目的/備考 |
| --- | --- | --- |
| `<repo_root>/.<brand>/staging/<compose_project_env>/config/functions.yml` | 関数定義 | deploy マージ結果 |
| `<repo_root>/.<brand>/staging/<compose_project_env>/config/routing.yml` | ルーティング定義 | deploy マージ結果 |
| `<repo_root>/.<brand>/staging/<compose_project_env>/config/resources.yml` | リソース定義 | deploy マージ結果 |
| `<repo_root>/.<brand>/staging/<compose_project_env>/config/.deploy.lock` | 排他ロック | 並行実行保護 |
| `<repo_root>/.<brand>/staging/<compose_project_env>/services/` | サービス構成 | staging アーティファクト |
| `<repo_root>/.<brand>/staging/<compose_project_env>/pyproject.toml` | 依存/環境設定 | staging アーティファクト |

## パス解決ルール
- staging ルートは固定で `<repo_root>/.<brand>/staging` を使用する。
- リポジトリルート配下の `.<brand>` が書き込み不可の場合はエラーとする。
- `compose_project` の決定順:
  1. `PROJECT_NAME` があればその値
  2. `PROJECT_NAME` が空なら `esb-<env>`（`env` が空なら `esb`）
- 最終パスキーは `<compose_project-env>`（すでに suffix が一致する場合はそのまま）を使用する。

## クリーンアップ
- 1つの env（=1キー）を削除:
  `rm -rf <repo_root>/.<brand>/staging/<compose_project_env>`
- 1つのプロジェクトに紐づくキーを削除:
  `rm -rf <repo_root>/.<brand>/staging/<compose_project>-*`

グローバル設定と証明書は削除対象外。

## 互換性メモ
- 旧レイアウト `<template_dir>/.<brand>/staging` は参照しません（アップデート後は再 deploy が必要です）。
- 旧レイアウト `~/.<brand>/.cache/staging` は現行 CLI では使用しません（レガシーとして残る可能性あり）。
- ハッシュは staging のパスには使いませんが、ビルドの fingerprint 生成では引き続き使用します。

---

## Implementation references
- `internal/infra/staging/staging.go`
- `internal/infra/config/global.go`
- `internal/infra/runtime/env_resolver.go`
- `internal/usecase/deploy/deploy_run.go`
