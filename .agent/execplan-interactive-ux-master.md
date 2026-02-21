# Deploy Interactive UX Master Repair Plan

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

The repository rule in `./AGENTS.md` requires an ExecPlan for complex features and refactors. This document must be maintained in accordance with `./.agent/PLANS.md`.

## Purpose / Big Picture

この修正により、`esb deploy` と `esb artifact apply` のインタラクティブ入力で「何に対して質問されているか」が常に分かり、入力漏れや競合時にその場で修正できるようにする。現在は自動採用や warning のみで進む分岐が複数あり、誤った stack/mode/output を選んだまま処理が進みやすい。修正後は、ユーザーが TTY 上で意図を確認して進めるため、誤デプロイのリスクを下げられる。

目視で確認できる完成条件は次のとおり。`esb deploy` を対話実行した際に、image runtime 質問で対象関数と対象 image URI が表示され、stack/mode 競合や不足値が再入力可能になり、最終確認サマリで実際に適用される image source/runtime が整合して表示される。

## Progress

- [x] (2026-02-21 01:55Z) インタラクティブ UX 監査結果を統合し、修正対象を高・中・低優先度で整理した。
- [x] (2026-02-21 01:55Z) ExecPlan を新規作成し、マイルストーン、検証方法、復旧方針、必要テストを定義した。
- [x] (2026-02-21 01:59Z) Milestone 1 の詳細計画を追加し、ファイル単位の編集内容とテスト方針を固定した。
- [x] (2026-02-21 02:05Z) Milestone 1 を実装し、image runtime/stack/env mismatch の prompt 文脈強化と stack 選択リトライを反映した。
- [x] (2026-02-21 02:24Z) Milestone 2 を実装し、output/project/compose/artifact apply の不足 prompt を追加。`go test ./internal/command -run 'TestResolveDeployOutput|TestResolveDeployProject|TestResolveDeployComposeFiles|TestResolveDeployInputs|TestRunArtifactApply|TestResolveArtifactApplyInputs' -count=1`、`go test ./internal/command -count=1`、`go test ./... -count=1`、`mise run lint` を実行して成功を確認した。
- [x] (2026-02-21 02:36Z) Milestone 3 を実装し、mode 推論値と `--mode` の競合時に TTY では `SelectValue` で採用値を選択できるようにした。Edit 後の再実行では `last.Mode` を default 候補として先頭提示する仕様へ変更。`go test ./internal/command -run 'TestResolveDeployMode|TestResolveDeployModeConflict|TestDeployInputsResolverResolveModePromptsOnInferenceConflict|TestResolveDeployInputsKeepsImageRuntimeAcrossEditLoop' -count=1`、`go test ./internal/command -count=1`、`go test ./... -count=1`、`mise run lint` を実行して成功を確認した。
- [x] (2026-02-21 02:49Z) Milestone 4 を実装し、template 既定 image source を `deployTemplateInput.ImageSources` に保持して summary/実行値の整合を確保した。parameter prompt は `AllowedValues` 検証・候補提示を追加し、`Type: String` の空入力許容を `[Optional: empty allowed]` 表示と非TTY挙動に整合させた。`go test ./internal/command -run 'TestResolveDeployInputs|TestConfirmDeployInputs|TestPromptTemplateParameters|TestExtractSAMParametersMapAnyAny' -count=1`、`go test ./internal/command -count=1`、`go test ./... -count=1`、`mise run test`、`mise run lint` を実行して成功を確認した。
- [x] (2026-02-21 02:49Z) すべての関連 unit test を更新し、`mise run test` と `mise run lint` を通して完了判定した。

## Surprises & Discoveries

- Observation: runtime mode が実行中環境から推論できる場合、`--mode` と衝突しても確認なしで `--mode` が無視される。
  Evidence: `internal/command/deploy_inputs_resolve.go` の `resolveMode` で warning を出した後に推論値を返している。

- Observation: stack 選択で未知値が返ると再入力せず空 stack 扱いになる。
  Evidence: `internal/command/deploy_stack.go` で lookup miss 時に `deployTargetStack{}` を返し、`TestResolveDeployTargetStackMultipleTTYUnknownSelection` がその挙動を固定している。

