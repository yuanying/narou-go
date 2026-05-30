# narou.rb ストレージ形式

## 概要

narou.rb は小説データを `library/` 配下にファイルシステムで管理する。フォーマットはすべて UTF-8 テキストであり、バイナリや Ruby 固有の Marshal 形式は使用されていない。

## ディレクトリ構造

```
library/
├── .narou/                          # グローバル設定・データベース
│   ├── database.yaml                # 全小説のメタデータ索引
│   ├── database.yaml.backup
│   ├── local_setting.yaml           # ユーザー設定
│   ├── local_setting.yaml.backup
│   ├── freeze.yaml                  # 更新停止小説リスト
│   ├── freeze.yaml.backup
│   ├── lock.yaml                    # ロック情報
│   ├── lock.yaml.backup
│   ├── latest_convert.yaml          # 最終変換日時
│   ├── latest_convert.yaml.backup
│   └── tag_colors.yaml              # タグ色定義
├── 小説データ/
│   ├── 小説家になろう/              # syosetu.com の小説（30 作品）
│   │   └── {ncode} {title}/
│   └── カクヨム/                    # kakuyomu.jp の小説（56 作品）
│       └── {numeric_id} {title}/
└── AozoraEpub3-1/                   # EPUB 変換ツール（Go 実装では不要）
```

macOS 由来の `._*` ファイルおよび `.DS_Store` が混在しているが、これらは無視してよい。

## 小説ディレクトリの命名規則

| サイト | 形式 | 例 |
|--------|------|----|
| 小説家になろう | `{ncode} {title}` | `n1231id 「足手まといなんだ！」...` |
| カクヨム | `{numeric_id} {title}` | `16817139557937843947 転生スライムの復讐譚` |

`ncode` は `n` + 5〜6 文字の英数字（例: `n1231id`）。
`numeric_id` は 17〜19 桁の数値文字列。

これは `database.yaml` の `file_title` フィールドに対応する（サイト名ディレクトリは除く）。

## 各小説ディレクトリの構造

```
{ncode} {title}/
├── toc.yaml             # TOC + 全話メタデータ（必須）
├── toc.yaml.backup      # バックアップ
├── 本文/                # 各話テキストデータ（必須）
│   ├── 1 タイトル.yaml
│   ├── 2 タイトル.yaml
│   └── cache/           # 差分比較用キャッシュ（Go 実装では不要）
│       └── YYYY.MM.DD@HH.MM.SS/
│           └── 1 タイトル.yaml
├── raw/                 # 生 HTML（任意。Go 実装では不使用）
│   └── 1 タイトル.html
├── 挿絵/                # 画像ファイル（挿絵付き作品のみ）
│   └── i{image_id}.jpg
├── setting.ini          # 作品固有の変換設定
├── converter.rb         # カスタム変換スクリプト（Ruby 固有。Go 実装では無視）
├── replace.txt          # 置換ルール（TSV 形式）
├── [著者名] タイトル.epub   # 変換済み EPUB（再生成対象）
└── [著者名] タイトル.mobi   # 変換済み MOBI（再生成対象）
```

### 本文ファイルの命名規則

`{index} {file_subtitle}.yaml`

- 小説家になろう: `index` は連番整数（`1`, `2`, `100` など）
- カクヨム: `index` はエピソード固有の長い数値 ID

例: `1 バッドエンディングなのだけれども。.yaml`

---

## ファイルフォーマット詳細

### `.narou/database.yaml`

全小説のメタデータを管理する YAML ファイル。キーは 0 始まりの整数インデックス。

```yaml
---
0:
  id: 0                                              # 内部 ID（integer）
  author: 野山　歩                                    # 著者名
  title: 勇者の母ですが、魔王軍の幹部になりました。      # タイトル
  file_title: n6437fp 勇者の母ですが、魔王軍の幹部になりました。  # ディレクトリ名（サイト名配下）
  toc_url: https://ncode.syosetu.com/n6437fp/        # TOC URL
  sitename: 小説家になろう                            # サイト名（"小説家になろう" or "カクヨム"）
  novel_type: 1                                      # 1=連載, 2=短編
  end: false                                         # 完結フラグ
  last_update: 2022-12-18 15:59:33.059556000 +09:00  # 最終更新（ダウンロード日時）
  new_arrivals_date: 2022-12-18 15:59:33.059559000 +09:00
  use_subdirectory: false                            # サブディレクトリ分割（通常 false）
  general_firstup: 2019-07-07 22:05:00.000000000 +09:00  # 初回投稿日時
  novelupdated_at: 2022-12-11 21:33:00.000000000 +09:00  # 小説更新日時
  general_lastup: 2022-12-11 19:32:00.000000000 +09:00
  length: 931245                                     # 総文字数
  suspend: false                                     # 停止フラグ
  general_all_no: 436                                # 総話数
  last_check_date: 2022-12-18 15:59:33.394082000 +09:00
  tags:                                              # タグ一覧（空または文字列リスト）
  - end
```

**Go 実装で必要なフィールド:**

| フィールド | 型 | 用途 |
|-----------|-----|------|
| `id` | int | 識別子 |
| `author` | string | EPUB メタデータ |
| `title` | string | EPUB メタデータ |
| `file_title` | string | ディレクトリ名の解決 |
| `toc_url` | string | サイト種別の判定 |
| `sitename` | string | ディレクトリパスの構築 |
| `novel_type` | int | 連載/短編の判定 |
| `end` | bool | 完結表示 |
| `tags` | []string | タグ情報 |

### `toc.yaml`

各小説のメタデータと全話の目次を保持する。

