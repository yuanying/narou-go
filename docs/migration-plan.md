# narou.rb → narou-go 移行計画

## 目標

既存の `library/` ディレクトリを変更せずに narou-go から直接読み込み、EPUB3 を生成できるようにする。narou.rb で管理中の小説を narou-go に**移行**するのではなく、**同一の library を両ツールで共用**する。

---

## library からの情報復元可能性

実際の `library/` データを調査した結果。

### 復元可能な情報（○）

| 情報 | ファイル・フィールド | 備考 |
|------|---------------------|------|
| タイトル | `toc.yaml → title` | |
| 作者名 | `toc.yaml → author` | |
| あらすじ | `toc.yaml → story` | プレーンテキスト |
| TOC URL | `toc.yaml → toc_url` | |
| 章タイトル | `toc.yaml → subtitles[].chapter` | 空文字の場合あり |
| 話タイトル | `toc.yaml → subtitles[].subtitle` | |
| 話番号 | `toc.yaml → subtitles[].index` | サイトにより形式が異なる |
| 初回投稿日 | `toc.yaml → subtitles[].subdate` | |
| 最終更新日 | `toc.yaml → subtitles[].subupdate` | |
| 前書き | `本文/*.yaml → element.introduction` | HTML。空の場合あり |
| 本文 | `本文/*.yaml → element.body` | HTML |
| 後書き | `本文/*.yaml → element.postscript` | HTML。空の場合あり |
| 挿絵画像 | `挿絵/i{id}.jpg` | JPEG のみ |
| サイト名 | `database.yaml → sitename` | |
| 完結フラグ | `database.yaml → end` | |
| 総文字数 | `database.yaml → length` | |
| 総話数 | `database.yaml → general_all_no` | |
| タグ | `database.yaml → tags` | |
| 連載/短編 | `database.yaml → novel_type` | 1=連載, 2=短編 |

### 条件付きで利用可能な情報（△）

| 情報 | ファイル | 備考 |
|------|---------|------|
| カスタム置換ルール | `replace.txt` | TSV 形式。MVP では無視可 |
| 作品固有変換設定 | `setting.ini` | INI 形式。横書き設定のみ MVP 対応を検討 |
| カスタム変換スクリプト | `converter.rb` | Ruby 固有。narou-go では無視 |

### narou-go では不要な情報（×）

| 情報 | 理由 |
|------|------|
| `raw/*.html` | 生 HTML。`本文/*.yaml` で代替可 |
| `本文/cache/` | 差分比較キャッシュ。更新検出のみに使用 |
| `[著者名] タイトル.epub` | 再生成対象 |
| `[著者名] タイトル.mobi` | 対象外 |
| `[著者名] タイトル.txt` | 中間ファイル。再生成対象 |
| `調査ログ.txt` | narou.rb の変換ログ |
| `.narou/lock.yaml` | narou.rb のプロセスロック |
| `.narou/freeze.yaml` | 更新停止リスト（narou.rb 用） |
| `.narou/latest_convert.yaml` | 最終変換日時（narou.rb 用） |
| `*.yaml.backup` | バックアップファイル |

---

## サイト別の差異

narou-go が library を読み込む際に考慮すべきサイト別の差異。

### 小説家になろう（ncode.syosetu.com）

- ディレクトリ名: `{ncode} {title}`（例: `n1231id タイトル`）
- `toc_url`: `https://ncode.syosetu.com/{ncode}/`
- `index`: 連番整数の文字列（`'1'`, `'2'`, ...）
- `subdate`/`subupdate`: `YYYY/MM/DD HH:MM` 形式

### カクヨム（kakuyomu.jp）

- ディレクトリ名: `{numeric_id} {title}`（例: `16817139557937843947 タイトル`）
- `toc_url`: `https://kakuyomu.jp/works/{numeric_id}`
- `index`: エピソード固有の長い数値 ID（`'16817330650333494286'`）
- `subdate`/`subupdate`: ISO 8601 形式（`2022-12-10T09:07:16Z`）

---

## ディレクトリパスの解決方法

`database.yaml` の `file_title` フィールドからディレクトリパスを構築する。

```
{library_root}/小説データ/{sitename}/{file_title}/
```

例:
- `sitename = "小説家になろう"`, `file_title = "n1231id タイトル"`
  → `library/小説データ/小説家になろう/n1231id タイトル/`
- `sitename = "カクヨム"`, `file_title = "16817139557937843947 タイトル"`
  → `library/小説データ/カクヨム/16817139557937843947 タイトル/`

---

## 互換性方針

### 読み込み

narou-go は `library/` を**読み込み専用**で扱う。既存の YAML ファイルを変更しない。

### 書き込み（将来の検討事項）

narou-go が更新機能を実装する場合は、narou.rb と同じ形式で YAML を書き込む。これにより narou.rb との併用が可能になる。

MVP では読み込みのみ実装し、更新機能は対象外とする。

---

## 移行手順（ユーザー向け）

narou.rb から narou-go への移行は**不要**。既存の `library/` をそのまま narou-go の `--library` オプションで指定するだけでよい。

```bash
narou-go convert n1231id --library /path/to/library
```

または環境変数・設定ファイルで `library` パスを設定する。

---

## リスクと制約

| リスク | 対応 |
|--------|------|
| `本文/` ディレクトリが空 | 変換スキップ（エラー表示） |
| `toc.yaml` が存在しない | エラー表示 |
| `element.body` が空 | 空ページとして出力 |
| 画像 URL がローカルに存在しない | 画像なしで出力（警告表示） |
| macOS の `._*` ファイル混入 | `._` プレフィックスのファイルを無視 |
| カクヨムの長い数値 index | ファイル名解決時に文字列として扱う |
