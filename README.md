# jp-holidays

日本の祝日を判定する Go ライブラリ。外部依存ゼロ。

[English README](README_en.md)

## 特徴

- 日付・月・年・範囲指定で祝日を検索
- 営業日ユーティリティ（次の営業日、前の営業日、営業日数カウント）
- カスタム休日のサポート（メモリ上、Calendar インスタンスごとに独立）
- スレッドセーフ（並行アクセス対応）
- 外部依存ゼロ — 祝日データはバイナリにコンパイル済み
- [内閣府公開データ](https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv)準拠（1955年〜2027年）
  - 毎週日曜日に最新版を取得して更新

## インストール

```bash
go get github.com/rabitt1ove/jp-holidays
```

## 基本的な使い方

```go
package main

import (
    "fmt"
    "time"

    jpholiday "github.com/rabitt1ove/jp-holidays"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func main() {
    t := time.Date(2026, time.January, 1, 0, 0, 0, 0, jst)

    fmt.Println(jpholiday.IsHoliday(t))    // true
    fmt.Println(jpholiday.HolidayName(t))  // 元日
    fmt.Println(jpholiday.IsBusinessDay(t)) // false
}
```

## API 一覧

### 祝日検索

| 関数 | 説明 |
| --- | --- |
| `IsHoliday(t time.Time) bool` | 指定日が祝日か判定 |
| `HolidayName(t time.Time) string` | 指定日の祝日名を取得（非祝日は空文字） |
| `HolidaysInYear(year int) []Holiday` | 指定年の祝日一覧 |
| `HolidaysInMonth(year int, month time.Month) []Holiday` | 指定月の祝日一覧 |
| `HolidaysBetween(from, to time.Time) []Holiday` | 指定範囲の祝日一覧（from, to を含む） |
| `Holidays() []Holiday` | 全祝日一覧 |

### 営業日ユーティリティ

| 関数 | 説明 |
| --- | --- |
| `IsBusinessDay(t time.Time) bool` | 営業日か判定（週末・祝日を除外） |
| `NextBusinessDay(t time.Time) time.Time` | 指定日以降の最初の営業日 |
| `PreviousBusinessDay(t time.Time) time.Time` | 指定日以前の最後の営業日 |
| `BusinessDaysBetween(from, to time.Time) int` | 範囲内の営業日数（from, to を含む） |
| `NextHoliday(t time.Time) (Holiday, bool)` | 指定日より後の次の祝日 |
| `PreviousHoliday(t time.Time) (Holiday, bool)` | 指定日より前の直近の祝日 |

### カスタム休日

| 関数 | 説明 |
| --- | --- |
| `AddCustomHoliday(t time.Time, name string)` | カスタム休日を追加 |
| `RemoveCustomHoliday(t time.Time)` | カスタム休日を削除 |
| `RemoveHoliday(t time.Time)` | 組み込み祝日を抑制（非表示にする） |
| `RestoreHoliday(t time.Time)` | 抑制した祝日を復元 |

### Calendar インスタンス

上記のすべての関数は `*Calendar` のメソッドとしても利用できます。`New()` で独立したインスタンスを作成し、インスタンスごとに異なるカスタム休日を管理できます：

```go
cal := jpholiday.New()
cal.AddCustomHoliday(time.Date(2024, 6, 15, 0, 0, 0, 0, jst), "会社記念日")

cal.IsHoliday(time.Date(2024, 6, 15, 0, 0, 0, 0, jst))       // true
jpholiday.IsHoliday(time.Date(2024, 6, 15, 0, 0, 0, 0, jst)) // false（デフォルトカレンダー）
```

## 型定義

```go
type Holiday struct {
    Date time.Time // 祝日の日付（UTC 0時）
    Name string    // 祝日名（例: "元日"）
}
```

## タイムゾーンの扱い

すべての `time.Time` 入力は、日付を抽出する前に **JST（Asia/Tokyo, UTC+9）に変換** されます。これにより、どのタイムゾーンで渡しても日本の暦日に基づいた正しい結果が返ります。

```go
// 2023-12-31 20:00 UTC = 2024-01-01 05:00 JST → 元日と判定
utcTime := time.Date(2023, 12, 31, 20, 0, 0, 0, time.UTC)
jpholiday.IsHoliday(utcTime)   // true（JSTでは1月1日）
jpholiday.HolidayName(utcTime) // "元日"
```

営業日判定（`IsBusinessDay` など）の曜日計算も同様に JST で行われます。

## ベンチマーク

Apple M2 Pro での計測結果 (`go test -bench=. -benchmem`)。
ns/op = 1操作あたりのナノ秒（10億分の1秒）。

| 関数 | 速度 | アロケーション |
| --- | --- | --- |
| `IsHoliday` | ~20 ns/op | 0 allocs |
| `HolidayName` | ~20 ns/op | 0 allocs |
| `IsBusinessDay` | ~21 ns/op | 0 allocs |
| `NextBusinessDay` | ~200 ns/op | 0 allocs |
| `BusinessDaysBetween` (1ヶ月) | ~1,300 ns/op | 0 allocs |
| `BusinessDaysBetween` (1年) | ~16,000 ns/op | 0 allocs |
| `HolidaysInYear` | ~12,000 ns/op | 9 allocs |
| `NextHoliday` / `PreviousHoliday` | ~11,000 ns/op | 0 allocs |

自分の環境で計測する場合:

```bash
go test -bench=. -benchmem ./...
```

## データソース

祝日データは内閣府が公開している CSV を使用しています：
<https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv>

- **データ範囲**: 1955年（昭和30年）〜 2027年（令和9年） — 内閣府の更新に応じて拡張
- **更新頻度**: 毎週日曜日に GitHub Actions で自動チェック
- **更新方法**: [デジタル庁推奨](https://www.digital.go.jp/resources/open_data)の [e-Gov データポータル CKAN API](https://data.e-gov.go.jp/data/api_guide) を使用して CSV の URL を動的に解決。API が利用できない場合は直接 URL にフォールバック。
- **更新フロー**: 新しいデータが検出された場合、プルリクエストが自動作成され、人間によるレビュー後にマージ

### データの出典

本ライブラリは、内閣府が [e-Gov データポータル](https://data.e-gov.go.jp/)で公開しているデータを使用しています。

> 出典：内閣府「国民の祝日」について
> <https://www8.cao.go.jp/chosei/shukujitsu/gaiyou.html>

データは [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/) 互換の条件で提供されています。本ライブラリの利用者は、基盤となる祝日データがこの政府ソースに由来することにご留意ください。

## ライセンス

MIT