- Observation: output は interactive でも prompt が一切出ず、previous か空文字が採用される。
  Evidence: `internal/command/deploy_inputs_output.go` と `TestResolveDeployOutputInteractiveUsesDefaultWithoutPrompt`。

- Observation: image runtime prompt は関数名だけを表示し、対象 image URI を表示していない。
  Evidence: `internal/command/deploy_image_runtime_prompt.go` のタイトル生成と `discoverImageFunctionNames` の戻り値仕様。

- Observation: stack 選択で再入力ループを実装するには `errOut` を `resolveDeployTargetStack` へ注入する必要があった。
  Evidence: warning 表示を `writeWarningf` で統一するため、`internal/command/deploy_inputs_resolve.go` の呼び出し側シグネチャ更新が必要だった。

- Observation: Milestone 1 時点の UT は prompt 文脈強化には十分だったが、Milestone 2 の不足 prompt（project/compose/artifact apply）を検知する網羅性は不足していた。
  Evidence: 実装前は `resolveDeployProject` / `resolveDeployComposeFiles` / `resolveArtifactApplyInputs` に相当するテストが存在しなかったため、Milestone 2 で `deploy_inputs_flow_test.go` と `artifact_test.go` に新規ケースを追加した。

- Observation: mode 競合 prompt で Edit ループの前回選択を保持するには、stored defaults ではなく `last.Mode` を default 候補に使う必要があった。
  Evidence: `resolveMode` で `previousSelectedMode := strings.TrimSpace(last.Mode)` を分離し、`resolveDeployModeConflict` へ渡すことで `TestDeployInputsResolverResolveModePromptsOnInferenceConflict` の先頭候補検証が安定した。

- Observation: Milestone 4 で `discoverImageFunctionNames` は未使用となり `golangci-lint` の `unused` で失敗した。
  Evidence: `mise run lint` 実行時に `internal/command/deploy_image_runtime_prompt.go:106:6: func discoverImageFunctionNames is unused` が報告され、関数削除で解消した。

## Decision Log

- Decision: この計画では、まず「既存 prompt の文脈不足」と「誤選択後に戻れない分岐」を先に修正し、その後に不足 prompt を追加する。
  Rationale: 既存の対話経路を壊さずに誤操作率を下げる効果が最も大きく、回帰リスクを段階的に管理できる。
  Date/Author: 2026-02-21 / Codex

- Decision: stack/mode 競合は warning のみで継続する設計から、TTY 時は明示選択を要求する設計へ寄せる。
  Rationale: 明示フラグが暗黙無効化される現在挙動は UX 上の驚きが大きく、運用ミスにつながるため。
  Date/Author: 2026-02-21 / Codex

- Decision: サマリは override のみではなく「実際に採用される image source/runtime」を表示する。
  Rationale: 確認画面が最終的な意思決定点であり、ここが実行内容と不一致だと修正機会が失われるため。
  Date/Author: 2026-02-21 / Codex

- Decision: Milestone 1 では新しい入力項目の追加は行わず、既存 prompt の情報密度と再入力可能性だけを修正対象に限定する。
  Rationale: 変更影響を局所化して回帰範囲を小さくし、Milestone 2 以降の不足 prompt 追加と責務を分離するため。
  Date/Author: 2026-02-21 / Codex

- Decision: image runtime prompt には template 既定 `ImageSource` に加え、`--image-uri` override がある場合は override 値を優先表示する。
  Rationale: 実際に deploy される対象イメージを prompt で即時に確認できるようにし、誤関数選択を減らすため。
  Date/Author: 2026-02-21 / Codex

- Decision: project prompt は「flag/env/host/stack で値が確定している場合は出さず、default 推論時のみ出す」方針にした。
  Rationale: 明示指定を上書きせず、誤採用が起きやすい default 分岐だけを対話化してノイズを最小化するため。
  Date/Author: 2026-02-21 / Codex

