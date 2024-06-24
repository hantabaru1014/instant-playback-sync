# instant-playback-sync
ネットワーク越しに動画(videoタグを使うあらゆる動画。youtube, primevideo, abema 等)の再生位置を同期するサービス。  
ブラウザ拡張機能として作ってしまうといろいろ面倒なポイントがあるので、ブックマークレットを使用して対象webサイトにスクリプトをインジェクションする。

[https://psync.baru.dev/](https://psync.baru.dev/) にホストしてあるのでお試しできます。  
ただ、絶賛開発中でまともにサービス提供するつもりのある段階じゃないので、ウォチパ中に落ちても怒らないでね(・ω<)☆

# サービスをホストする
httpsのサイト(ほとんどの動画サイト)では、WebSocket通信が`wss://`でないと接続できないので、リバースプロキシ等の後ろにおいてhttpsでホストしてください。

docker-compose.yml
```yaml
version: "3.8"
services:
  app:
    image: "ghcr.io/hantabaru1014/instant-playback-sync:latest"
    ports:
      - 8080:8080
```

# for developer
## 動かす
`localhost:8080`でホストする。  
httpsのサイトでは`wss://`でないと接続できないので、どうにかしてhttps被せるかブラウザ騙してテストしてください。
```sh
LOG_LEVEL=DEBUG go run main.go
```

## 環境変数
- `LOG_LEVEL` : ログレベル。`DEBUG`, `INFO`(default), `WARN`, `ERROR`

# TODO
他の人に使ってもらうまでにやらないといけない最低限のやつ (TODO: これが終わって以降はissue/projectとかで管理すべき)

- ちゃんと接続されたことがわかるように「同期OK」的な表示を画面端に出す
- フロントをsvelteでまともなやつにする
    - 使い方の説明を書く
- bookmarkletもTSで開発できるようにする
