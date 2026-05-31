# narou.rb library 書き込み仕様

## 目的

`narou-go download` の保存先を Go 独自形式の `data/{id}/novel.yaml` から、narou.rb 互換の library root へ移行する。

これにより、`download` した作品がそのまま `narou-go list` と `narou list` の両方に表示され、`narou-go convert` の入力としても使えるようにする。

## 背景

現在の `download` は `internal/storage` を使い、次の形式で保存している。

```text
data/
└── {id}/
    ├── novel.yaml
    └── images/
```

一方、現在の `list` と `convert` は `--library` で指定された narou.rb の library root を読む実装であり、`{library}/.narou/database.yaml`、`toc.yaml`、`本文/*.yaml` を前提にしている。そのため、`download` 済み作品が `list` に表示されない。

今後は `download` の標準保存先を narou.rb 互換 library root に統一し、`data/` 独自形式は廃止対象とする。

## narou.rb 実装から確認した前提

この仕様は `./narou.rb` の実装を参照して決める。

| 確認対象 | narou.rb 側の実装 |
| --- | --- |
| database 保存 | `narou.rb/lib/database.rb`、`narou.rb/lib/inventory.rb` |
| library root 判定 | `Narou.root_dir`。カレントから親へ `.narou/` を探す |
| 初期化 | `Narou.init`。カレントに `.narou/` と `小説データ/` を作る |
| database 更新項目 | `narou.rb/lib/downloader.rb` の `update_database` |
| ID 採番 | `Database#create_new_id`。既存キー最大値 + 1 |
| 小説ディレクトリ | `Downloader#get_novel_data_dir`。`小説データ/{sitename}/{file_title}` |
| 本文ファイル名 | `Downloader#section_file_path`。`本文/{index} {file_subtitle}.yaml` |
| toc/本文保存 | `Downloader#save_novel_data`。`YAML.dump` で保存 |
| ファイル名変換 | `Helper.replace_filename_special_chars`、`truncate_path`、`truncate_folder_title` |
| YAML 書き込み安全性 | `narou.rb/lib/extension.rb` の `File.write` monkey patch |

## スコープ

### 対象

- `narou-go download` が narou.rb 互換 library root に新規作品を書き込む
- `narou-go update` が同じ library root 内の既存作品を更新する
- `narou-go list` がダウンロード済み作品を表示できる
- `narou list` が narou-go でダウンロードした作品を表示できる
- `narou-go convert` がダウンロード済み作品を EPUB 化できる
- 小説家になろう、カクヨムの両方に対応する

### 対象外

- narou.rb の全メタデータ完全再現
- narou.rb の `setting.ini`、`replace.txt`、`converter.rb` の生成
- `raw/`、`本文/cache/` の生成
- narou.rb 本体からの更新実行との完全な相互運用保証
- 既存 `data/` 形式からの自動移行

## CLI 仕様

### library root 解決

`narou-go` は `--library` が省略された場合、narou.rb と同じくカレントディレクトリを library root として扱う。

ただし、カレントまたは親ディレクトリに `.narou/` が存在する場合は、最初に見つかった `.narou/` の親ディレクトリを library root とする。これは `narou.rb` の `Narou.root_dir` と同じ挙動であり、library root のサブディレクトリから実行しても同じ library を参照できるようにするため。

解決順:

1. `--library path` が指定されていれば、その path を library root とする
2. カレントから親方向に `.narou/` を探し、見つかればその親を library root とする
3. 見つからなければ、カレントディレクトリを library root とする

この仕様により、通常利用は narou.rb と同じく library ディレクトリ内で実行する。

```bash
cd /path/to/narou-library
narou-go list
narou-go download n9669bk
narou list
```

`narou-go download` は、library root に `.narou/` がなければ `.narou/` と `小説データ/` を作成する。これにより、そのディレクトリは narou.rb からも library root として認識される。

`narou list` 互換を満たすため、`database.yaml` には narou.rb の list 表示で必要な日時フィールドを必ず書く。少なくとも `last_update`、`new_arrivals_date`、`general_lastup` は Ruby/Psych が `Time` として読める YAML timestamp scalar にする。空文字にはしない。