- Decision: compose files prompt は `auto`（空配列）を第一既定値とし、Edit ループ時は前回値を default 候補として再利用する。
  Rationale: 既存の「未指定=auto」動作を維持しつつ、再編集時に同じ入力を繰り返す手間を減らすため。
  Date/Author: 2026-02-21 / Codex

- Decision: `artifact apply` の不足値 prompt は TTY + prompter が有効な場合のみ起動し、非TTYは従来どおり core 側エラーへ委譲する。
  Rationale: CI や非対話実行の互換性を保ち、対話時のみ UX を改善するため。
  Date/Author: 2026-02-21 / Codex

- Decision: mode 競合時は TTY で `SelectValue` を使って「推論 mode」か「`--mode`」を明示選択させ、nonTTY は warning を維持して推論 mode を採用する。
  Rationale: 対話実行ではユーザー意思を優先し、非対話実行では既存挙動との互換を維持するため。
  Date/Author: 2026-02-21 / Codex

- Decision: template image source は override のみでなく「template 既定 + override 上書き」の合成結果を `deployTemplateInput.ImageSources` に保持する。
  Rationale: summary 表示と build/generate 実行時に同一の effective image source を使い、確認画面と実処理の不整合を防ぐため。
  Date/Author: 2026-02-21 / Codex

- Decision: parameter prompt で `Type: String` 無既定パラメータは空入力許容を明示 (`[Optional: empty allowed]`) し、nonTTY でも同一ルールを適用する。`AllowedValues` がある場合は候補提示と値検証を行う。
  Rationale: 既存の TTY/非TTY 不整合と `[Required]` 表示の誤解を解消し、入力失敗を prompt 時点で防ぐため。
  Date/Author: 2026-02-21 / Codex

## Outcomes & Retrospective

Milestone 1 から Milestone 4 まですべて完了した。Milestone 4 では、template 既定 image source を `deployTemplateInput.ImageSources` に保持して summary と実処理の整合を確保し、parameter prompt には `AllowedValues` 検証と候補提示、`Type: String` の空入力許容を明示した。Milestone 1 の既存UTは初期スコープには十分だったが、Milestone 2/3/4 で追加した不足 prompt・mode 競合対話・parameter 制約を検知するには不十分だったため、`deploy_inputs_flow_test.go`、`artifact_test.go`、`deploy_inputs_mode_test.go`、`deploy_template_prompt_test.go`、`deploy_inputs_resolve_test.go` に観点を追加して補完した。`go test ./internal/command -count=1`、`go test ./... -count=1`、`mise run test`、`mise run lint` は成功している。

## Context and Orientation

この修正の中心は `internal/command` である。ここでいう TTY は「標準入力がターミナルで、対話入力が可能な実行環境」を意味する。`interaction.Prompter` は `Input`、`Select`、`SelectValue` の 3 種類を提供し、実装は `internal/infra/interaction/selector.go` の `HuhPrompter` が担う。`resolveDeployInputs` が deploy 入力解決の中枢で、template/env/mode/stack/runtime/image 系の prompt はここから各 helper に分岐する。

今回編集対象となる主要ファイルは `internal/command/deploy_image_runtime_prompt.go`、`internal/command/deploy_stack.go`、`internal/command/deploy_inputs_output.go`、`internal/command/deploy_inputs_resolve.go`、`internal/command/deploy_runtime_env.go`、`internal/command/deploy_summary.go`、`internal/command/deploy_template_prompt.go`、`internal/command/artifact.go`。対応テストは同名 `_test.go` 群に追加・更新する。既存アーキテクチャ境界（`command -> usecase -> domain/infra`）は維持し、対話ロジックの責務は `internal/command` に閉じる。

## Plan of Work

### Milestone 1: 既存 prompt の文脈と再入力性を改善する

この段階で、すでに prompt が存在するフローの「分かりにくさ」と「誤選択後の復帰不能」を解消する。`internal/command/deploy_image_runtime_prompt.go` では image 関数抽出を名前だけでなく image source も持つ構造に変更し、タイトルを `function + effective image URI + default runtime` で表示する。`internal/command/deploy_stack.go` では選択肢ラベルに stack 名に加えて project/env を表示し、未知選択や空選択時は warning を出して再選択させる。`internal/command/deploy_runtime_env.go` では env mismatch タイトルに compose project と source を含め、どの推論に基づく不一致かを明示する。

