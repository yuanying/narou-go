# Web ダウンロード調査

## narou.rb の構成

- `lib/downloader.rb` がダウンロードの中心。URL/ID 判定、目次取得、本文取得、更新判定、保存を担当する。
- サイト固有の URL、目次、本文、作品情報の抽出ルールは `webnovel/*.yaml` に分離されている。
- `lib/sitesetting.rb` は URL に対して該当サイト設定を選び、正規表現の named group を設定値へ展開する。
- `lib/novelinfo.rb` は作品情報ページまたは目次ページからタイトル、作者、種別、あらすじ、更新日時などを抽出する。
- `lib/html.rb` は HTML を narou.rb の中間表現へ変換する。Go 版では青空文庫中間形式は使わず、HTML/XHTML へ直接寄せる。
- `lib/illustration.rb` は mitemin 画像 URL とローカル画像の解決・保存を担当する。

## 共通挙動

- `Downloader.get_target_type` は URL、Nコード、既存 ID、タイトル検索を判定する。Nコードは `/^n\d+[a-z]+$/i` で小文字化する。
- User-Agent は `lib/extension.rb` の `make_open_uri_options` で設定する。未設定時は `Mozilla/5.0 (Windows NT 10.0; Win64; x64)`。
- 取得間隔は `download.interval`、デフォルト `0.7` 秒。小説家になろうでは `download.wait-steps` が 0 または 10 超なら 10 に丸め、10 話ごとに長めの wait を挟む。
- ネットワークリトライは `LIMIT_TO_RETRY_NETWORK = 5`、リトライ待機 `10` 秒。503 と 404 は原則中断扱い。
- 目次取得時はリダイレクト先からサイト設定を再判定する。削除・非公開は 404 相当として扱う。
- 更新判定は、目次上のタイトル/章/更新日時、本文ファイル存在、必要に応じて本文差分で行う。
- 本文取得後は `element.data_type = html` とし、`introduction`、`body`、`postscript` を保存する。

## 小説家になろう

- 設定ファイルは `webnovel/ncode.syosetu.com.yaml`。
- URL は `https?://ncode.syosetu.com/(?<ncode>n\d+[a-z]+)`。Nコード入力は `https://ncode.syosetu.com/{ncode}/` に正規化される。
- 作品情報は `https://api.syosetu.com/novelapi/api/` または `https://ncode.syosetu.com/novelview/infotop/ncode/{ncode}/` から取得する。
- 目次 URL は `https://ncode.syosetu.com/{ncode}/`。
- 目次抽出は `p-eplist__chapter-title` と `p-eplist__sublist` を対象に、章、href、index、subtitle、subdate、subupdate を取得する。
- ページングは `c-pager__item--next` と `c-pager__item--last` で検出する。
- 連載/短編は novel type で分岐する。短編は index `1` の単話として subtitle を作る。
- 本文は `div.js-novel-text.p-novel__text`、前書きは `p-novel__text--preface`、後書きは `p-novel__text--afterword` から抽出する。
- mitemin 画像は本文中の `<img>` から URL を取り、`挿絵/i{id}.jpg` 相当へ保存する。

## カクヨム

- 設定ファイルは `webnovel/kakuyomu.jp.yaml`。
- URL は `https://kakuyomu.jp/works/(?<ncode>\d+)`。Go 版の内部 ID は衝突防止のため `kakuyomu-{workID}` とする。
- 作品ページの `<script id="__NEXT_DATA__" type="application/json">` から Apollo state を読む。
- narou.rb は Apollo state から `Work:{workID}`、著者、`tableOfContents`、Chapter、Episode を解決して目次用の中間文字列を作る。
- 作品タイトル、作者、あらすじ、公開日、更新日、文字数、状態も Apollo state から取得する。
- 目次は Chapter level 1/2 と Episode を順に展開する。章がない場合は空文字。
- Episode の `publishedAt` を更新日相当として扱う。
- 本文は `div.widget-episodeBody.js-episode-body` から抽出する。
- カクヨムには明確な前書き/後書きがないため、MVP では `Preface = ""`、`Body = 本文`、`Afterword = ""` とする。
- 本文中画像は `<img>` を検出し、相対 URL を episode URL から絶対化して取得する。

## Go 版での実装方針

- サイトごとに `internal/downloader/syosetu` と `internal/downloader/kakuyomu` へ分離する。
- 共通 HTTP は `internal/downloader.Client` に集約し、User-Agent、context、timeout、retry、wait、429/5xx backoff を実装する。
- 外部サイトへの通常テストは行わず、HTML/JSON fixture と fake HTTP transport で検証する。
- 保存先は narou.rb 互換 library root とし、`.narou/database.yaml`、`toc.yaml`、`本文/*.yaml`、`挿絵/` を書く。
- `download --epub` は保存後に library の小説ディレクトリ直下へ EPUB を出力する。