### download

```bash
narou-go download [--library path] [--epub] [--kindle] target
```

- `--library` 省略時は library root 解決に従う
- 新規作品を library root に保存する
- 保存後、対象作品は `narou-go list` と `narou list` に表示される
- `--epub` 指定時は保存した library データを使って EPUB を生成する

### update

```bash
narou-go update [--library path] [--epub] [--kindle] id_or_url
```

- `--library` 省略時は library root 解決に従う
- `id_or_url` は N コード、カクヨム work ID、または URL を受け付ける
- `database.yaml` から既存作品を探し、同じ小説ディレクトリを更新する
- 見つからない場合はエラーにする

### 互換オプション

既存の `--data` は廃止予定とする。

移行期間中は次のどちらかを選ぶ。

- 推奨: `--data` を指定されたらエラーにし、`--library` への変更を促す
- 互換重視: `--data` を受け付けるが deprecated warning を出し、旧形式保存だけを行う

最終的には `internal/storage` と `data/` 保存仕様を削除する。

## 書き込むディレクトリ構造

```text
{library_root}/
├── .narou/
│   └── database.yaml
└── 小説データ/
    ├── 小説家になろう/
    │   └── n9669bk 作品タイトル/
    │       ├── toc.yaml
    │       ├── 本文/
    │       │   ├── 1 第一話.yaml
    │       │   └── 2 第二話.yaml
    │       └── 挿絵/
    │           └── i514881.jpg
    └── カクヨム/
        └── 16817330668905575239 作品タイトル/
            ├── toc.yaml
            ├── 本文/
            └── 挿絵/
```

`挿絵/` は画像がある場合のみ作成する。

## database.yaml

### 保存場所

```text
{library_root}/.narou/database.yaml
```

存在しない場合は `{library_root}/.narou/` を作成し、空の database から開始する。

### キーと ID

narou.rb 形式ではトップレベルキーが整数で、各 entry の `id` と一致する。

新規追加時は、既存キーの最大値 + 1 を採番する。空の場合は `0` から始める。

```yaml
---
0:
  id: 0
  author: 作者名
  title: 作品タイトル
  file_title: n9669bk 作品タイトル
  toc_url: https://ncode.syosetu.com/n9669bk/
  sitename: 小説家になろう
  novel_type: 1
  end: false
  last_update: 2026-05-31 12:00:00.000000000 +00:00
  new_arrivals_date: 2026-05-31 12:00:00.000000000 +00:00
  use_subdirectory: false
  general_firstup: 2020-01-01 00:00:00.000000000 +00:00
  novelupdated_at: 2020-01-02 12:00:00.000000000 +00:00
  general_lastup: 2020-01-02 12:00:00.000000000 +00:00
  length: 0
  suspend: false
  general_all_no: 2
  last_check_date: 2026-05-31 12:00:00.000000000 +00:00
  tags: []
```

### 必須フィールド

narou-go が読み込みに使うフィールドは必ず書く。

| フィールド | 値 |
| --- | --- |
| `id` | database key と同じ整数 |
| `author` | `model.Novel.Author` |
| `title` | `model.Novel.Title` |
| `file_title` | 小説ディレクトリ名 |
| `toc_url` | 作品目次 URL |
| `sitename` | `小説家になろう` または `カクヨム` |
| `novel_type` | `1` 連載、`2` 短編 |
| `end` | 完結フラグ。不明なら `false` |
| `tags` | 不明なら空配列。完結時は `end` を含める |

### 最小互換フィールド

narou.rb の `Downloader#update_database` が書く項目に合わせ、以下も書く。ただし値は最小でよい。

| フィールド | 値 |
| --- | --- |
| `last_update` | 保存時刻 |
| `new_arrivals_date` | 保存時刻 |
| `use_subdirectory` | `false` |
| `general_firstup` | 先頭話の投稿日。不明なら保存時刻 |
| `novelupdated_at` | 作品更新日時。不明なら最終話の更新日時、さらに不明なら保存時刻 |
| `general_lastup` | 最終話の投稿日または更新日時。不明なら保存時刻 |
| `length` | 未計算なら `0` |
| `suspend` | `false` |