このマイルストーン完了時点で、ユーザーは「どの対象を選んでいるか」を prompt 単体で判断でき、誤入力時にその場で修正できる。

#### Milestone 1 Detailed Plan

このマイルストーンは、すでに存在する 3 つの対話経路だけを対象にする。対象は image runtime 選択、running stack 選択、environment mismatch 選択であり、output や project の新規 prompt 追加は次マイルストーンへ持ち越す。ここでは「質問文だけ見て対象を特定できること」と「無効選択で黙って継続しないこと」を達成条件にする。

`internal/command/deploy_image_runtime_prompt.go` では、`discoverImageFunctionNames` を置き換え、関数名だけでなく template から解決済みの `ImageSource` を保持する内部構造を返すようにする。`promptTemplateImageRuntimes` はこの構造を用いてタイトルを生成し、最低限 `function name` と `image URI` と `default runtime` を表示する。`internal/command/deploy_inputs_resolve.go` の `resolveSingleTemplateInput` からは、既存の `templateImageSources`（`--image-uri` override）を渡し、override がある場合はそれを優先した effective image URI を prompt に表示する。これにより、同名に近い関数が複数ある場合でも、どのイメージを対象に選択しているか即時に判別できる。

`internal/command/deploy_stack.go` では、選択肢文字列を `stack name` のみから、`stack name` と `project/env` を含む表示ラベルへ変更する。表示ラベルと `deployTargetStack` を対応付ける lookup を作り、選択値が空または lookup 不一致の場合は再度 `Select` するループにする。無効入力を受けたことは `writeWarningf` 経由で通知するため、`resolveDeployTargetStack` に `errOut io.Writer` を追加し、呼び出し元の `internal/command/deploy_inputs_resolve.go` から `r.errOut` を渡す。これで「誤選択時に空stackで続行する」経路を塞ぐ。

`internal/command/deploy_runtime_env.go` では、`promptEnvMismatch` のタイトル情報を拡張する。`running env` と `current env` だけでなく、`compose project`、`template`（空なら `<none>`）、`inferred source`、`current source` を表示して、なぜ mismatch が起きたかを prompt 単体で理解できるようにする。`reconcileEnvWithRuntime` から `promptEnvMismatch` を呼ぶ箇所は両方とも新しい引数を渡す。ここで選択肢の value semantics は変えず、採用後の `applyEnvSelection` は既存のまま維持する。

テストは既存ケースの修正と、新規ケースの追加を同時に行う。`internal/command/deploy_image_runtime_prompt_test.go` には prompt title に image URI が含まれることを検証するアサーションを追加し、override image URI が表示されるケースを新設する。`internal/command/deploy_running_projects_test.go` は option label 変更に合わせて選択値と期待値を更新し、未知選択のとき再試行して最終的に有効値で確定するケースへ差し替える。`internal/command/deploy_runtime_env_reconcile_test.go` には `promptEnvMismatch` の title 文字列を検証する prompter を追加し、project/source が含まれることを固定する。

このマイルストーンの実装順は、image runtime、stack selection、env mismatch の順にする。理由は、最初の 2 つが `Select` のタイトル・lookup 変更で影響範囲が比較的限定され、最後の env mismatch で `SelectValue` タイトル変更を加えると差分レビューが追いやすいためである。各段階で `go test ./internal/command -run ...` の対象を絞って回帰を確認し、最後に Milestone 1 専用の統合テストセットを再実行する。

Milestone 1 の完了判定は 3 つの観測で行う。第一に image runtime prompt に対象 image URI が出ること。第二に stack の無効選択で空stackが返らず再入力になること。第三に env mismatch prompt で project/source が見えること。これらが unit test で固定され、かつ `go test ./internal/command -run 'TestPromptTemplateImageRuntimes|TestResolveDeployTargetStack|TestReconcileEnvWithRuntime' -count=1` が成功した時点で Milestone 1 は完了とする。

