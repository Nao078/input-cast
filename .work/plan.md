# input-cast-bridge 実装タスクリスト

## 方針

- [x] FyneはUI担当、ゲームパッド入力取得は別レイヤーとして分離する
- [x] `/gamepad` のブラウザ Gamepad API 依存を、バックグラウンド常駐アプリで置き換える
- [x] 既存server/overlayは変更を最小にし、入力送信APIは `POST /api/input/gamepad` を流用する
- [x] 初期対応範囲は Linuxクライアント + XInput対応に絞る
- [x] Linux上でXInput互換コントローラとして見える入力を優先して扱う
- [x] GP2040-CEの起動時入力モードとして、Switch / XInput / PS3 / PS5 / Xbox One / キーボードに対応する
- [ ] Windows固有のRawInput / DirectInput対応は後続課題にする

## 通信仕様

- [x] 共通の入力状態モデルを定義する
- [x] 既存serverへ送るHTTPクライアントを作る
- [x] 送信先URLを設定可能にする
- [x] 同一入力状態の重複送信を抑制する
- [x] 送信失敗時のエラー表示と再試行方針を決める
  - 方針: ボタン連打時の負荷を避けるため、未接続時は入力送信しない。送信失敗ごとのログ出力や即時再試行はしない。

送信JSON:

```json
{
  "device_id": "gamepad",
  "buttons": {
    "up": false,
    "down": false,
    "left": true,
    "right": false,
    "b1": true
  }
}
```

## ディレクトリ構成

- [x] `cmd/bridge-fyne/` を作成する
- [x] `internal/bridge/client.go` を作成する
- [x] `internal/bridge/state.go` を作成する
- [x] `internal/gamepad/backend.go` を作成する
- [x] `internal/gamepad/linux_xinput.go` を作成する

想定構成:

```txt
cmd/bridge-fyne/
internal/bridge/client.go
internal/bridge/state.go
internal/gamepad/linux_xinput.go
internal/gamepad/backend.go
```

## Linux Input Backend

- [x] Linux上でXInput互換デバイスを読む方式を決める
- [x] 既存の `internal/input/evdev_linux.go` から流用できる処理を整理する
- [x] `/dev/input/event*` または対応ライブラリ経由の読み取り処理を実装する
- [x] XInput相当のキー/軸入力を内部標準ボタン名へ変換する
- [x] Switch modeのevdev入力を内部標準ボタン名へ変換する
- [x] PS3 modeのevdev入力を内部標準ボタン名へ変換する
- [x] PS5 modeのevdev入力を内部標準ボタン名へ変換する
- [x] Xbox One modeのevdev入力を内部標準ボタン名へ変換する
- [x] キーボード modeのevdev入力を内部標準ボタン名へ変換する
- [x] 入力モードごとのデバイス名/キーコード/軸コードをログまたは診断表示で確認できるようにする
- [x] `input` グループ権限がない場合のエラー表示を作る
- [x] デバイス接続/切断時の再スキャンを確認する
- [x] Linux版で入力がserverへ送信されることを確認する

## Windows Backend

- [x] Windows版は初期対象外として明記する
- [x] Windows XInput対応の要否を後で再評価する
- [x] Windows XInput backendの初期実装を追加する
- [x] Windows XInput backendを `GOOS=windows GOARCH=amd64 go test ./internal/gamepad` で確認する
- [x] Windows向けbridge-fyne本体をLinux上のmingwクロスビルドで確認する
- [ ] Windows上でbridge-fyne本体をビルド確認する
- [ ] Windows実機でXInputコントローラ入力がserverへ送信されることを確認する
- [ ] RawInput / DirectInput / SDL2 対応の要否を後で再評価する

## Fyne UI

- [x] Server URL入力を作る
- [x] Device status表示を作る
- [x] Start / Stopボタンを作る
- [x] Auto start設定を作る
- [x] Start minimized設定を作る
- [x] Polling interval設定を作る
- [x] 入力プレビューを作る
- [x] 接続ログ表示を少量だけ作る
- [x] Layout Editor相当の機能をbridge側に追加する
- [x] config profile切り替えをbridge側に追加する
- [x] 任意名の新規profile作成をbridge側に追加する
- [x] 背景画像アップロードは入れない
- [x] 詳細な履歴表示は入れない

## Bridge Layout Editor

- [x] serverの現在profileを読み込み、bridgeプレビューへ反映する
- [x] `default.json` 固定ではなく、`GGST.json` / `SF6.json` / 任意profileを扱えるようにする
- [x] `Profile` プルダウンへ既存profileを表示する
- [x] `New Profile...` から新規profileを作成できるようにする
- [x] `Load` で選択profileを読み込めるようにする
- [x] `Save` で選択profileへ保存できるようにする
- [x] 編集モード中は `Device Status` と `Log` を非表示にする
- [x] 編集モード中にcontroller位置/サイズ/色/imageを変更できるUIを作る
- [x] 編集モード中に全ボタンの色を一括変更できるUIを作る
- [x] 編集モード中にJSON fine tuningを残す
- [x] 編集モード中にボタンをドラッグして位置変更できるようにする
- [x] 編集モード中にボタン右クリックで個別設定ダイアログを開く
- [x] ボタン右クリックダイアログで表示/非表示、ラベル、履歴ラベル、色、押下時色、文字色、サイズを変更できるようにする
- [x] 色設定UIに現在色と変更後色のプレビューを表示する
- [x] bridgeプレビュー通常時の色を `color`、押下時の色を `pressed_color` に合わせる
- [x] overlay側のcontrollerボタン表示をbridgeプレビューに近づける
- [x] overlay側のボタン文字サイズをボタンサイズ連動にする

## 設定保存

- [x] Server URLを保存する
- [x] Auto startを保存する
- [x] Start minimizedを保存する
- [x] Polling intervalを保存する
- [x] 設定ファイルの保存場所をOSごとに決める

## 動作確認

- [x] server未起動時の表示を確認する
- [x] server起動後に自動復帰できるか確認する
- [x] overlayへ入力が反映されるか確認する
- [x] フルスクリーンゲーム中でも入力が継続するか確認する
- [x] Linuxでビルドできるか確認する
- [x] Windowsビルドは初期対象外として扱う

## ドキュメント

- [x] READMEにbridgeアプリの使い方を追加する
- [x] Linuxの権限設定を追記する
- [x] LinuxクライアントでXInput互換入力を使う方針を追記する
- [x] `/gamepad` はブラウザ向け簡易bridgeとして残す方針を追記する

## 将来課題

- [x] Windowsクライアント対応を検討する
- [ ] Windows実機でbridge-fyneを起動確認する
- [ ] GP2040-CEの各入力モード実機ログを収集する
- [ ] DInput相当モードが必要か検討する
- [ ] RawInput対応を検討する
- [ ] SDL2 backendを検討する
- [ ] 複雑なボタンマッピングUIを検討する
- [ ] タスクトレイ常駐を検討する
- [ ] 自動アップデートまたは配布手順を検討する