以下は narou.rb の通常 download では `update_database` 直後に必ず入るとは限らないが、narou.rb の list/web/update 周辺が参照するため、narou-go では書く。

| フィールド | 値 |
| --- | --- |
| `general_all_no` | 話数 |
| `last_check_date` | 保存時刻 |

日時フィールドは YAML timestamp scalar として出力し、引用符付き文字列にしない。`narou list` は `last_update` や `general_lastup` に対して `strftime` や時刻比較を行うため、文字列や空文字ではなく Ruby の `Time` として読める値である必要がある。

## 小説ディレクトリ

### パス

```text
{library_root}/小説データ/{sitename}/{file_title}/
```

### sitename

| サイト | 値 |
| --- | --- |
| 小説家になろう | `小説家になろう` |
| カクヨム | `カクヨム` |

### file_title

| サイト | 形式 |
| --- | --- |
| 小説家になろう | `{ncode} {title}` |
| カクヨム | `{work_id} {title}` |

カクヨムの `model.Novel.ID` は `kakuyomu-{work_id}` なので、`file_title` では `kakuyomu-` prefix を外す。

narou.rb は既存 database entry に `file_title` がある場合、それを維持する。narou-go の `update` も同じく、既存 entry の `file_title` を原則維持する。タイトル変更に追従したディレクトリリネームは初期実装では行わない。

### ファイル名安全化

`title` と `file_subtitle` はファイル名に使うため、narou.rb の `Helper.replace_filename_special_chars` に合わせて置換する。

| 文字 | 置換 |
| --- | --- |
| `/` | `／` |
| `:` | `：` |
| `*` | `＊` |
| `?` | `？` |
| `"` | `”` |
| `<` | `〈` |
| `>` | `〉` |
| `[` | `［` |
| `]` | `］` |
| `{` | `｛` |
| `}` | `｝` |
| `|` | `｜` |
| `.` | `．` |
| `` ` `` | `｀` |
| `\` | `￥` |
| タブ、改行 | 削除 |

`file_title` は narou.rb の `truncate_folder_title` に合わせ、50 文字を超える場合は先頭 50 文字に切り詰め、前後の空白を削除する。

`file_subtitle` は narou.rb の `title_to_filename` に合わせ、ルビタグ相当を除去できる場合は除去したうえで、50 文字を超える場合は先頭 50 文字に切り詰める。

同じ `file_title` が既に存在し、別作品を指す場合は `file_title` 末尾に ` ({id})` を付けて衝突回避する。

## toc.yaml

### 保存場所

```text
{novel_dir}/toc.yaml
```

### 形式

```yaml
---
title: 作品タイトル
author: 作者名
toc_url: https://ncode.syosetu.com/n9669bk/
story: |-
  あらすじ
subtitles:
- index: "1"
  href: /n9669bk/1/
  chapter: 第一章
  subchapter: ""
  subtitle: 第一話
  file_subtitle: 第一話
  subdate: 2020/01/01 00:00
  subupdate: 2020/01/02 12:00
  download_time: 2026-05-31 12:00:00.000000000 +00:00
