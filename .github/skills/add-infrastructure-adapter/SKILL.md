---
name: add-infrastructure-adapter
description: 新しい外部サービス接続を追加する際に使用する。LLMプロバイダ、キャッシュバックエンド、外部APIなどのinfrastructure層実装の追加手順に従う。
---

# インフラストラクチャ層の拡張

外部サービスとの接続を追加する際のパターンを示す。既存の実装として、LLMプロバイダ(gemini/noop)、キャッシュ(SQLite/メモリ)、Misskey API、RSS取得がある。

## 新しいLLMプロバイダの追加

LLMプロバイダを追加する場合、ファクトリパターンに従う。

事前に次を確定する。

- プロバイダ名(例: bedrock)
- 必須設定項目(APIキー、モデル名、リージョンなど)
- 失敗時の扱い(初期化失敗でエラー返却するか)

1. internal/infrastructure/llm/に{プロバイダ名}_summarizer.goを作成する
2. repository.SummarizerRepositoryインターフェースを実装する(Summarize, IsEnabledの2メソッド)
3. internal/infrastructure/llm/summarizer.goのNewSummarizerRepository内のswitch文にcaseを追加する
4. interfaces/config/config.goに必要な設定を追加する
5. main.goでinterfaces/configの値をllm.Configへ変換して渡す

コンストラクタはエクスポートせず先頭小文字で定義する。Configから必要なフィールドを取得し、不足する場合はエラーを返す。タイムアウトはcontext.WithTimeoutで制御する。

テストは{プロバイダ名}_summarizer_test.goを追加し、最低限次を検証する。

- 必須設定不足時にエラーを返す
- Summarizeの正常系で期待する要約が返る
- 外部APIエラーをラップして返す
- context timeout時に失敗する

## 新しいキャッシュバックエンドの追加

1. internal/infrastructure/storage/に{バックエンド名}_cache.goを作成する
2. repository.CacheRepositoryインターフェースを実装する(GetLatestPublishedTime, SaveLatestPublishedTime, IsProcessed, MarkAsProcessedの4メソッド)
3. main.goの分岐ロジックに新しいバックエンドの選択肢を追加する

io.Closerの実装が必要な場合はClose() errorメソッドも追加する。CleanupOldGUIDsのようなバックエンド固有のメソッドはmain.goで型アサーションして使用する。

## 新しい外部APIの追加

1. domain/repository/に新しいインターフェースを定義する
2. internal/infrastructure/{サービス名}/にパッケージを作成する
3. インターフェースを実装する(構造体はエクスポートしない、コンストラクタはNew{型名}でインターフェース型を返す)
4. 必要な設定をinterfaces/config/config.goのConfig構造体にenvconfigタグ付きで追加する
5. main.goで具象型を生成しサービスに注入する

接続先やAPIキーなどの実装固有の設定は、infrastructure層内にConfig構造体を定義する。interfaces/config/config.goの設定からinfrastructure層のConfigへの変換はmain.goで行う。

## 設定の追加に伴う対応

1. interfaces/config/config.goにフィールドを追加する
2. config_test.goにテストを追加する
3. main.goで新しい設定を読み取り、infrastructure層に渡す
4. READMEに環境変数の説明を追記する
