# Leverless Overlay (レバーレス入力表示ツール)

GP2040-CE系レバーレスコントローラーの入力をブラウザやOBSに表示するためのツールです。

## 💡 最もおすすめの構成（推奨環境）
本ツールは、**「サーバー」を裏で動かし、「Input Cast Client（専用アプリ）」で入力をキャプチャする構成**が最も安定するため推奨されています。


```

[レバーレス] ──> [Input Cast Client (常駐アプリ)] ──> [サーバー (Go)] ──> [OBS (ブラウザソース)]

```

---

## 🚀 クイックスタート (推奨構成での起動手順)

まずは基本となるサーバーと常駐アプリ（Input Cast Client）を起動します。

### 1. サーバーの起動
```bash
# 通常起動
go run ./cmd/server

# 設定ファイルを指定して起動する場合（configs/ 内のファイルを指定）
go run ./cmd/server -config configs/SF6.json

```

💡 **Dockerで起動したい場合**

```bash
cp .env.example .env
docker compose up --build

```

### 2. Input Cast Client の環境設定・ビルド

ゲーム画面（フルスクリーン）の裏でも確実に入力を拾い続けるための常駐アプリを準備します。

#### 🐧 Linux環境の場合

初回のみ必要な依存パッケージ（fpm等）をインストールします。

```bash
sudo apt update
sudo apt install -y ruby ruby-dev build-essential gcc make rpm
sudo gem install --no-document fpm

# パッケージのビルドとインストール
packaging/linux/build-package.sh
sudo apt install ./dist/input-cast-client-0.1.0-1.deb  # Ubuntu/Debian
# または sudo dnf install dist/input-cast-client-0.1.0-1.rpm  # Fedora/RHEL

```

#### 🪟 Windows環境の場合

*※ビルドには `mingw-w64` などのGCC環境が必要です。*

```bash
CGO_ENABLED=1 \
CC=x86_64-w64-mingw32-gcc \
GOOS=windows \
GOARCH=amd64 \
go build -ldflags="-H=windowsgui" -o input-cast-client.exe ./cmd/input-cast-client

```

### 3. Input Cast Client の起動

* **Linux:** `input-cast-client` を実行（またはパスの通った場所でコマンド実行）
* **Windows:** `input-cast-client.exe` をダブルクリック、またはバックグラウンドで実行

---

## 📺 各種URL・接続先一覧

起動後、用途に合わせて以下のURLを使い分けます。

| 用途 | URL | 概要 |
| --- | --- | --- |
| **OBS表示用** | `http://localhost:8080/overlay` | OBSの「ブラウザソース」にこれを貼り付けます。 |
| **画面プレビュー** | `http://localhost:8080/preview` | ブラウザで現在の見え方を確認できます。 |
| **ブラウザ版Client** | `http://localhost:8080/gamepad` | WSL環境など、ネイティブアプリが動かない時の簡易ブラウザ版。 |

---

## 🛠️ レイアウトのカスタマイズ方法

ボタンの配置や色の変更は、**Input Cast Client（常駐アプリ）** またはブラウザで `/gamepad` を開いた中にある **Layout Editor** から直感的に行えます。

* **位置変更:** プレビュー上のボタンを**ドラッグ＆ドロップ**
* **詳細設定:** ボタンを**右クリック**（表示/非表示、ラベル、カラー、サイズ変更）
* **全体変更:** `All Button Colors` から全ボタンの一括色替え
* **保存先:** 設定は `configs/*.json` に保存され、次回起動時に自動で復元されます。

---

## ❓ 動かない時のトラブルシューティング

### 1. Linuxで入力を検知しない

`/dev/input/event*` の読み取り権限が足りていない可能性があります。

```bash
# 対策1: root権限で実行する
sudo ./input-cast-client

# 対策2: ユーザーをinputグループに追加して再ログイン
sudo usermod -a -G input "$USER"

```

### 2. Windows（ブラウザ版Client）で「No gamepad visible」と出る

1. コントローラーのボタンを何か1回押してブラウザに認識させてください。
2. 認識しない場合は、Windowsの `joy.cpl`（ゲームコントローラー設定）を開き、OS自体がコントローラーを認識しているか確認してください。
3. 認識していない場合、GP2040-CEの起動モードを **XInputモード** に切り替えて接続し直してください。

---

## 🛠️ 開発者向け情報 (API仕様)

### エンドポイント一覧

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/ws` | WebSocket（入力状態・設定リアルタイム配信） |
| `GET` | `/api/config` | 現在の設定JSONを取得 |
| `POST` | `/api/config` | 設定JSONを保存 |
| `GET` | `/api/config/profiles` | プロファイル一覧取得 |
| `POST` | `/api/config/profile` | プロファイル切り替え |
| `POST` | `/api/input/gamepad` | Bridgeからの入力送信レシーバー |

**Mock（テスト用）入力送信例:**

```bash
curl -X POST http://localhost:8080/api/input/mock \
  -H 'Content-Type: application/json' \
  -d '{"device_id":"mock","buttons":{"left":true,"b1":true}}'

```
