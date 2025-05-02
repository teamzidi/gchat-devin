# gchat-devin
Devin Google Chat Bot

## 概要
Google Chat から Devin AI を利用するためのインテグレーションサービスです。

## 機能
- Google Chat から `@Devin` メンションまたは `/devin` コマンドによって Devin インスタンスを立ち上げてプロンプトを送信
- スレッドで Devin の応答を受け取る
- スレッドで追加の要求を Devin に送る
- `@Devin` メンションまたは `/devin usage` コマンドによって API の残りを確認

## 技術スタック
- Go 言語
- Google Cloud Run
- Cloud Firestore
- Secret Manager

## 開発環境のセットアップ
1. 必要なツールをインストール
   - Go 1.21 以上
   - Docker
   - Google Cloud SDK

2. 環境変数の設定
   ```
   export PROJECT=<your-gcp-project-id>
   export DEVIN_API_KEY=<your-devin-api-key>
   ```

3. ローカルでの実行
   ```
   make run
   ```

4. デプロイ
   ```
   make deploy
   ```

## Google Chat との連携方法
1. Google Cloud Console で Google Chat API を有効化
2. Google Chat API の設定で Webhook URL を設定
   - URL: https://<your-service-url>
   - 認証: なし
3. Google Chat でボットを追加

## ライセンス
MIT License
