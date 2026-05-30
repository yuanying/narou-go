# narou.rb 処理パイプライン

## 重要前提

**`library/` に保存されているテキストは HTML 形式**であり、青空文庫形式ではない。

`本文/*.yaml` の `element.data_type` は常に `"html"` であり、`body`/`introduction`/`postscript` は syosetu.com / kakuyomu.jp から取得した HTML がそのまま保存されている。

青空文庫形式（`｜text《ruby》`、`［＃傍点］` 等）は、**narou.rb が AozoraEpub3 との通信に使う中間表現**に過ぎない。narou-go では青空文庫形式への変換は不要。

---

## 担当クラス一覧

| クラス | ファイル | 役割 |
|--------|----------|------|
| `Downloader` | `lib/downloader.rb` | 小説取得・更新確認・差分取得（1,349 行） |
| `NovelConverter` | `lib/novelconverter.rb` | 変換オーケストレーション（774 行） |
| `ConverterBase` | `lib/converterbase.rb` | テキスト整形・青空文庫中間形式生成（1,446 行） |
| `HTML` | `lib/html.rb` | HTML→青空文庫中間形式変換 |
| `Illustration` | `lib/illustration.rb` | 挿絵のダウンロードとローカル管理 |
| `NovelSetting` | `lib/novelsetting.rb` | 作品固有設定の管理 |
| `Database` | `lib/database.rb` | `database.yaml` の読み書き |

---

## 取得パイプライン（Download）

### 担当: `Downloader`

```
narou download {ncode}
    ↓
Downloader.new(target)
    ↓
SiteSetting.find(url)         # webnovel/*.yaml からサイト設定を選択
    ↓
get_latest_table_of_contents  # TOC HTML を取得・パース
    ↓
get_subtitles(toc_source)     # 話一覧を抽出
    ↓
update_body_check(old, new)   # 更新済み話を検出
    ↓
sections_download_and_save    # 各話をダウンロード
│   ↓
│   a_section_download(subtitle)
│   ├── download_raw_data     # HTTP GET で HTML 取得
│   ├── save_raw_data         # raw/ へ保存（任意）
│   └── save_novel_data       # 本文/*.yaml へ保存（HTML のまま）
    ↓
update_database               # .narou/database.yaml を更新
```

### 保存形式

各話は `本文/{index} {file_subtitle}.yaml` に以下の構造で保存される：

```yaml
index: '1'
href: "/n1231id/1/"
chapter: ''
subtitle: タイトル
file_subtitle: タイトル
subdate: 2023/03/19 05:00
subupdate: 2023/03/22 12:26
element:
  data_type: html       # 常に "html"
  introduction: '<p>...</p>'
  body: '<p id="L1">...</p>'
  postscript: ''
```

### 更新確認の仕組み

`update_body_check` は以下の順で変更を検出する：

1. `subdate`/`subupdate` タイムスタンプを比較
2. 話タイトルの変更を比較
3. `strong_update` オプション時はコンテンツ全体を比較

---

## 変換パイプライン（Convert）

### narou.rb の変換フロー（参考）

```
本文/*.yaml (HTML)
    ↓ html.to_aozora()         [HTML クラス]
プレーンテキスト + 青空文庫マーカー
    ↓ converter.convert()      [ConverterBase クラス]
青空文庫形式テキストファイル (.txt)
    ↓ java AozoraEpub3 ...
EPUB ファイル
```

### narou-go の変換フロー

```
本文/*.yaml (HTML)
    ↓ HTML パーサー
EPUB3 XHTML（<ruby>、CSS class 等）
    ↓ EPUB ビルダー
EPUB3 ファイル
```

青空文庫形式への変換は不要。

---

## HTML 変換規則

`lib/html.rb` の `to_aozora()` が行う変換の根拠として、library に保存された HTML の主要パターンを示す。

### narou.rb の変換（参考）

| HTML（library に保存） | 青空文庫中間形式（narou.rb） |
|----------------------|-----------------------------|
| `<ruby>base<rt>ruby</rt></ruby>` | `｜base《ruby》` |
| `<em class="emphasisDots">text</em>` | `［＃傍点］text［＃傍点終わり］` |
| `<b>text</b>` | `［＃太字］text［＃太字終わり］` |
| `<i>text</i>` | `［＃斜体］text［＃斜体終わり］` |
| `<s>text</s>` | `［＃取消線］text［＃取消線終わり］` |
| `<img src="url">` | `［＃挿絵（url）入る］` |
| `<br />` | `\n` |
| `</p>` | `\n` |