### Milestone 2: 不足している interactive 入力を追加する

この段階で、現在は暗黙採用される不足値を TTY で補完できるようにする。`internal/command/deploy_inputs_output.go` で `--output` 未指定時の interactive prompt を実装し、既定値は previous または導出候補を提示する。`internal/command/deploy_inputs_resolve.go` の project 解決に「default 採用前の確認または入力」を追加し、無意識の誤 project を防ぐ。compose files も未指定時の入力経路を追加し、明示指定したい運用を対話で救済する。`internal/command/artifact.go` の `runArtifactApply` は `deps.Prompter` と TTY 判定を利用し、`artifact` と `out` が不足する場合に入力させる。

このマイルストーン完了時点で、重要入力の不足は「即失敗」か「暗黙継続」ではなく、TTY 上で修正可能になる。

### Milestone 3: 推論値と明示値の衝突解決を interactive 化する

この段階で、warning のみで値を無視する分岐をなくす。`internal/command/deploy_inputs_resolve.go` の mode 解決で、推論 mode と `--mode` の不一致時に TTY なら `SelectValue` で採用値を選ばせる。非TTY は現行方針を維持しつつ、エラーメッセージを具体化する。さらに Edit ループで再編集可能にするため、`resolveInputIteration` に「前回ユーザー選択値」と「推論由来値」を区別して保持し、`Edit` 後に再度 prompt が出せる条件を明確化する。

このマイルストーン完了時点で、明示値が黙って無視される挙動がなくなり、Edit が実際に再入力として機能する。

### Milestone 4: 最終確認サマリと parameter UX の整合性を取る

この段階で、確認画面を実行結果と一致させる。`internal/command/deploy_inputs_resolve.go` で template の image sources を override のみでなく template 既定値も含めて保持し、`internal/command/deploy_summary.go` で表示する。`internal/command/deploy_template_prompt.go` は `[Required]` 表示と空入力許可の整合を取り、`Type: String` の空入力許容を明示ラベル化するか、必須時は実際に再入力させる。必要に応じて parameter 制約（AllowedValues の候補提示）を追加し、入力失敗を前倒しで防ぐ。

このマイルストーン完了時点で、最終確認の表示内容が実際の deploy request と一致し、parameter 入力の期待と挙動が一致する。

## Concrete Steps

作業ディレクトリは常に `/home/akira/esb-cli` を使う。以下は実装時に順に実行する。

    cd /home/akira/esb-cli
    git status --short

Milestone 1 実装後に対象テストを先行実行する。

    cd /home/akira/esb-cli
    go test ./internal/command -run 'TestPromptTemplateImageRuntimes|TestResolveDeployTargetStack|TestReconcileEnvWithRuntime' -count=1

期待する結果は `ok   github.com/poruru-code/esb-cli/internal/command`。

Milestone 2 実装後に output/project/compose/artifact apply 周辺テストを追加して実行する。

    cd /home/akira/esb-cli
    go test ./internal/command -run 'TestResolveDeployOutput|TestResolveDeployInputs|TestRunArtifactApply' -count=1

Milestone 3 実装後に mode 衝突と Edit ループの回帰テストを実行する。

    cd /home/akira/esb-cli
    go test ./internal/command -run 'TestResolveDeployMode|TestResolveDeployInputsKeepsImageRuntimeAcrossEditLoop|TestResolveDeployInputs' -count=1

Milestone 4 実装後に summary/parameter 系テストを実行する。

    cd /home/akira/esb-cli
    go test ./internal/command -run 'TestConfirmDeployInputs|TestPromptTemplateParameters' -count=1

最後に全体検証を実行する。

    cd /home/akira/esb-cli
    mise run test
    mise run lint

## Validation and Acceptance

受け入れ条件は「ユーザーが間違いに気付き、同一セッション内で修正できること」と「確認画面が実際の適用値を示すこと」である。次の挙動を満たせば完了とする。

