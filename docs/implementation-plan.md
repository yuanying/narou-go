# narou-go 実装計画

## 実装方針

- **Go 1.24+**
- **EPUB3 直接生成**（AozoraEpub3 不要）
- **library 読み込み対応**（narou.rb の `library/` をそのまま使用）
- **青空文庫中間形式を経由しない**（HTML → EPUB3 XHTML の直接変換）

---

## アーキテクチャ

### パッケージ構成

```
narou-go/
├── cmd/
│   └── narou-go/
│       └── main.go              # CLI エントリポイント
├── internal/
│   ├── library/                 # library/ 読み込み
│   │   ├── database.go          # .narou/database.yaml パーサー
│   │   ├── novel.go             # 小説ディレクトリの読み込み
│   │   ├── toc.go               # toc.yaml パーサー
│   │   └── section.go           # 本文/*.yaml パーサー
│   ├── converter/               # HTML → EPUB3 XHTML 変換
│   │   ├── html.go              # HTML パーサー・変換
│   │   ├── typography.go        # 縦書き処理（縦中横、傍点等）
│   │   └── image.go             # 挿絵 URL → ローカルパス解決
│   └── epub/                    # EPUB3 生成
│       ├── builder.go           # EPUB 構造生成のオーケストレーション
│       ├── opf.go               # package.opf 生成
│       ├── nav.go               # nav.xhtml（目次）生成
│       ├── content.go           # コンテンツ XHTML 生成
│       └── css.go               # 縦書き CSS
├── docs/                        # 調査ドキュメント
├── go.mod
└── go.sum
```

---

## 依存ライブラリ