### narou-go の変換（直接 XHTML へ）

| HTML（library に保存） | EPUB3 XHTML（narou-go） |
|----------------------|------------------------|
| `<ruby>base<rt>ruby</rt></ruby>` | そのまま使用可 |
| `<em class="emphasisDots">text</em>` | CSS class で傍点スタイル |
| `<b>text</b>` | `<strong>` または CSS |
| `<i>text</i>` | `<em>` または CSS |
| `<s>text</s>` | CSS `text-decoration: line-through` |
| `<img src="url">` | ローカル画像参照に解決 |
| `<br />` | 保持 |
| `<p id="L{n}">` | `<p>` タグへ変換（id 除去） |

---

## 縦書き処理

`lib/converterbase.rb` が行うテキスト変換は、**HTML タグを除去した後のプレーンテキストに対して**パターン検出を行う。検出ロジック自体は青空文庫形式に依存しない。

narou-go では同じパターン検出を HTML テキストノードに対して行い、Aozora マーカーではなく EPUB3 の HTML/CSS で出力する。

### 縦中横（tate-chu-yoko）

`convert_tatechuyoko`（`lib/converterbase.rb:384`）が担当。

検出対象：
- `！！`、`！？`、`！！？` などの感嘆符・疑問符の連続
- 2桁のアラビア数字（`hankaku_num_to_zenkaku_num` から `tcy()` で処理）

| narou.rb 出力 | narou-go 出力 |
|--------------|--------------|
| `［＃縦中横］!!?［＃縦中横終わり］` | `<span class="tcy">!!?</span>` |
| `［＃縦中横］12［＃縦中横終わり］` | `<span class="tcy">12</span>` |

CSS: `span.tcy { text-combine-upright: all; }`

### 傍点（sesame marks）

`is_sesame?` / `sesame()`（`lib/converterbase.rb:905`）が担当。`「` の後にルビ区切り文字 `《》` が来る場合に傍点と判定する。

HTML の `<em class="emphasisDots">` はすでに傍点を示しているため、EPUB3 では CSS で直接スタイルを当てる。

### 行頭かぎ括弧の二分アキ

`half_indent_bracket`（`lib/converterbase.rb:600`）が担当。行頭の `「『(（【〈《` に二分アキを追加する。Kindle Paperwhite でのインデント崩れ対策。

| narou.rb 出力 | narou-go 出力 |
|--------------|--------------|
| `［＃二分アキ］「` | `<p class="half-indent">「...` |

### 自動字下げ（auto_indent）

`auto_indent`（`lib/converterbase.rb:615`）が担当。行頭が会話以外の場合に全角スペースで字下げする。

---

## 挿絵処理

`lib/illustration.rb` の `Illustration` クラスが担当。

1. `element.body` 中の `<img src="url">` を検出
2. URL からダウンロードしてローカルに保存（`挿絵/i{id}.jpg`）
3. 変換時にローカルパスで参照

**narou-go での対応:** `挿絵/` ディレクトリに既にダウンロード済みの画像が存在するため、ダウンロードは不要。`<img src="url">` の URL をファイル名に解決して EPUB に埋め込む。

narou.rb の URL→ファイル名解決規則（`lib/illustration.rb:19`）:

```
NAROU_ILLUST_URL = "https://%s.mitemin.net/userpageimage/viewimage/icode/%s/"
NAROU_ILLUST_TAG_PATTERN = /[ 　\t]*?<(i[0-9]+)\|([0-9]+)>\n?/m
```

`<img>` タグの URL または narou 固有タグ `<i{id1}|{id2}>` から `i{id1}.jpg` へ解決する。

---

## 設定の優先順位

`lib/novelsetting.rb` が管理する変換設定の優先順位（高い順）：

1. `force.*` 設定（一時的な強制上書き）
2. `setting.ini`（作品固有設定）
3. `default.*` 設定
4. narou.rb ハードコードデフォルト

`local_setting.yaml` の `default.enable_half_indent_bracket: true` は library 内の実データより確認。
