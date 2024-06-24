# instant-playback-sync
videoタグの動画をネットワーク越しに一緒に見るためのサービス

# サービスをホストする
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
httpsのサイトでは`wss://`でないと接続できないので、どうにかしてhttpsのリバースプロキシの後ろにおいてテストしてください。
```sh
LOG_LEVEL=DEBUG go run main.go
```

## 環境変数
- `LOG_LEVEL` : ログレベル。`DEBUG`, `INFO`(default), `WARN`, `ERROR`
