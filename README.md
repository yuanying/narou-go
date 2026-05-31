# narou-go

narou-go は、Web 小説をダウンロードして EPUB3 / Kindle 形式に変換するためのコマンドラインツールです。

現在は次の使い方に対応しています。

- 既存の narou.rb `library/` を読み込んで EPUB3 を作る
- 小説家になろう、カクヨムから作品をダウンロードする
- ダウンロード済み作品を更新する
- EPUB3 から Kindle 用の `.mobi` を作る

## 必要なもの

- Go 1.24 以上
- Kindle 形式を作る場合は `aphrael`

Kindle 形式を使う場合は、先に `aphrael` をインストールしてください。

```bash
uv tool install git+https://github.com/yuanying/aphrael.git
```

`aphrael` コマンドが見つかることを確認します。

```bash
aphrael --help
```

## すぐ試す

リポジトリ内で `go run` すると、ビルドせずにそのまま実行できます。

```bash
go run ./cmd/narou-go list --library ./library
```

以降の例では、コマンド名を `narou-go` と書きます。手元でまだインストールしていない場合は、`narou-go` の代わりに `go run ./cmd/narou-go` を使ってください。

## インストール

手元の環境に `narou-go` コマンドとして入れる場合は、リポジトリ内で次を実行します。

```bash
go install ./cmd/narou-go
```

インストール後、narou.rb の library root で確認します。

```bash
cd /path/to/narou-library
narou-go list
```

## 既存の narou.rb library から EPUB を作る

既存の narou.rb の library root でそのまま実行できます。

```bash
narou-go list
```

作品を EPUB に変換します。

```bash
narou-go convert n9669bk --output book.epub
```

`--output` を省略すると、narou.rb と同じ `[作者名] 作品タイトル.epub` が小説保存フォルダに作られます。

## Web からダウンロードする

小説家になろうは N コードまたは URL を指定できます。

```bash
narou-go download n9669bk
narou-go download https://ncode.syosetu.com/n9669bk/
```

カクヨムは作品 URL を指定します。

```bash
narou-go download https://kakuyomu.jp/works/16817330668905575239
```

ダウンロード結果は narou.rb 互換の library 形式で、カレントディレクトリに保存されます。
narou.rb の library 内で実行すると、そのまま `narou list` でも表示できます。

```text
./
├── .narou/
│   └── database.yaml
└── 小説データ/
    ├── 小説家になろう/
    │   └── n9669bk 作品タイトル/
    │       ├── toc.yaml
    │       └── 本文/
    └── カクヨム/
        └── 16817330668905575239 作品タイトル/
            ├── toc.yaml
            └── 本文/
```

保存先を明示する場合は `--library` を指定します。

```bash
narou-go download --library ./my-library n9669bk
```

## ダウンロードと同時に EPUB を作る

`--epub` を付けると、保存後に EPUB も作ります。

```bash
narou-go download --epub n9669bk
```

出力例:

```text
小説データ/小説家になろう/n9669bk 作品タイトル/[作者名] 作品タイトル.epub
```

## Kindle 形式を作る

`--kindle` を付けると、EPUB を作ったあとに `aphrael` を呼び出して `.mobi` を作ります。

```bash
narou-go download --kindle n9669bk
```

出力例:

```text
小説データ/小説家になろう/n9669bk 作品タイトル/[作者名] 作品タイトル.epub
小説データ/小説家になろう/n9669bk 作品タイトル/[作者名] 作品タイトル.mobi
```

既存の narou.rb `library/` から変換するときも使えます。

```bash
narou-go convert n9669bk --library ./library --output book.epub --kindle
```

この場合は次のように出力されます。

```text
library/小説データ/小説家になろう/n9669bk 作品タイトル/book.epub
library/小説データ/小説家になろう/n9669bk 作品タイトル/book.mobi
```

## 更新する

一度ダウンロードした作品は `update` で更新できます。

```bash
narou-go update n9669bk
narou-go update kakuyomu-16817330668905575239
```

更新時にも EPUB / Kindle 出力を指定できます。

```bash
narou-go update --epub n9669bk
narou-go update --kindle kakuyomu-16817330668905575239
```

保存先を明示している場合は、更新時も同じ `--library` を指定してください。

```bash
narou-go update --library ./my-library --kindle n9669bk
```

## 対応サイト

| サイト | 入力例 |
| --- | --- |
| 小説家になろう | `n9669bk` |
| 小説家になろう | `https://ncode.syosetu.com/n9669bk/` |
| カクヨム | `https://kakuyomu.jp/works/16817330668905575239` |

カクヨム作品は、保存時の ID が `kakuyomu-{作品ID}` になります。

## コマンド一覧

```bash
narou-go list [--library path]
narou-go convert ncode [--library path] [--output path] [--kindle]
narou-go download [--library path] [--epub] [--kindle] target
narou-go update [--library path] [--epub] [--kindle] id
```

## 開発者向け

テスト:

```bash
go test ./...
```

Lint:

```bash
go tool golangci-lint run ./...
```

詳しい設計や調査メモは `docs/` を参照してください。
