GOBIN ?= $(shell go env GOPATH)/bin
GO_VERSION := $(shell awk '/^go / { print $$2; exit }' go.mod)

.PHONY: setup check-tools lint fmt test bench vulncheck generate ci help tidy

## go.mod の制約に従って tidy 実行
tidy:
	go mod tidy -go=$(GO_VERSION)

## ツールの存在チェック（なければ make setup を促す）
check-tools:
	@test -x $(GOBIN)/golangci-lint || (echo "Error: golangci-lint not found. Run 'make setup' first." && exit 1)
	@test -x $(GOBIN)/govulncheck || (echo "Error: govulncheck not found. Run 'make setup' first." && exit 1)
	@test -f .git/hooks/pre-commit || (echo "Error: lefthook hooks not installed. Run 'make setup' first." && exit 1)

## 開発環境セットアップ（ツールのインストール + lefthook セットアップ）
setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/evilmartians/lefthook@latest
	$(GOBIN)/lefthook install

## リンター実行
lint: check-tools
	$(GOBIN)/golangci-lint run ./...

## フォーマット（自動修正）
fmt: check-tools
	$(GOBIN)/golangci-lint fmt ./...
	$(GOBIN)/golangci-lint run --fix ./...

## テスト実行
test:
	go test -v -race -count=1 ./...
	cd cmd/genholidays && go test -v -race -count=1 ./...

## ベンチマーク実行
bench:
	go test -bench=. -benchmem -count=1 -run=^$$ ./...

## 脆弱性チェック
vulncheck: check-tools
	$(GOBIN)/govulncheck ./...

## 祝日データ生成（内閣府CSVから holidays_data.go を生成）
generate:
	cd cmd/genholidays && go run main.go -output ../../holidays_data.go

## CI相当のチェックをローカルで一括実行
ci: lint test vulncheck

## ヘルプ
help:
	@echo "使用可能なターゲット:"
	@echo "  make tidy         - go.mod の go バージョンに合わせて tidy 実行"
	@echo "  make check-tools  - 必要ツールと hooks の存在チェック"
	@echo "  make setup        - 開発ツールのインストール + lefthook セットアップ"
	@echo "  make lint         - リンター実行"
	@echo "  make fmt          - フォーマット + 自動修正"
	@echo "  make test         - テスト実行（-race 付き）"
	@echo "  make bench        - ベンチマーク実行"
	@echo "  make vulncheck    - 依存パッケージの脆弱性チェック"
	@echo "  make generate     - 祝日データ生成（内閣府CSV取得）"
	@echo "  make ci           - CI相当チェック一括実行（lint + test + vulncheck）"
