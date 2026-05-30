# AozoraEpub3 利用調査

## 概要

narou.rb は青空文庫形式のテキストファイルを EPUB3 に変換するために Java 製の外部ツール [AozoraEpub3](https://github.com/hmdev/AozoraEpub3) を利用している。narou-go では AozoraEpub3 を不要とし、EPUB3 を直接生成する。

本ドキュメントは、narou-go が置き換えるべき処理を把握するために AozoraEpub3 の利用実態を記録する。

---

## 呼び出し方法

根拠: `lib/novelconverter.rb` の `txt_to_epub` メソッド（行 129〜241）

```bash
java \
  -Dfile.encoding=UTF-8 \
  -Dstdout.encoding=UTF-8 \
  -Dstderr.encoding=UTF-8 \
  -Dsun.stdout.encoding=UTF-8 \
  -Dsun.stderr.encoding=UTF-8 \
  -cp AozoraEpub3.jar \
  AozoraEpub3 \
  -enc UTF-8 \
  -of \
  [-device kindle] \
  [-c 0] \
  [-dst "出力ディレクトリ"] \
  [-hor] \
  "入力ファイル.txt"
```

### オプション詳細

| オプション | 条件 | 説明 |
|-----------|------|------|
| `-enc UTF-8` | 常時 | 入力ファイルのエンコーディング |
| `-of` | 常時 | 出力形式（EPUB） |
| `-device kindle` | Kindle デバイス設定時 | Kindle 最適化出力 |
| `-c 0` | 表紙画像が存在する場合 | 先頭挿絵を表紙として使用 |
| `-dst "path"` | 出力先指定時 | 出力ディレクトリ |
| `-hor` | `enable_yokogaki = true` の場合 | 横書きレイアウト |

AozoraEpub3 のカレントディレクトリは `AozoraEpub3.jar` の親ディレクトリに変更してから実行する（AozoraEpub3 がカレントディレクトリから設定ファイルを読み込むため）。

### 実行結果の判定

AozoraEpub3 はエラー時も終了コード 0 を返す。成功・失敗の判定は標準出力で行う：

- 成功: 標準出力に `変換完了` の文字列が含まれる
- エラー: `[ERROR]` または `エラーが発生しました :` で始まる行が含まれる

---

## 入力ファイル

### 青空文庫形式テキストファイル（*.txt）

AozoraEpub3 への主入力。narou.rb が `lib/converterbase.rb` で生成する。

構造（`template/novel.txt.erb` より）：

```
タイトル
著者名
［＃挿絵（cover.jpg）入る］   ← 表紙画像（存在する場合）
-------

あらすじ

URL: https://...
-------

［＃改ページ］
第1章

話タイトル

前書き本文...

本文...

後書き...
```

### 設定ファイル（AozoraEpub3.ini）

根拠: `narou.rb/preset/AozoraEpub3.ini`

```ini
Vertical=1          ; 縦書き
TitlePage=2         ; タイトルページスタイル
CoverPage=1         ; 表紙ページ
TocPage=1           ; 目次ページ
DakutenType=2       ; 濁点フォント処理
AutoYoko=1          ; 縦中横の自動検出
Gaiji32=1           ; 32ドット外字
ChukiRuby=0         ; 注記タグへの自動ルビ無効
```

### カスタム注記タグ（custom_chuki_tag.txt）

根拠: `narou.rb/preset/custom_chuki_tag.txt`

narou.rb 固有の拡張注記タグ。AozoraEpub3 が CSS クラスに変換する：

| 注記タグ | 変換先 CSS クラス | 用途 |
|---------|-----------------|------|
| `［＃二分アキ］` | `half_em_space` | 行頭半角スペース |
| `［＃一字下げ］` | `pt1` | 1字字下げ |
| `［＃二字下げ］` | `pt2` | 2字字下げ |
| `［＃三字下げ］` | `pt3` | 3字字下げ |
| `［＃前書き］` | `introduction` | 前書きブロック |
| `［＃後書き］` | `postscript` | 後書きブロック |
| `［＃濁点］` | `dakuten` | 濁点フォント |
| `［＃zws］` | `&#8203;` | ゼロ幅スペース（Kindle 単語選択用） |

---

## 利用している機能

根拠: narou.rb のソースコードと実際の変換処理から確認。

| 機能 | 使用 | 詳細 |
|------|------|------|
| ルビ | ○ | `｜text《ruby》` 形式 |
| 傍点 | ○ | `［＃傍点］text［＃傍点終わり］` |
| 太字 | ○ | `［＃太字］text［＃太字終わり］` |
| 斜体 | ○ | `［＃斜体］text［＃斜体終わり］` |
| 取消線 | ○ | `［＃取消線］text［＃取消線終わり］` |
| 縦中横 | ○ | `［＃縦中横］!!?［＃縦中横終わり］` |
| 表紙画像 | ○ | `-c 0` オプション |
| 挿絵 | ○ | `［＃挿絵（path）入る］` |
| 目次 | ○ | 自動生成 |
| 縦書き CSS | ○ | デフォルト（`-hor` なし） |
| 横書き | △ | `enable_yokogaki` 設定時のみ |
| Kindle 最適化 | ○ | `-device kindle` |
| 二分アキ | ○ | カスタム注記タグ |
| ゼロ幅スペース | ○ | Kindle 単語選択用 |

---

## 利用していない機能

narou-go の実装スコープから除外できる機能：

| 機能 | 理由 |
|------|------|
| 外字（Gaiji）変換 | web 小説では使用されない |
| OCR | スキャン画像からの文字認識。不要 |
| PDF 出力 | 対象外 |
| 画像のみ EPUB | 対象外 |
| 自動ルビ（未読み漢字への付与） | `ChukiRuby=0` で無効化済み |
| ローマ字ルビ | 使用されていない |
| 縦書き以外のデフォルト | 横書きは `enable_yokogaki` でオプション |

---

## narou-go での置き換え方針

AozoraEpub3 が担っていた処理を narou-go で直接実装する：

1. **HTML → EPUB3 XHTML 変換**
   - library の HTML を直接 XHTML に変換（青空文庫中間形式を経由しない）
   - `<ruby>` タグはそのまま利用
   - `<em class="emphasisDots">` は CSS クラスで傍点スタイル

2. **縦中横処理**
   - テキストノード内の `！！`、`！？`、2桁数字を検出
   - `<span class="tcy">` でラップ
   - CSS: `text-combine-upright: all`

3. **EPUB3 構造生成**
   - `package.opf`（OPF メタデータ）
   - `nav.xhtml`（ナビゲーション・目次）
   - コンテンツ XHTML ファイル群
   - 縦書き CSS（`writing-mode: vertical-rl`）

4. **画像の EPUB 内配置**
   - `挿絵/*.jpg` を EPUB ZIP 内に配置
   - 表紙画像の設定

---

## 参考: EPUB3 縦書き CSS

AozoraEpub3 が生成する縦書きスタイルの代替として、narou-go で実装すべき最低限の CSS：

```css
body {
  writing-mode: vertical-rl;
  -epub-writing-mode: vertical-rl;
}

span.tcy {
  text-combine-upright: all;
  -webkit-text-combine: horizontal;
}

em.emphasisDots {
  text-emphasis: sesame;
  -webkit-text-emphasis: sesame;
}

p.half-indent::first-letter {
  margin-inline-start: 0.5em;
}
```
