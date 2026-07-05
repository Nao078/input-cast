# Leverless Overlay (レバーレス入力表示ツール)

GP2040-CE系レバーレスコントローラーの入力をブラウザやOBSに表示するためのツールです。

## 安全方針

input-cast はゲームプロセス、ゲームメモリ、描画APIにアクセスしません。
入力デバイスから取得した入力状態を、配信・練習用に可視化する外部ツールです。

自動入力、マクロ、入力補正、ゲーム状態解析、メモリ読み取りは行いません。

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
* **入力履歴コピー:** `history.max_entries` は画面表示件数、`history.copy_max_entries` はコピーに残す履歴件数です。`copy_max_entries` は未指定時100件です。

## 🧩 コンボ表示

Input Cast Client の **Combo** セクションから YAML をアップロードすると、overlay の入力履歴右側にコンボ成立状況を表示できます。

```yaml
version: 1
game: "Street Fighter 6"
character: "Ryu"

commands:
  - id: hadoken
    name: "波動拳"
    notation: "236P"
    maxGapFrames: 8

moves:
  - id: ryu_5lp
    name: "立ち弱P"
    notation: "5LP"
    input: "5LP"
    tags: ["normal", "light"]
    startup: 4
    active: 3
    recovery: 7
    cancelWindows:
      - type: "chain"
        start: 8
        end: 18
        targetTags: ["light"]
      - type: "special"
        start: 10
        end: 22
        targetTags: ["special"]

  - id: ryu_2lk
    name: "しゃがみ弱K"
    notation: "2LK"
    input: "2LK"
    tags: ["normal", "light"]
    startup: 5
    active: 2
    recovery: 10
    cancelWindows:
      - type: "chain"
        start: 8
        end: 18
        targetTags: ["light"]

  - id: ryu_hadoken
    name: "波動拳"
    notation: "236P"
    command: "hadoken"
    tags: ["special"]

recipes:
  - id: ryu_basic_001
    name: "基本: 立ち弱P > しゃがみ弱K > 立ち弱P > 波動拳"
    notation: "LP > 2LK > LP > 236P"
    steps:
      - move: "ryu_5lp"
      - move: "ryu_2lk"
      - move: "ryu_5lp"
      - move: "ryu_hadoken"
    priority: 10

practice:
  mode: "focus"
  activeRecipe: "ryu_basic_001"

practiceSets:
  - id: ryu_beginner
    name: "リュウ基本練習"
    mode: "playlist"
    recipes:
      - ryu_basic_001
    loop: true
    advanceOnComplete: true
```

`commands` は `236P` / `623P` などのコマンド入力成立判定、`moves` は技名・入力・タグ・キャンセル可能タイミング、`recipes` は overlay に表示するコンボです。`commands[].maxGapFrames` は `236P` のようなコマンド内部の方向入力猶予、`moves[].cancelWindows` は技データ由来の標準受付フレームです。

Step 間のタイミング判定は `move.cancelWindows` を使います。前後の step が move 参照ではない場合や、該当する cancel window がない場合は、従来の単純順序判定にフォールバックします。

`input` と `commands[].notation` は `236HP` のような短縮表記に対応します。方向はテンキー表記、ボタンは `configs/*.json` の `id` / `label` / `history_label` に加えて、`LP` / `MP` / `HP` / `LK` / `MK` / `HK` / `P` / `K` の標準エイリアスを使えます。

input-cast はゲームプロセス、ゲームメモリ、描画APIにアクセスしません。入力デバイスから取得した入力状態と YAML の技データをもとに、入力上のコマンド成立・レシピ進行・タイミングを可視化します。ゲーム内のヒット成否、ガード、距離、ジャグル状態は判定しません。

OBS/Preview に表示するレシピは原則1つです。`practice.mode: "focus"` では `activeRecipe` を表示し、`practice.mode: "playlist"` では `activeSet` の playlist から現在の1レシピを表示します。

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
| `GET` | `/api/combos` | コンボYAML/セット一覧と現在選択を取得 |
| `POST` | `/api/combos/upload` | コンボYAMLをアップロード |
| `POST` | `/api/combos/active` | アクティブなコンボセットを切り替え |
| `POST` | `/api/input/gamepad` | Bridgeからの入力送信レシーバー |

**Mock（テスト用）入力送信例:**

```bash
curl -X POST http://localhost:8080/api/input/mock \
  -H 'Content-Type: application/json' \
  -d '{"device_id":"mock","buttons":{"left":true,"b1":true}}'

```
