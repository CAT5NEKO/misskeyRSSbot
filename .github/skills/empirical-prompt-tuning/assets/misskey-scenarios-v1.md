# misskeyRSSbot Skill評価シナリオ v1

固定ルール: 本ファイルは評価反復中に編集しない。変更する場合はv2を新規作成する。

## S1 median: 新規設定値追加
- 目的: 典型的な機能拡張でSkillの再現性を確認する
- 対象Skill: implement-feature
- 前提: RSS取得間隔に関連する新しい環境変数を1つ追加し、main.goまで配線する
- 入力: 追加する設定名とデフォルト値
- 期待成果物:
  - interfaces/config/config.goの設定追加
  - config_test.goの検証追加
  - main.goの配線更新

## S2 edge: 新規外部アダプタ追加
- 目的: 複数層に跨る複雑変更で曖昧点を検出する
- 対象Skill: add-infrastructure-adapter
- 前提: 新しいLLMプロバイダを追加し、既存ファクトリに統合する
- 入力: プロバイダ名、必須設定項目
- 期待成果物:
  - internal/infrastructure/llm/{provider}_summarizer.go
  - summarizer.goのfactory分岐追加
  - 該当テスト追加

## S3 hold-out: 設計相談から実装方針決定
- 目的: 過適合を検出する
- 対象Skill: architecture-consultation
- 前提: 新機能要件を渡し、どの層に何を置くかを判断させる
- 入力: 要件文のみ
- 期待成果物:
  - 層ごとの責務分割案
  - 新規IFの要否判断
  - main.goでのDI方針