`esb deploy` を TTY で実行し image runtime 質問に到達したとき、質問文に関数名と image URI の両方が表示されること。stack 選択で無効値を返した場合は再入力になること。`--output` 未指定時は output が質問されること。推論 mode と `--mode` が衝突した場合は採用値を選択できること。最終確認サマリに image sources と image runtimes が function ごとに表示され、実際の request と一致すること。

テスト観点では、既存テストの修正だけでなく以下を新規追加する。未知 stack 選択時リトライ、mode 衝突時 interactive 選択、output prompt 発火、artifact apply の不足入力 prompt、summary の template 既定 image source 表示、parameter 必須表示と空入力挙動の整合。

## Idempotence and Recovery

この計画の変更はすべて加法的に実施し、各マイルストーンで `go test` を通してから次へ進むため、途中中断しても再開しやすい。interactive 仕様変更でテストが壊れた場合は、まず該当 `_test.go` を新仕様へ更新して意図を固定し、その後実装を合わせる。`resolveDeployInputs` など中枢関数を変更する際は、1 マイルストーンごとに commit を分けることで切り戻し可能性を保つ。破壊的なデータ移行は行わないため、設定ファイル破損リスクは低い。

## Artifacts and Notes

この計画で重要な修正対象の証跡は次のとおり。

    internal/command/deploy_image_runtime_prompt.go: title が function 名のみ
    internal/command/deploy_stack.go: 未知選択時に空 stack を返す
    internal/command/deploy_inputs_output.go: interactive でも prompt しない
    internal/command/deploy_inputs_resolve.go: mode 衝突時に --mode を warning 後に無視
    internal/command/deploy_summary.go: image source は override 起点で、template 既定値が表示対象外

実装完了時には、ここに代表的なテスト出力（失敗から成功への差分）を追記する。

## Interfaces and Dependencies

外部ライブラリ追加は行わず、既存の `internal/infra/interaction.Prompter` を利用する。インターフェースは次を維持しつつ、command 層の関数シグネチャを必要最小限で拡張する。

`internal/infra/interaction/interaction.go`:

    type Prompter interface {
        Input(title string, suggestions []string) (string, error)
        Select(title string, options []string) (string, error)
        SelectValue(title string, options []SelectOption) (string, error)
    }

`internal/command/deploy_image_runtime_prompt.go` では、image 関数を `name` と `image source` の組で扱う内部型を新設し、prompt タイトル生成に使う。`internal/command/deploy_inputs_resolve.go` では、template 既定 image source と override のマージ結果を `deployTemplateInput.ImageSources` に保持する。`internal/command/deploy_stack.go` は再入力ループを導入し、unknown selection を silent fallback させない。`internal/command/artifact.go` は既存の `Dependencies` を使って interactive 入力を追加する。

Revision Note (2026-02-21 01:55Z): 初版作成。インタラクティブ UX 監査結果を、実装順序と検証手順を持つ修正マスタープランへ統合した。
Revision Note (2026-02-21 01:59Z): ユーザー依頼により Milestone 1 の詳細計画を追記。対象範囲、関数単位の変更、テスト更新、完了条件を具体化した。
Revision Note (2026-02-21 02:05Z): Milestone 1 実装完了に伴い Progress/Decision Log/Outcomes を更新。実行済みテスト結果を記録した。
Revision Note (2026-02-21 02:24Z): Milestone 2 実装完了に伴い Progress/Surprises/Decision Log/Outcomes を更新。不足していたUT観点（project/compose/artifact apply）を追加した旨と実行済み検証結果を記録した。
Revision Note (2026-02-21 02:36Z): Milestone 3 実装完了に伴い Progress/Surprises/Decision Log/Outcomes を更新。mode 競合の interactive 解決と Edit ループの default 維持を反映し、追加したUTと検証コマンド結果を記録した。
Revision Note (2026-02-21 02:49Z): Milestone 4 実装完了に伴い Progress/Surprises/Decision Log/Outcomes を更新。summary と実行値の image source 整合、parameter prompt の `AllowedValues`/必須表示整合、最終検証（`mise run test`, `mise run lint`）結果を追記した。