```

### フィールド対応

| toc.yaml | model |
| --- | --- |
| `title` | `Novel.Title` |
| `author` | `Novel.Author` |
| `toc_url` | `Novel.SourceURL` |
| `story` | `Novel.Story` |
| `subtitles[].index` | `Episode.ID` |
| `subtitles[].href` | `Episode.URL` からホストを除いた path。難しい場合は絶対 URL |
| `subtitles[].chapter` | `Episode.Chapter` |
| `subtitles[].subchapter` | `Episode.Subchapter` |
| `subtitles[].subtitle` | `Episode.Title` |
| `subtitles[].file_subtitle` | ファイル名安全化した `Episode.Title` |
| `subtitles[].subdate` | `Episode.PublishedAt` |
| `subtitles[].subupdate` | `Episode.UpdatedAt` |
| `subtitles[].download_time` | `Episode.DownloadedAt` または保存時刻 |

短編の場合も `subtitles` は 1 件作る。

narou.rb は `download_time` を本文ダウンロード時に `subtitle_info` へ追加し、その値を `toc.yaml` にも保持する。narou-go でも `toc.yaml` と本文 YAML の両方に同じ `download_time` を書く。

## 本文 YAML

### 保存場所

```text
{novel_dir}/本文/{index} {file_subtitle}.yaml
```

### 形式

```yaml
---
index: "1"
href: /n9669bk/1/
chapter: 第一章
subchapter: ""
subtitle: 第一話
file_subtitle: 第一話
subdate: 2020/01/01 00:00
subupdate: 2020/01/02 12:00
download_time: 2026-05-31 12:00:00.000000000 +00:00
element:
  data_type: html
  introduction: |-
    <p>前書き</p>
  body: |-
    <p id="L1">本文</p>
  postscript: |-
    <p>後書き</p>
