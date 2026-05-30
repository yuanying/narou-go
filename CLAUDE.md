# narou-go

narou.rb の後継となる Go 実装。既存の `library/` を直接読み込み、EPUB3 を生成する。

## コマンド

### ユニットテスト

```bash
go test ./...
```

### Lint

```bash
go tool golangci-lint run ./...
```

## 開発方針

- TDD で進める。テストを先に書き、失敗を確認してから実装する
- `library/` と `narou.rb/` はバージョン管理対象外（`.gitignore` で除外済み）
- 実装の詳細は `docs/` を参照