| ライブラリ | 用途 |
|-----------|------|
| `gopkg.in/yaml.v3` | YAML 読み込み（database.yaml, toc.yaml, 本文/*.yaml） |
| `golang.org/x/net/html` | HTML パース（element.body の HTML を XHTML に変換） |
| `archive/zip`（標準） | EPUB（ZIP アーカイブ）生成 |
| `encoding/xml`（標準） | OPF・NAV・XHTML の XML 生成 |

---

## MVP スコープ

MVP では以下を実装する。更新機能（ダウンロード）は対象外。

### Library Reader

**対象ファイル:**
- `internal/library/database.go`
- `internal/library/novel.go`
- `internal/library/toc.go`
- `internal/library/section.go`

**実装内容:**

```go
// database.go
type NovelEntry struct {
    ID         int       `yaml:"id"`
    Author     string    `yaml:"author"`
    Title      string    `yaml:"title"`
    FileTitle  string    `yaml:"file_title"`
    TocURL     string    `yaml:"toc_url"`
    SiteName   string    `yaml:"sitename"`
    NovelType  int       `yaml:"novel_type"`
    End        bool      `yaml:"end"`
    Tags       []string  `yaml:"tags"`
}

func LoadDatabase(libraryPath string) (map[int]NovelEntry, error)
func FindByNcode(db map[int]NovelEntry, ncode string) (*NovelEntry, error)
```

```go
// toc.go
type TOC struct {
    Title     string     `yaml:"title"`
    Author    string     `yaml:"author"`
    TocURL    string     `yaml:"toc_url"`
    Story     string     `yaml:"story"`
    Subtitles []Subtitle `yaml:"subtitles"`
}

type Subtitle struct {
    Index       string `yaml:"index"`
    Href        string `yaml:"href"`
    Chapter     string `yaml:"chapter"`
    Subchapter  string `yaml:"subchapter"`
    Subtitle    string `yaml:"subtitle"`
    FileSubtitle string `yaml:"file_subtitle"`
    Subdate     string `yaml:"subdate"`
    Subupdate   string `yaml:"subupdate"`
}

func LoadTOC(novelDir string) (*TOC, error)
```

```go
// section.go
type Section struct {
    Index        string  `yaml:"index"`
    Chapter      string  `yaml:"chapter"`
    Subtitle     string  `yaml:"subtitle"`
    FileSubtitle string  `yaml:"file_subtitle"`
    Element      Element `yaml:"element"`
}

type Element struct {
    DataType     string `yaml:"data_type"`
    Introduction string `yaml:"introduction"`
    Body         string `yaml:"body"`
    Postscript   string `yaml:"postscript"`
}

func LoadSection(novelDir string, subtitle Subtitle) (*Section, error)
func ListSectionFiles(novelDir string) ([]string, error)
```

**ディレクトリパス解決:**

```go
// novel.go
func NovelDir(libraryPath string, entry NovelEntry) string {
    return filepath.Join(libraryPath, "小説データ", entry.SiteName, entry.FileTitle)
}
```

### HTML Converter

**対象ファイル:** `internal/converter/html.go`, `internal/converter/typography.go`

**実装内容:**

HTML テキストノードに対して以下の処理を行い XHTML を生成する：

1. `<p id="L{n}">` → `<p>`（id 属性除去）
2. `<ruby>base<rt>ruby</rt></ruby>` → そのまま保持
3. `<em class="emphasisDots">text</em>` → CSS で傍点スタイル
4. `<b>` → `<strong>`
5. `<i>` → `<em>`
6. `<s>` → `<span class="strikethrough">`
7. `<img src="url">` → ローカル画像参照に解決
8. `<a href>` → 除去（テキストのみ保持）

**縦書き処理（typography.go）:**

HTML テキストノードのみを対象に以下を適用：

- `！！`、`！？`、`！！？` → `<span class="tcy">!!?</span>`
- 2桁のアラビア数字 → `<span class="tcy">12</span>`
- 行頭 `「『(（【` → `<p class="half-indent">` または字下げ CSS

### EPUB3 Builder

**対象ファイル:** `internal/epub/`

**EPUB3 構造:**

```
novel.epub (ZIP)
├── mimetype                    # "application/epub+zip"
├── META-INF/
│   └── container.xml
└── OEBPS/
    ├── package.opf             # メタデータ・スパイン
    ├── nav.xhtml               # 目次
    ├── style/
    │   └── vertical.css        # 縦書き CSS
    ├── images/
    │   ├── cover.jpg           # 表紙（挿絵/の最初の画像）
    │   └── i{id}.jpg           # 挿絵
    └── text/
        ├── p001.xhtml          # 各話コンテンツ
        ├── p002.xhtml
        └── ...
```

**縦書き CSS（`vertical.css`）:**

```css
body {
  writing-mode: vertical-rl;
  -epub-writing-mode: vertical-rl;
  line-height: 1.6;
}

span.tcy {
  text-combine-upright: all;
  -webkit-text-combine: horizontal;
}

em.emphasisDots {
  text-emphasis: sesame;
  -webkit-text-emphasis: sesame;
  font-style: normal;
}
```

**OPF のメタデータ（`package.opf`）:**

```xml
<metadata>
  <dc:title>{title}</dc:title>
  <dc:creator>{author}</dc:creator>
  <dc:language>ja</dc:language>
  <meta property="rendition:layout">reflowable</meta>
  <meta property="rendition:orientation">auto</meta>
  <meta property="rendition:spread">auto</meta>
</metadata>
```

### Image Support

**対象ファイル:** `internal/converter/image.go`

- `<img src="url">` の URL を `挿絵/` ディレクトリのファイルに解決
- `挿絵/i{id}.jpg` をそのまま EPUB に埋め込み
- 表紙: `挿絵/` 内の最初の画像または `cover.jpg`/`cover.png` を使用
- 対応形式: JPEG, PNG, GIF, WebP

URL→ファイル名解決ロジック（`lib/illustration.rb:19` に基づく）：
- `https://{domain}.mitemin.net/.../icode/{id1}/` → `i{id1}.jpg`
- ローカルパス参照の場合はそのまま使用

### CLI

**対象ファイル:** `cmd/narou-go/main.go`

最低限のコマンドインターフェース：

```
narou-go convert {ncode} [--library /path/to/library] [--output /path/to/output.epub]
narou-go list [--library /path/to/library]
```

`--library` のデフォルト値: `./library`

---

## 実装優先順位

| 優先度 | 機能 | 備考 |
|--------|------|------|
| 1 | Library Reader | YAML 読み込み |
| 2 | HTML Converter | element.body の HTML → XHTML |
| 3 | EPUB Builder | EPUB3 ZIP 生成 |
| 4 | Image Support | 挿絵の埋め込み |
| 5 | CLI | convert コマンド |
| 後回し | 縦書き処理（縦中横等） | 読めるが最適化不足 |
| 後回し | setting.ini 対応 | 横書き等のオプション |
| 対象外 | 更新機能（ダウンロード） | narou.rb で継続 |
| 対象外 | MOBI 生成 | 対象外 |

---

## TDD での進め方

各コンポーネントを以下の順で実装する：

1. テストを書く（期待する入出力を定義）
2. テストが失敗することを確認
3. テストが通る実装を書く
4. リファクタリング

### Library Reader のテスト例

```go
func TestLoadDatabase(t *testing.T) {
    db, err := LoadDatabase("testdata/library")
    require.NoError(t, err)
    assert.Equal(t, "勇者の母ですが、魔王軍の幹部になりました。", db[0].Title)
    assert.Equal(t, "小説家になろう", db[0].SiteName)
}

func TestLoadSection(t *testing.T) {
    section, err := LoadSection("testdata/library/小説データ/小説家になろう/n1231id .../", subtitle)
    require.NoError(t, err)
    assert.Equal(t, "html", section.Element.DataType)
    assert.Contains(t, section.Element.Body, "<p id=")
}
```

---

## 不要な機能

以下は narou-go では実装しない：

- MOBI 生成（kindlegen 依存）
- PDF 生成
- 外字変換
- 青空文庫完全互換
- ダウンロード機能（MVP では）
- Web UI
- メール送信
- device 検出・転送
- カスタム converter.rb の実行（Ruby 固有）