```

### フィールド対応

| 本文 YAML | model |
| --- | --- |
| `index` | `Episode.ID` |
| `href` | `Episode.URL` からホストを除いた path。難しい場合は絶対 URL |
| `chapter` | `Episode.Chapter` |
| `subchapter` | `Episode.Subchapter` |
| `subtitle` | `Episode.Title` |
| `file_subtitle` | ファイル名安全化した `Episode.Title` |
| `subdate` | `Episode.PublishedAt` |
| `subupdate` | `Episode.UpdatedAt` |
| `download_time` | `Episode.DownloadedAt` または保存時刻 |
| `element.data_type` | `html` 固定 |
| `element.introduction` | `Episode.Preface` |
| `element.body` | `Episode.Body` |
| `element.postscript` | `Episode.Afterword` |

既存の `convert` はこの形式をそのまま読める。

## 画像

### 保存場所

```text
{novel_dir}/挿絵/
```

### ファイル名

小説家になろうの mitemin 画像は、URL から `icode` を取得できる場合に `i{icode}.jpg` として保存する。

例:

```text
https://12345.mitemin.net/userpageimage/viewimage/icode/514881/
-> 挿絵/i514881.jpg
```

`icode` を取得できない場合は、URL path の basename を使う。拡張子が不明な場合は HTTP response の `Content-Type` から推定する。

### 本文中の src

本文 HTML の `<img src="...">` は、既存の `converter.ImageRegistry` が解決できる形にする。

推奨は mitemin URL を元のまま残し、画像ファイル名を `i{icode}.jpg` にすること。既存実装は mitemin URL から `挿絵/i{icode}.jpg` を探せる。

カクヨムなど mitemin でない画像は、本文中の `src` を `挿絵/{filename}` または `{filename}` に書き換え、対応するファイルを `挿絵/` に置く。

## update 仕様

`update` は既存の小説ディレクトリを上書き更新する。

### 作品検索

次の順で対象を探す。

1. `database.yaml` の `toc_url` が入力 URL と一致する
2. 小説家になろうの場合、`toc_url` または `file_title` が N コードを含む
3. カクヨムの場合、`toc_url` または `file_title` が work ID を含む

### 更新単位

最小実装では、全話を再ダウンロードして `toc.yaml` と `本文/*.yaml` を再生成してよい。

削除された話がある場合に古い YAML が残ると `convert` の対象にはならないが、ユーザーには紛らわしい。更新時は `toc.yaml` に存在しない旧 `本文/*.yaml` を削除する。ただし `cache/` ディレクトリは触らない。

### database 更新

既存 entry の `id` は維持する。

更新するフィールド:

- `author`
- `title`
- `file_title`
- `toc_url`
- `sitename`
- `novel_type`
- `end`
- `last_update`
- `new_arrivals_date`
- `novelupdated_at`
- `general_lastup`
- `general_all_no`
- `last_check_date`

narou.rb と同じく、既存 entry の `file_title` は維持する。タイトル変更があっても、小説ディレクトリの自動リネームは初期実装では行わない。

## 書き込み安全性

### バックアップ

narou.rb の `File.write` monkey patch は、YAML を一時ファイルに書いてから rename し、さらに `本文/` 直下以外の YAML について `.backup` を作る。

narou-go もこれに合わせ、`本文/` 直下以外の YAML について `.backup` を作る。narou.rb の挙動に合わせ、`.backup` は旧内容ではなく新しく書く内容のコピーでよい。

対象:

- `library/.narou/database.yaml`
- `{novel_dir}/toc.yaml`

`本文/*.yaml` の `.backup` 作成は narou.rb と同じく必須にしない。作成しても narou-go の `ListSectionFiles` は `.yaml.backup` を無視できるが、最小互換では作らない。

### アトミック書き込み

YAML は narou.rb と同じく、同一ディレクトリ内のランダム名一時ファイルに書き、`rename` で置き換える。

例:

```text
database.yaml.tmp
-> database.yaml
```

### 失敗時

`database.yaml` 更新前に小説ディレクトリの生成や本文保存が失敗した場合は、database を変更しない。

`database.yaml` 更新後に失敗した場合はエラーを返す。自動ロールバックは必須にしないが、`.backup` から手動復旧できる状態にする。

## 実装方針

### 新規パッケージ

`internal/library` に書き込み機能を追加する。

候補:

```go
func SaveDownloadedNovel(libraryPath string, novel *model.Novel, now time.Time) (*NovelEntry, error)
func UpdateDownloadedNovel(libraryPath string, novel *model.Novel, now time.Time) (*NovelEntry, error)
```

または読み込みと分離する場合:

```text
internal/librarywriter
```

ただし既存の `library.NovelEntry`、`library.TOC`、`library.Section` と同じ構造を使うため、まずは `internal/library` に置くのが自然。

### CLI 変更

- `webOptions.dataPath` を `libraryPath` に置き換える
- `parseWebOptions` は `--library` を読む
- `runDownload` は `storage.SaveNovel` ではなく library writer を呼ぶ
- `runUpdate` は `storage.LoadNovel` ではなく `database.yaml` から既存作品を探して再ダウンロードする
- `download --epub` は保存後に `runConvert` 相当の library 読み込み経路を使う

### 削除候補

library 書き込み移行後、以下は削除候補になる。

- `internal/storage`
- `docs/web-download.md` の `data/{id}/novel.yaml` 記述
- README の `--data` と `data/` ディレクトリ説明

## テスト方針

TDD で以下から追加する。

1. `SaveDownloadedNovel` が空の temp library に `.narou/database.yaml`、`toc.yaml`、`本文/*.yaml` を作る
2. 保存後に `library.LoadDatabase`、`library.LoadTOC`、`library.LoadSection` で読める
3. temp library をカレントにして `narou-go list` を実行すると作品が出る
4. temp library をカレントにして `narou-go convert {ncode}` を実行すると EPUB が作れる
5. temp library をカレントにして `narou list` を実行すると作品が出る
6. `database.yaml` の `last_update`、`new_arrivals_date`、`general_lastup` が narou.rb から `Time` として読める
7. カクヨム作品の `file_title` が `kakuyomu-` prefix なしになる
8. ファイル名に `/` や改行が含まれるタイトルを安全化できる
9. 既存 database に追記したとき ID が最大値 + 1 になる
10. update で既存 ID と `file_title` を維持し、話数増減が `toc.yaml` と `本文/` に反映される

## 移行手順

1. library writer のテストを追加する
2. `model.Novel` から `NovelEntry`、`TOC`、`Section` への変換を実装する
3. `--library` 省略時の library root 解決を実装する
4. `download` の library 書き込みを実装し、`--data` を非推奨にする
5. `update` の library 更新を実装する
6. README と `docs/web-download.md` を更新する
7. `internal/storage` を削除する

## 未決事項

- narou.rb の `update` まで完全互換にするために追加で必要な database/toc 項目があるか
- `last_update` などの日時フォーマットを UTC にするか、ローカルタイムにするか
- カクヨム画像のローカルファイル名をどの規則に固定するか
- `length` を本文 HTML から計算するか、当面 `0` にするか
- `--data` を即削除するか、1 リリースだけ deprecated とするか
