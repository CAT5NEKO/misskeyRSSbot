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
- 失敗時の扱い(初期化失敗時とnoopフォールバック失敗時の両方)

プロバイダ名は`LLM_PROVIDER`に設定する小文字値を使う(例: `gemini`, `bedrock`)。

上記3点が未確定なら実装を開始しない。特に必須設定項目は「変数名」「必須/任意」「default値」を表で先に決める。

実装開始ゲート:

- `プロバイダ名`
- `必須/任意/デフォルト表`
- `失敗時の扱い`

この3点がそろうまでコード変更しない。

このプロジェクトでは分岐キーに`LLM_PROVIDER`を使う。値は`cfg.GetLLMConfig().Provider`を経由して`llm.Config.Provider`に渡され、`internal/infrastructure/llm/summarizer.go`の`switch cfg.Provider`で選択される。

このプロジェクトではLLM関連設定を`interfaces/config/config.go`で一元管理する。`LLM_REGION`のような項目をconfigに置くことは本プロジェクトの規約上許容する。

このSkillの対象は「単一の有効プロバイダ選択」のみ。複数プロバイダ同時初期化や実行時切り替えは対象外。

1. internal/infrastructure/llm/に{プロバイダ名}_summarizer.goを作成する
2. repository.SummarizerRepositoryインターフェースを実装する

実装するメソッドシグネチャ:

- `Summarize(ctx context.Context, url, title string) (string, error)`
- `IsEnabled() bool`
3. internal/infrastructure/llm/summarizer.goのNewSummarizerRepository内のswitch文にcaseを追加する

case追加の最小例:

```go
switch cfg.Provider {
case "gemini":
	return newGeminiSummarizer(ctx, cfg)
case "bedrock":
	return newBedrockSummarizer(ctx, cfg)
case "myprovider":
	return newMyproviderSummarizer(ctx, cfg)
case "noop", "":
	return newNoopSummarizer(), nil
default:
	return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
}
```
4. interfaces/config/config.goに必要な設定を追加する
5. main.goでinterfaces/configの値をllm.Configへ変換して渡す

`llm.Config`の対象フィールド:

- `Provider string`
- `APIKey string`
- `Model string`
- `Region string`
- `MaxTokens int`
- `SystemInstruction string`
- `Timeout time.Duration`

`llm.Config`は`internal/infrastructure/llm/summarizer.go`に既存定義があるため、新規に重複定義しない。

環境変数名は既存規約の`LLM_*`を使う。

`config.go`の追加例:

```go
LLMProvider string `envconfig:"LLM_PROVIDER" default:""`
LLMAPIKey   string `envconfig:"LLM_API_KEY"`
LLMModel    string `envconfig:"LLM_MODEL"`
LLMRegion   string `envconfig:"LLM_REGION" default:""`
LLMTimeout  int    `envconfig:"LLM_TIMEOUT" default:"30"`
```

必須/任意の判定手順:

1. プロバイダSDK/API仕様で必須入力を列挙する
2. 列挙した必須入力を`required`として`config.go`に追加する
3. 任意入力のみdefault値を設定する
4. `GetLLMConfig()`に全項目を追加する
5. アダプタコンストラクタでrequired欠落を検証してエラー返却する

エラーは`fmt.Errorf("...: %w", err)`でラップして返す。

required判定は`envconfig`の`required`タグではなく、プロバイダごとにコンストラクタ内で行う。

required判定の最小例:

```go
if cfg.Model == "" {
	return nil, fmt.Errorf("myprovider model is required")
}
```

main.goでは既存パターンに合わせて次の順で配線する。

1. `llmCfg := cfg.GetLLMConfig()`
2. `llm.NewSummarizerRepository(ctx, llm.Config{...})`に`llmCfg`の各フィールドを明示的にマッピング
3. 初期化失敗時は既存実装と同様に`noop`へフォールバック
4. `noop`フォールバックも失敗した場合は`log.Fatal`で終了する

`llmCfg.Timeout`は`GetLLMConfig()`で秒から`time.Duration`へ変換済みの値をそのまま渡す。

配線の最小例:

```go
llmCfg := cfg.GetLLMConfig()
summarizerRepo, err := llm.NewSummarizerRepository(ctx, llm.Config{
	Provider:          llmCfg.Provider,
	APIKey:            llmCfg.APIKey,
	Model:             llmCfg.Model,
	Region:            llmCfg.Region,
	MaxTokens:         llmCfg.MaxTokens,
	Timeout:           llmCfg.Timeout,
	SystemInstruction: llmCfg.SystemInstruction,
})
if err != nil {
	summarizerRepo, err = llm.NewSummarizerRepository(ctx, llm.Config{Provider: "noop"})
	if err != nil {
		log.Fatal("Failed to create fallback noop summarizer:", err)
	}
}
```

コンストラクタはエクスポートせず先頭小文字で定義する。Configから必要なフィールドを取得し、不足する場合はエラーを返す。タイムアウトはcontext.WithTimeoutで制御する。

`context.WithTimeout`は`Summarize`メソッド内で適用する。

`s.timeout <= 0`の場合は`30 * time.Second`を使う。

最小例:

```go
func (s *myProviderSummarizer) Summarize(ctx context.Context, url, title string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return "", nil
}
```

`IsEnabled()`は有効な要約器で`true`を返し、無効化用実装(noop)で`false`を返す。

`IsEnabled()`は`internal/application/rss_feed_service.go`から参照される。

`noop`フォールバック失敗時の`log.Fatal`は防御的ガードであり、通常経路では発生しない想定。

`main.go`では`NewSummarizerRepository`が返した`error != nil`をすべてフォールバック対象として扱う。

`repository.SummarizerRepository`は既存インターフェースを使い、新規定義しない。

タイムアウト値は`config.go`の`LLM_TIMEOUT`(default 30秒)を`GetLLMConfig()`経由で渡す。アダプタ側で固定値を新規定義しない。

既存プロバイダの必須項目例:

- gemini: `LLM_API_KEY`, `LLM_MODEL`
- bedrock: `LLM_MODEL`, `LLM_REGION`

テストは{プロバイダ名}_summarizer_test.goを追加し、最低限次を検証する。

- 必須設定不足時にエラーを返す
- Summarizeの正常系で期待する要約が返る
- 外部APIエラーをラップして返す
- context timeout時に失敗する

テスト時の外部API呼び出しは`httptest`または差し替え可能なクライアントインターフェースでモックし、実ネットワークに依存しない。

テストはテーブル駆動で記述する。

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
