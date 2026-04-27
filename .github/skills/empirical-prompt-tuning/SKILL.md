---
name: empirical-prompt-tuning
description: Skillsやプロンプトの再現性を改善する際に使用する。固定シナリオ、別AI実行、両面評価、1反復1テーマ、hold-out検証、停止条件に基づいて反復する。
---

# Empirical Prompt Tuning

Skillsの品質を主観ではなく再現性で評価し、暗黙知を除去するための実践手順。

## いつ使うか

- 新しく作ったSkillが別セッションで再現しない
- 指示が曖昧で実行AIの裁量に依存している
- 成功判定がぼやけていて改善ポイントが定まらない
- 既存Skillを継続的に鍛錬したい

## 原則

- 固定シナリオを先に作る。改善途中でシナリオを変更しない
- 毎反復で新しい実行AIを使う。同一会話の使い回しは禁止
- 1反復1テーマで最小修正する。複数テーマ同時修正は禁止
- 要件チェックリストに少なくとも1つの[critical]を含める
- [critical]がすべて○でなければ成功扱いにしない
- 量的指標は補助、質的指標を主判定にする

## 手順

1. 対象Skillと評価範囲を固定する
2. 事前静的チェックを行う
3. シナリオを2から3件作る
4. 要件チェックリストを作る
5. 実行AI用テンプレートで各シナリオを評価実行する
6. レポートから不明瞭点を1件だけ選ぶ
7. Skill本文を最小修正する
8. 新しい実行AIで再評価する
9. 収束、発散、打ち切りの条件で継続判断する

## 事前静的チェック

- frontmatterのdescriptionが宣言する範囲と本文のカバー範囲が一致しているか
- 具体手順が不足していないか
- 必須アウトプット項目がテンプレートと一致しているか

## 必須アウトプット

各シナリオで次を回収する。

- 成果物
- 要件達成: 各項目の○/×/部分的と理由
- 不明瞭点: 解釈に迷った箇所
- 裁量補完: 指示不足を自分判断で埋めた箇所
- 再試行: やり直し回数と理由
- 指示側計測: 所要時間、使用ツール数、チェック達成率

## 実行テンプレート

- 実行AI依頼文: [assets/executor-prompt-template.md](./assets/executor-prompt-template.md)
- シナリオ定義: [assets/scenario-template.md](./assets/scenario-template.md)
- 要件チェックリスト: [assets/requirements-template.md](./assets/requirements-template.md)
- 反復ログ: [assets/iteration-log-template.md](./assets/iteration-log-template.md)
- 評価基準と停止条件: [references/evaluation-rules.md](./references/evaluation-rules.md)
- このリポジトリ用シナリオv1: [assets/misskey-scenarios-v1.md](./assets/misskey-scenarios-v1.md)
- このリポジトリ用要件v1: [assets/misskey-requirements-v1.md](./assets/misskey-requirements-v1.md)

## 失敗時の対処

- 実行AIが自己回答して別AI実行しない場合: 実行依頼に「別AIとして白紙読解で実行する」を明示する
- レポート形式が崩れる場合: 出力構造をテンプレートで固定し再実行する
- 並列評価で不安定な場合: 並列度を下げて直列化する
- 反復しても不明瞭点が減らない場合: パッチ修正を止めてSkill構造を再設計する
- 別AI dispatchが使えない環境の場合: 評価をスキップした事実と理由を記録し、自己再読を代替にしない

## 禁止事項

- シナリオを後出しで調整して成功率を上げること
- 同一実行AIの記憶に依存して改善判定すること
- メトリクスだけで良し悪しを決めること
- 1反復で複数論点を同時に直すこと