```yaml
---
title: "タイトル"
author: 著者名
toc_url: https://ncode.syosetu.com/n1231id/   # サイト URL
story: |-                                      # あらすじ（プレーンテキスト）
  あらすじ本文...
subtitles:
- index: '1'                                   # 話番号（文字列）
  href: "/n1231id/1/"                          # 相対 URL パス
  chapter: ''                                  # 章タイトル（空文字の場合あり）
  subchapter: ''                               # サブ章（空文字の場合あり）
  subtitle: その01                              # 話タイトル
  file_subtitle: その01                         # ファイル名用タイトル（特殊文字除去済み）
  subdate: 2023/03/19 05:00                    # 初回投稿日時（文字列）
  subupdate: 2023/03/22 12:26                  # 最終更新日時（文字列）
  download_time: 2023-11-02 17:06:30.481572000 +09:00  # ダウンロード日時
```

**カクヨムの差異:**

- `index`: 連番ではなくエピソード固有の長い数値 ID（例: `'16817330650333494286'`）
- `href`: `/works/{work_id}/episodes/{episode_id}` 形式
- `subdate`/`subupdate`: ISO 8601 形式（`2022-12-10T09:07:16Z`）
- `story`: HTML タグを含まないプレーンテキスト

**Go 実装で必要なフィールド:**

| フィールド | 型 | 用途 |
|-----------|-----|------|
| `title` | string | EPUB メタデータ |
| `author` | string | EPUB メタデータ |
| `toc_url` | string | 参照情報 |
| `story` | string | あらすじページ |
| `subtitles[].index` | string | ファイル名解決 |
| `subtitles[].chapter` | string | 章見出し |
| `subtitles[].subtitle` | string | 話タイトル |
| `subtitles[].file_subtitle` | string | ファイル名解決 |

### `本文/{index} {file_subtitle}.yaml`

各話のテキストデータ。**本文は HTML 形式で保存されており、青空文庫形式ではない。**

```yaml
---
index: '10'                                    # 話番号（文字列）
href: "/n1231id/10/"                           # 相対 URL パス
chapter: ''                                    # 章タイトル
subchapter: ''                                 # サブ章
subtitle: その10                               # 話タイトル
file_subtitle: その10                          # ファイル名用タイトル
subdate: 2023/03/19 05:00
subupdate: 2023/03/29 07:36
element:
  data_type: html                              # 常に "html"
  introduction: ''                             # 前書き（HTML または空文字）
  postscript: ''                               # 後書き（HTML または空文字）
  body: |-                                     # 本文（HTML）
    <p id="L1">──来たれ、勇者！　魔王討伐を志す者求む！──</p>
    <p id="L2"><br /></p>
    <p id="L3">　という募集が各国でなされた。</p>
```

**`element.body` に出現する HTML パターン:**

| パターン | 説明 |
|---------|------|
| `<p id="L{n}">text</p>` | 本文段落（id は行番号） |
| `<p id="L{n}"><br /></p>` | 空行 |
| `<ruby>base<rt>ruby</rt></ruby>` | ルビ |
| `<em class="emphasisDots">text</em>` | 傍点 |
| `<b>text</b>` | 太字 |
| `<i>text</i>` | 斜体 |
| `<s>text</s>` | 取消線 |
| `<img src="url">` | 挿絵（URL は mitemin.net または相対パス） |
| `<a href="url">text</a>` | リンク（稀） |

`introduction` と `postscript` も同じ HTML 形式。空の場合は空文字列 `''`。

### `挿絵/` ディレクトリ

画像ファイルを保持する。すべて JPEG 形式。

```
挿絵/
├── i514881.jpg    # 命名規則: i{image_id}.jpg
├── i514884.jpg
└── i515794.jpg
```

画像 ID は mitemin.net の URL に対応する（`https://{id2}.mitemin.net/userpageimage/viewimage/icode/{id1}/`）。

**Go 実装での利用:** `element.body` 中の `<img src="...">` URL を解決し、対応する `挿絵/` 内の画像ファイルを EPUB に埋め込む。

### `setting.ini`

作品固有の変換設定。INI 形式で、行頭 `;` はコメント。デフォルトでは全行コメントアウトされており、ユーザーが必要な設定のみ有効化する。

```ini
; 横書きにする
; enable_yokogaki = false

; 数字の漢数字変換を有効にする
; enable_convert_num_to_kanji = true

; ルビを有効にする
; enable_ruby = true
```

**Go 実装での扱い:** MVP では無視可。主要な設定（横書き/縦書き、ルビ有無）のみ対応を検討する。

### `replace.txt`

作品固有の置換ルール。TSV 形式（タブ区切り）で、行頭 `;` はコメント。

```
; コメント
置換前<TAB>置換後
一〇歳	十歳
```

**Go 実装での扱い:** MVP では無視可。

---

## Go 実装が読み込むファイル一覧

| ファイル | 必須 | 内容 |
|---------|------|------|
| `.narou/database.yaml` | ○ | 小説一覧・メタデータ |
| `{novel_dir}/toc.yaml` | ○ | 目次・話メタデータ |
| `{novel_dir}/本文/{index} {title}.yaml` | ○ | 各話テキスト（HTML） |
| `{novel_dir}/挿絵/*.jpg` | △ | 挿絵画像 |
| `{novel_dir}/setting.ini` | × | 変換設定（MVP では無視） |
| `{novel_dir}/replace.txt` | × | 置換ルール（MVP では無視） |

`raw/` ディレクトリ、`cache/` ディレクトリ、`converter.rb`、`.epub`/`.mobi` ファイルは Go 実装では不要。
