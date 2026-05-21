# Leverless Overlay

GP2040-CE 系レバーレスコントローラ向けの入力表示ツール MVP です。OBS には Browser Source として overlay URL を読み込ませます。

## 起動

```bash
go run ./cmd/server
```

設定ファイルを指定して起動する場合:

```bash
go run ./cmd/server -config SF6.json
```

設定プロファイルは `configs/*.json` として保存され、`/gamepad` の Layout Editor から `SF6.json` / `GGST.json` のように切り替え・保存できます。

Docker で起動する場合:

```bash
cp .env.example .env
docker compose up --build
```

設定ファイルは `/gamepad` の Layout Editor から切り替えます。最後に読み込んだプロファイル名は `configs/.active-profile` に保存され、次回起動時に自動で復元されます。

Docker 実行時も `configs/` はコンテナへマウントされるため、Layout Editor の保存内容はホスト側の `configs/*.json` に残ります。背景画像アップロードは `web/overlay/uploads/` に保存されます。

OBS Browser Source の URL:

```txt
http://localhost:8080/overlay
```

プレビュー URL:

```txt
http://localhost:8080/preview
```

入力側ブラウザ:

```txt
http://localhost:8080/gamepad
```

表示側は OBS Browser Source で `/overlay` を開き、入力側は通常のブラウザで `/gamepad` を開きます。どちらもブラウザで完結するため、別途クライアントアプリはありません。

reverse proxy のサブパスで公開する場合は、`/api/` や `/ws` をグローバル location にせず、アプリ全体を同じ prefix にまとめてください。例:

```nginx
location = /input-cast {
    return 301 /input-cast/;
}

location ^~ /input-cast/ {
    proxy_pass http://192.168.1.58:12000/;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
}
```

この構成では `https://<host>/input-cast/overlay` と `https://<host>/input-cast/gamepad` でアクセスします。

## API

- `GET /overlay`: OBS 用 overlay HTML
- `GET /preview`: preview 用 HTML
- `GET /ws`: overlay へ入力状態と設定を配信する WebSocket
- `GET /api/config`: 現在の設定 JSON を取得
- `POST /api/config`: 設定 JSON を保存し、接続中 overlay へ即時配信
- `POST /api/input/mock`: mock 入力状態を送信

## GP2040-CE 実機入力

Linux では `oshid` provider が `/dev/input/event*` を定期スキャンし、見つかったゲームパッド入力を内部標準ボタン名へ変換して overlay へ配信します。server 起動後にGP2040-CEを接続しても数秒以内に検出します。

```bash
go run ./cmd/server
```

別ターミナルで provider 状態を確認できます。

```bash
curl http://localhost:8080/api/providers
```

`/dev/input/event*` の読み取り権限がない場合は入力を読めません。開発時の確認では次のどちらかを使ってください。

```bash
sudo go run ./cmd/server
```

または、ユーザーを `input` グループに追加して再ログインします。

```bash
sudo usermod -a -G input "$USER"
```

Docker コンテナ内で Linux の `/dev/input/event*` を読む場合は、実行環境に応じて `compose.yml` に `/dev/input` のデバイスマウントや権限設定を追加してください。Windows / WSL では、通常は実機入力をコンテナに直接渡さず、Windows 側ブラウザの `/gamepad` を Gamepad API bridge として使う構成が扱いやすいです。

WSL のように `/dev/input` が存在しない環境では、Linux evdev から実機を読めません。その場合は Windows 側のブラウザで次を開き、Gamepad API bridge として使います。

```txt
http://localhost:8080/gamepad
```

GP2040-CE のボタンを一度押してブラウザに認識させてから `Start` を押してください。bridge は `/api/input/gamepad` に入力状態を送り、overlay へ配信します。

`No gamepad visible` のままの場合は、bridge の `Scan Gamepads` と `Check WebHID` を押して切り分けます。

- `Scan Gamepads` で何も出ない: ブラウザの Gamepad API から見えていません。Windows の `joy.cpl` で認識状態を確認してください。
- `joy.cpl` にも出ない: GP2040-CE の入力モードを XInput / DInput / Switch などに切り替えて再接続してください。
- `Check WebHID` には出るが Gamepad API に出ない: HIDとしては見えていますがゲームパッドとして公開されていません。GP2040-CEの入力モード変更が必要です。
- OBS Browser Source内ではなく、まず Chrome または Edge の通常タブで `http://localhost:8080/gamepad` を開いて確認してください。

mock 入力例:

```bash
curl -X POST http://localhost:8080/api/input/mock \
  -H 'Content-Type: application/json' \
  -d '{"device_id":"mock","buttons":{"left":true,"b1":true,"b4":true}}'
```

離上例:

```bash
curl -X POST http://localhost:8080/api/input/mock \
  -H 'Content-Type: application/json' \
  -d '{"device_id":"mock","buttons":{"left":false,"b1":false,"b4":false}}'
```

## 設定

設定は `configs/default.json` に保存されます。`overlay.width` / `overlay.height` を基準解像度として、OBS Browser Source の表示サイズに合わせて自動スケールします。初期設定はフルHD、つまり `1920x1080` です。

`history.enabled` で入力履歴の表示を切り替えられます。仮想コントローラは履歴設定に関係なく常に表示されます。履歴は状態が変わるたびに、`フレーム数`、`矢印入力`、`ボタン` の順で表示します。同時押しは `↓←` や `B1+B2` のようにまとめて表示します。

内部標準ボタン名は GP2040-CE の汎用名に寄せて、`up` / `down` / `left` / `right` / `b1`-`b4` / `l1` / `l2` / `r1` / `r2` / `s1` / `s2` / `l3` / `r3` / `a1` / `a2` を使います。

## 入力取得方針

MVP では実 HID 読み取りには踏み込まず、`internal/input.Provider` 経由の mock 入力だけで完成形を動かします。次の段階で Web Gamepad API、OS HID、WebHID、serial などの provider を追加する想定です。

GP2040-CE は XInput、DInput、Switch、PS 系など複数の入力モードを持つため、各 provider は取得したデバイス固有の入力名を上記の内部標準名へ正規化してから server に渡します。OBS Browser Source 内で Gamepad API を直接扱う方式は環境差や権限差が出やすいため、server 経由で WebSocket 配信する構成を初期方針にしています。

## License

MIT License. See [LICENSE](LICENSE).
