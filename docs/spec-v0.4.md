# disk-smi 仕様書 v0.4

## 1. プロジェクト情報

| 項目      | 内容                                   |
| ------- | ------------------------------------ |
| プロジェクト名 | `disk-smi`                           |
| 正式名称    | **Disk System Management Interface** |
| コマンド名   | `disk-smi`                           |
| リポジトリ名  | `disk-smi`                           |
| 主対象     | macOS上のSSD                           |
| 実装言語    | Go                                   |
| 標準言語    | 英語                                   |
| 日本語表示   | `-jp`                                |
| データ取得   | `smartctl`、`diskutil`                |
| 配布方法    | GitHub Releases、Homebrew Tap         |
| 基本動作    | 読み取り専用                               |

`disk-smi` は、`disk` と `SMI = System Management Interface` を組み合わせた名称とする。

本プロジェクトにおける「Management」は、ストレージ状態の収集・整理・監視を意味する。少なくともv1系では、SSDの初期化、消去、パーティション変更、修復、マウント変更、SMART設定変更などは行わない。

---

# 2. 製品コンセプト

> CrystalDiskInfoに相当するSSD情報を、macOSのターミナル上で、`nvidia-smi`に近い固定グリッド形式で表示する。

主な目的は以下。

* SSDの総合状態を即座に確認できる
* 推定耐久残量を確認できる
* 総読み込み量・総書き込み量を確認できる
* 通電時間、電源投入回数、異常電源断回数を確認できる
* 温度やエラー情報を確認できる
* 英語・日本語の双方で縦線が正確に揃う
* Homebrewから導入できる
* 取得不能な値を、誤って正常またはゼロと表示しない

---

# 3. 対応範囲

## 3.1 v1.0正式対応

* macOS
* Apple Silicon Mac
* Intel Mac
* 内蔵NVMe SSD
* `smartmontools`がJSONとして取得できるSMART情報
* Unicode対応ターミナル
* ASCII罫線への切り替え
* 英語表示
* 日本語表示
* 単発表示
* 定期更新表示
* JSON出力

## 3.2 Best-effort対応

* SATA SSD
* Thunderbolt接続SSD
* USB-SATA SSD
* USB-NVMe SSD
* 複数物理SSD

外付けSSDはUSBブリッジやケース側のSMARTパススルー対応状況に依存する。

取得できない場合は、正常とは判定せず、`UNKNOWN`または`UNAVAILABLE`を表示する。

## 3.3 初期対象外

* SSDの消去
* SSDの修復
* パーティション変更
* ファームウェア更新
* SMART設定変更
* セルフテスト開始・停止
* RAID内部の物理ドライブ制御
* NAS経由のドライブ
* HDD固有の詳細診断
* 故障時期の予測
* 残り使用日数の推定
* 故障確率の算出

---

# 4. 用語と表示上の重要原則

## 4.1 「総合状態」と「耐久残量」を分離する

以下は別の指標として表示する。

```text
Overall Status    GOOD
Endurance         70%
```

日本語では以下。

```text
総合状態          正常
耐久残量          70%
```

`70%`は健康確率や故障確率ではない。SSDが報告した耐久使用率を基にした、推定公称耐久の残量である。

したがって、以下の表現は禁止する。

```text
Health 70%
健康度 70%
故障まで70%
寿命70%
```

採用する表現は以下。

```text
Endurance 70%
Endurance remaining 70%
耐久残量 70%
耐久残量（推定）70%
```

## 4.2 耐久残量0%の意味

人間向け表示では、耐久残量が取得できる場合、必ず注記を表示する。

英語：

```text
NOTE: Endurance 0% means the device-reported rated endurance has been consumed.
It does not mean immediate SSD failure.
```

日本語：

```text
注記: 耐久残量0%はSSDが報告する推定公称耐久を消費した目安です。
即時故障や使用不能を意味しません。
```

SSDは0%より前に故障する可能性も、0%到達後も動作する可能性もある。

---

# 5. 標準UI

## 5.1 表示方針

SSD 1台につき、1つの大型パネルを表示する。

パネルは以下の構成とする。

1. SSD識別情報
2. 上段4列の主要指標
3. 健康・耐久情報
4. 電源・使用状況
5. 信頼性・温度情報
6. デバイス情報
7. 耐久残量に関する注記

複数SSDがある場合は、大型パネルを縦方向に並べる。

---

## 5.2 標準英語表示

基準幅は100表示セルとする。

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ APPLE SSD AP1024Z / 1.00 TB / NVMe / Internal / disk0                                            │
├───────────────────────┼──────────────┼──────────────────────┼────────────────────────────────────┤
│    OVERALL STATUS     │ TEMPERATURE  │    POWER-ON TIME     │         LIFETIME HOST I/O          │
│                       │              │                      │                                    │
│         GOOD          │     36°C     │     1,500 hours      │        Host writes  4.20 TB        │
│     Endurance 70%     │              │      62 d 12 h       │        Host reads   3.50 TB        │
│                       │              │                      │                                    │
├─ HEALTH & ENDURANCE ───────────────────────────┼─ POWER & USAGE ─────────────────────────────────┤
│ Endurance used                             30% │ SSD power cycles                            428 │
│ Available spare                           100% │ Unsafe shutdowns                              7 │
│ Spare threshold                            10% │ Controller busy time                  112 hours │
│ SMART assessment                        PASSED │ Read commands                        82,019,334 │
│ Critical warning                          None │ Write commands                       61,274,004 │
├─ RELIABILITY & THERMALS ───────────────────────┼─ DEVICE INFORMATION ────────────────────────────┤
│ Media/data errors                            0 │ Model                         APPLE SSD AP1024Z │
│ Error log entries                            0 │ Firmware                              874.120.9 │
│ Warning-temp time                        0 min │ NVMe version                                1.4 │
│ Critical-temp time                       0 min │ Transport                           PCIe / NVMe │
│ Temperature sensors                  36 / 41°C │ Serial                                 ****9K2A │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ NOTE: Endurance 0% means the device-reported rated endurance has been consumed.                  │
│ It does not mean immediate SSD failure.                                                          │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 5.3 日本語表示

```bash
disk-smi -jp
```

表示例：

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ APPLE SSD AP1024Z / 1.00 TB / NVMe / 内蔵 / disk0                                                │
├───────────────────────┼──────────────┼──────────────────────┼────────────────────────────────────┤
│       総合状態        │     温度     │       通電時間       │           累積ホストI/O            │
│                       │              │                      │                                    │
│         正常          │     36°C     │      1,500時間       │       総書き込み量  4.20 TB        │
│     耐久残量 70%      │              │      62日12時間      │       総読み込み量  3.50 TB        │
│                       │              │                      │                                    │
├─ 健康・耐久 ───────────────────────────────────┼─ 電源・使用状況 ────────────────────────────────┤
│ 耐久使用率                                 30% │ SSD電源投入回数                           428回 │
│ 予備領域                                  100% │ 異常電源断回数                              7回 │
│ 予備領域閾値                               10% │ 累積I/O処理時間                         112時間 │
│ SMART判定                                 合格 │ 読み込みコマンド数                   82,019,334 │
│ 重大警告                                  なし │ 書き込みコマンド数                   61,274,004 │
├─ 信頼性・温度 ─────────────────────────────────┼─ デバイス情報 ──────────────────────────────────┤
│ メディア・整合性エラー                     0件 │ モデル                        APPLE SSD AP1024Z │
│ エラーログ件数                             0件 │ ファームウェア                        874.120.9 │
│ 警告温度滞在時間                           0分 │ NVMeバージョン                              1.4 │
│ 危険温度滞在時間                           0分 │ 接続方式                            PCIe / NVMe │
│ 温度センサー                         36 / 41°C │ シリアル                               ****9K2A │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│ 注記: 耐久残量0%はSSDが報告する推定公称耐久を消費した目安です。                                  │
│ 即時故障や使用不能を意味しません。                                                               │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

英語版と日本語版は同じ表示セル幅を持つ。

日本語版は、英語版を描画した後に文字列置換して作ってはならない。日本語ラベルへ切り替えた後、すべての表示幅を再計算する。

---

# 6. 表示項目

## 6.1 SSD識別情報

| 内部フィールド      | 英語           | 日本語       |
| ------------ | ------------ | --------- |
| model        | Model        | モデル       |
| capacity     | Capacity     | 容量        |
| protocol     | Protocol     | プロトコル     |
| transport    | Transport    | 接続方式      |
| location     | Location     | 接続場所      |
| device       | Device       | デバイス      |
| serial       | Serial       | シリアル      |
| firmware     | Firmware     | ファームウェア   |
| nvme_version | NVMe version | NVMeバージョン |

ヘッダーでは以下を表示する。

```text
Model / Capacity / Protocol / Location / Device
```

例：

```text
APPLE SSD AP1024Z / 1.00 TB / NVMe / Internal / disk0
```

---

## 6.2 主要指標

上段4列には以下を表示する。

| 列 | 主表示       | 副表示       |
| - | --------- | --------- |
| 1 | 総合状態      | 耐久残量      |
| 2 | 現在温度      | なし        |
| 3 | 通電時間      | 日数・時間換算   |
| 4 | 総ホスト書き込み量 | 総ホスト読み込み量 |

---

## 6.3 健康・耐久

| 英語                  | 日本語     |
| ------------------- | ------- |
| Endurance used      | 耐久使用率   |
| Endurance remaining | 耐久残量    |
| Available spare     | 予備領域    |
| Spare threshold     | 予備領域閾値  |
| SMART assessment    | SMART判定 |
| Critical warning    | 重大警告    |

### 耐久残量計算

```text
endurance_remaining = clamp(100 - endurance_used, 0, 100)
```

例：

```text
Endurance used       30%
Endurance remaining  70%
```

使用率が126%の場合：

```text
Endurance used       126%
Endurance remaining  0%
```

使用率126%を100%へ丸めてはならない。生の使用率を保持する。

---

## 6.4 累積ホストI/O

| 英語                | 日本語          |
| ----------------- | ------------ |
| Host reads        | 総読み込み量       |
| Host writes       | 総書き込み量       |
| Read commands     | 読み込みコマンド数    |
| Write commands    | 書き込みコマンド数    |
| Media/NAND writes | NAND・媒体書き込み量 |

NVMeのデータユニットをバイトへ変換する場合：

```text
bytes = data_units × 1,000 × 512
```

既定では10進単位を使用する。

```text
1 kB = 1,000 B
1 MB = 1,000,000 B
1 GB = 1,000,000,000 B
1 TB = 1,000,000,000,000 B
```

`--iec`指定時はGiB・TiBを使用する。

ホスト書き込み量とNAND物理書き込み量は別物である。NAND書き込み量が取得できない場合、ホスト書き込み量から推測してはならない。

---

## 6.5 電源・稼働情報

| 英語                   | 日本語       |
| -------------------- | --------- |
| Power-on time        | 通電時間      |
| SSD power cycles     | SSD電源投入回数 |
| Unsafe shutdowns     | 異常電源断回数   |
| Controller busy time | 累積I/O処理時間 |

`Power cycles`はMacの起動回数とは表現しない。

`Unsafe shutdowns`はmacOSのクラッシュ回数とは表現しない。

通電時間は2種類の表示を行う。

```text
1,500 hours
62 d 12 h
```

日本語：

```text
1,500時間
62日12時間
```

---

## 6.6 信頼性・温度

| 英語                   | 日本語         |
| -------------------- | ----------- |
| Media/data errors    | メディア・整合性エラー |
| Error log entries    | エラーログ件数     |
| Warning-temp time    | 警告温度滞在時間    |
| Critical-temp time   | 危険温度滞在時間    |
| Temperature sensors  | 温度センサー      |
| Warning temperature  | 警告温度        |
| Critical temperature | 危険温度        |
| Last self-test       | 最終自己診断      |

エラーログ件数が1以上であることだけを理由に、SSDを危険判定してはならない。

異常電源断回数が多いことだけを理由に、SSDを危険判定してはならない。

総書き込み量が多いことだけを理由に、SSDを危険判定してはならない。

---

# 7. 総合状態判定

## 7.1 状態

| 内部値      | 英語       | 日本語 |
| -------- | -------- | --- |
| good     | GOOD     | 正常  |
| caution  | CAUTION  | 注意  |
| critical | CRITICAL | 危険  |
| unknown  | UNKNOWN  | 不明  |

取得品質は別に管理する。

| 内部値         | 英語          | 日本語  |
| ----------- | ----------- | ---- |
| full        | FULL        | 完全   |
| partial     | PARTIAL     | 一部   |
| unavailable | UNAVAILABLE | 取得不能 |

---

## 7.2 CRITICAL

次のいずれかを満たした場合。

* SMART総合判定が失敗
* NVMe Critical Warningの重大ビットが立っている
* SSDが読み取り専用状態へ移行
* SSDが信頼性低下を報告
* Available Spareが閾値を下回った
* 現在温度がSSD自身の危険温度閾値以上
* 直近の自己診断が致命的失敗
* デバイス自身が重大障害を報告

---

## 7.3 CAUTION

次のいずれかを満たした場合。

* 耐久使用率が90%以上
* 耐久残量が10%以下
* 耐久使用率が100%以上
* メディア・整合性エラーが1件以上
* メディアエラーが前回取得時より増加
* 現在温度が警告温度閾値以上
* 自己診断履歴に失敗がある
* 一部の重要指標が欠落している

耐久使用率100%以上だけを理由に、`CRITICAL`としてはならない。

---

## 7.4 GOOD

次をすべて満たす場合。

* SMART判定が合格
* 重大警告なし
* 予備領域が閾値より上
* メディア・整合性エラーなし
* 危険温度状態ではない
* 直近自己診断に失敗なし
* 耐久使用率90%未満
* 正常判定に必要な主要データを取得できている

---

## 7.5 UNKNOWN

次のいずれか。

* SMART情報を取得できない
* 権限不足
* USBブリッジが非対応
* 出力JSONが不完全
* メーカー固有値を解釈できない
* 正常判定に必要な主要フィールドが不足
* `smartctl`自体を実行できない

取得できないことを、正常と解釈してはならない。

---

## 7.6 判定理由コード

健康判定は表示文字列だけではなく、機械可読な理由コードを持つ。

例：

```text
SMART_FAILED
CRITICAL_WARNING_ACTIVE
CRITICAL_WARNING_READ_ONLY
AVAILABLE_SPARE_BELOW_THRESHOLD
ENDURANCE_LOW
ENDURANCE_RATED_LIMIT_REACHED
MEDIA_ERRORS_PRESENT
TEMPERATURE_WARNING
TEMPERATURE_CRITICAL
SELF_TEST_FAILED
REQUIRED_DATA_MISSING
SMART_DATA_UNAVAILABLE
PERMISSION_REQUIRED
```

英語・日本語の説明は、同じ理由コードから生成する。

---

# 8. CLI仕様

## 8.1 基本コマンド

```bash
disk-smi
```

検出された全物理SSDを表示する。

```bash
disk-smi disk0
```

指定SSDだけを表示する。

```bash
disk-smi /dev/disk0
```

完全なデバイスパスも受け付ける。

---

## 8.2 日本語表示

```bash
disk-smi -jp
```

`-jp`は、`-j`と`-p`の組み合わせではなく、1つのオプション名として解釈する。

以下も同義とする。

```bash
disk-smi --jp
disk-smi --lang ja-JP
disk-smi --lang=ja-JP
```

標準は英語。

```bash
disk-smi
```

---

## 8.3 主要オプション

| オプション            | 内容               |
| ---------------- | ---------------- |
| `-jp`            | 日本語表示            |
| `--lang LANG`    | 表示言語指定           |
| `--ascii`        | ASCII罫線          |
| `--width N`      | パネル幅指定           |
| `--width auto`   | 端末幅から自動決定        |
| `-l N`           | N秒ごとに再取得         |
| `--loop N`       | 定期更新             |
| `--summary`      | 複数SSD用簡易一覧       |
| `--json`         | JSON出力           |
| `--json-pretty`  | 整形済みJSON         |
| `--no-color`     | 色を無効化            |
| `--color auto`   | TTYの場合のみ色を使用     |
| `--color always` | 常に色を使用           |
| `--color never`  | 色を使用しない          |
| `--iec`          | GiB・TiB表示        |
| `--show-serial`  | シリアル番号を完全表示      |
| `--version`      | バージョン表示          |
| `--help`         | ヘルプ              |
| `--input FILE`   | 開発・テスト用fixture入力 |

`-j`と`-p`はv1.0では使用せず、将来用として予約する。

---

## 8.4 定期更新

```bash
disk-smi -l 5
```

日本語：

```bash
disk-smi -jp -l 5
```

仕様：

* 既定更新間隔は5秒
* 最低更新間隔は2秒
* `Control+C`で終了
* TTYでは同じ画面を再描画
* 非TTYではサンプルを時系列で追記
* 端末サイズ変更時にレイアウトを再計算
* カウンター差分から速度を算出
* カウンターが減少した場合はリセットまたはデバイス交換として扱う

2回目以降の追加表示候補：

```text
Read rate
Write rate
Read IOPS
Write IOPS
Temperature change
```

日本語：

```text
読み込み速度
書き込み速度
読み込みIOPS
書き込みIOPS
温度変化
```

---

# 9. ターミナル描画仕様

## 9.1 表示幅の基準

すべての幅計算を、次のいずれでもなく、**ターミナル表示セル数**で行う。

使用禁止の基準：

* UTF-8バイト数
* Goの`len(string)`
* Unicodeコードポイント数だけの計算

必要な基準：

```text
terminal display cell width
```

例：

| 文字      | 表示幅 |
| ------- | --: |
| `A`     |   1 |
| `1`     |   1 |
| `温`     |   2 |
| `度`     |   2 |
| 結合文字    |   0 |
| ANSIコード |   0 |
| 一般的な絵文字 |   2 |

---

## 9.2 必須関数

描画層は次に相当する機能を持つ。

```go
DisplayWidth(text string) int
PadLeft(text string, width int) string
PadRight(text string, width int) string
PadCenter(text string, width int) string
TruncateDisplay(text string, width int) string
RenderCell(text string, width int, alignment Alignment) string
StripANSI(text string) string
SanitizeTerminalText(text string) string
```

`width`は表示セル数である。

通常の以下の実装だけに依存してはならない。

```go
fmt.Printf("%-20s", text)
```

---

## 9.3 グラフェム単位の処理

切り詰めは、バイト単位やrune単位ではなく、可能な限りUnicodeグラフェムクラスタ単位で行う。

途中で切断してはならないもの：

* 日本語文字
* 結合文字列
* ZWJ絵文字
* 異体字セレクタ付き文字
* サロゲート相当の複合表現

切り詰め記号は既定でASCIIの以下を使用する。

```text
...
```

---

## 9.4 ANSIカラー

ANSIエスケープシーケンスは表示幅0として扱う。

例：

```text
\x1b[32mGOOD\x1b[0m
```

表示幅は4である。

推奨処理順：

1. 色なしの文字列を生成
2. セル幅を計算
3. パディングを追加
4. ANSI装飾を適用
5. ANSIを除去した状態で最終幅を再検証

---

## 9.5 固定グリッド

パネル幅を`W`とする。

```text
内部幅 I = W - 2
```

左右外枠が各1セルである。

### 上段4列

区切りが3本ある。

```text
A = I - 3

C1 = floor(A × 0.25)
C2 = floor(A × 0.15)
C3 = floor(A × 0.24)
C4 = A - C1 - C2 - C3
```

`W = 100`の場合：

```text
C1 = 23
C2 = 14
C3 = 22
C4 = 36
```

検算：

```text
1 + 23 + 1 + 14 + 1 + 22 + 1 + 36 + 1 = 100
```

### 下段2列

```text
D = I - 1
L = floor(D / 2)
R = D - L
```

`W = 100`の場合：

```text
L = 48
R = 49
```

検算：

```text
1 + 48 + 1 + 49 + 1 = 100
```

---

## 9.6 行幅保証

すべての描画行は、次を満たさなければならない。

```text
DisplayWidth(renderedLine) == panelWidth
```

描画層は、行幅が一致しない場合にエラーを返す。

テストでは、全行について幅を検証する。

---

## 9.7 日本語表示

日本語表示時の処理順：

```text
1. ja-JPロケールを選択
2. 日本語ラベルを取得
3. 日本語文字列をサニタイズ
4. 表示セル幅を計算
5. 切り詰め
6. パディング
7. 罫線と結合
8. 行全体の幅を検証
9. ANSI色を適用
10. 再度表示幅を検証
```

英語版完成後の文字列置換は禁止する。

---

## 9.8 UnicodeとASCII

標準はUnicode罫線。

```text
┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ │ ─
```

ASCIIモード：

```bash
disk-smi --ascii
```

使用文字：

```text
+ - |
```

次の場合はASCIIへフォールバックできる。

* `--ascii`指定
* `TERM=dumb`
* UTF-8以外のロケール
* Unicode罫線の表示が困難な端末

---

## 9.9 端末幅

### 100列以上

上段4列、下段2列の完全表示。

### 80〜99列

主要構造を維持しつつ、長いラベルやモデル名を切り詰める。

### 60〜79列

上段を2列×2段に変更する。

```text
Overall Status | Temperature
Power-on Time  | Lifetime Host I/O
```

### 60列未満

縦型表示へフォールバックする。

```text
Model: APPLE SSD AP1024Z
Status: GOOD
Endurance: 70%
Temperature: 36°C
Host writes: 4.20 TB
```

狭い端末で罫線を無理に維持し、表示を破壊してはならない。

---

# 10. データ欠損

取得不能な値は、ゼロとして表示してはならない。

| 状態   | 英語                    | 日本語    |
| ---- | --------------------- | ------ |
| 値なし  | `—`                   | `—`    |
| 非対応  | `UNSUPPORTED`         | `非対応`  |
| 権限不足 | `PERMISSION REQUIRED` | `権限不足` |
| 取得失敗 | `ERROR`               | `取得失敗` |
| 解釈不能 | `UNKNOWN`             | `不明`   |

例：

```text
Media/NAND writes  —
```

これは0バイトではなく、値が取得できなかったことを意味する。

内部データモデルでも、0と未取得を区別する。

---

# 11. 情報取得仕様

## 11.1 デバイス検出

macOS標準の読み取り専用コマンドを使用する。

```bash
/usr/sbin/diskutil list -plist
/usr/sbin/diskutil info -plist /dev/disk0
```

対象は物理ディスクとする。

APFSコンテナ、論理ボリューム、パーティションを物理SSDとして重複表示してはならない。

---

## 11.2 SMART取得

標準取得：

```bash
smartctl -a -j /dev/disk0
```

詳細取得：

```bash
smartctl -x -j /dev/disk0
```

要件：

* シェルを介さず実行する
* `exec.CommandContext`相当を使う
* 固定引数配列を使う
* タイムアウトを設定する
* デバイス名を検証する
* 標準出力のJSONを優先して解析する
* 非ゼロ終了コードでもJSONが存在する場合は解析する
* `smartctl`の終了コードを単純な成功・失敗として扱わない
* stderrも構造化して保持する
* 実行したコマンドをデバッグ出力できるようにする
* シリアル番号を通常ログへ出さない

---

## 11.3 権限

初期実装では、`disk-smi`自身が自動で`sudo`を起動しない。

権限不足時：

```text
SMART data requires elevated permission.

Run:
  sudo disk-smi
```

日本語：

```text
SMART情報の取得には管理者権限が必要です。

次を実行してください:
  sudo disk-smi
```

禁止事項：

* `/etc/sudoers`の変更
* setuidバイナリ化
* パスワードの読み取り
* `sudo`パスワードの保存
* シェル経由のコマンド実行

---

## 11.4 デバイス名検証

受け付ける形式：

```text
disk0
disk1
/dev/disk0
/dev/disk1
/dev/rdisk0
```

任意のパスを外部コマンドへ渡してはならない。

`..`、改行、空白、シェル記号を含む入力は拒否する。

---

# 12. 内部データモデル

未取得とゼロを区別できる型を使用する。

概念例：

```go
type Optional[T any] struct {
    Value  T
    Valid  bool
    Reason MissingReason
}
```

主要モデル：

```go
type DriveSnapshot struct {
    Device     DeviceInfo
    Metrics    DriveMetrics
    Assessment HealthAssessment
    Source     SourceInfo
}
```

## 12.1 DeviceInfo

```go
type DeviceInfo struct {
    DevicePath   string
    BSDName      string
    Model        string
    Serial       Optional[string]
    Firmware     Optional[string]
    CapacityByte BigCounter
    Protocol     string
    Transport    Optional[string]
    Location     Optional[string]
    NVMeVersion  Optional[string]
}
```

## 12.2 DriveMetrics

```go
type DriveMetrics struct {
    SMARTPassed            Optional[bool]
    CriticalWarning        Optional[uint64]

    TemperatureCelsius     Optional[int64]
    WarningTemperature     Optional[int64]
    CriticalTemperature    Optional[int64]
    TemperatureSensors     []Optional[int64]

    EnduranceUsedPercent   Optional[uint64]
    AvailableSparePercent  Optional[uint64]
    SpareThresholdPercent  Optional[uint64]

    HostReadsBytes         Optional[BigCounter]
    HostWritesBytes        Optional[BigCounter]
    MediaWritesBytes       Optional[BigCounter]

    ReadCommands           Optional[BigCounter]
    WriteCommands          Optional[BigCounter]

    PowerOnHours           Optional[BigCounter]
    PowerCycles            Optional[BigCounter]
    UnsafeShutdowns        Optional[BigCounter]
    ControllerBusyMinutes  Optional[BigCounter]

    MediaErrors            Optional[BigCounter]
    ErrorLogEntries        Optional[BigCounter]

    WarningTemperatureTime Optional[BigCounter]
    CriticalTemperatureTime Optional[BigCounter]
}
```

累積カウンターは64ビットへ収まると仮定しない。

Goでは`math/big.Int`または同等のオーバーフローしない表現を使用する。

---

# 13. JSON出力

```bash
disk-smi --json
disk-smi -jp --json
```

表示言語に関係なく、JSONキーと列挙値は英語で固定する。

大きな累積値は、JavaScript等で精度を失わないよう10進文字列として出力する。

例：

```json
{
  "schema_version": 1,
  "generated_at": "2026-06-25T19:42:10+09:00",
  "display_locale": "ja-JP",
  "drives": [
    {
      "device": {
        "path": "/dev/disk0",
        "name": "disk0",
        "model": "APPLE SSD AP1024Z",
        "capacity_bytes": "1000204886016",
        "protocol": "NVMe",
        "location": "internal",
        "serial_masked": "****9K2A"
      },
      "health": {
        "overall_status": "good",
        "data_quality": "full",
        "reason_codes": []
      },
      "endurance": {
        "used_percent": 30,
        "remaining_percent": 70,
        "remaining_is_failure_probability": false,
        "zero_means_immediate_failure": false
      },
      "io": {
        "host_reads_bytes": "3500000000000",
        "host_writes_bytes": "4200000000000",
        "read_commands": "82019334",
        "write_commands": "61274004"
      },
      "power": {
        "power_on_hours": "1500",
        "power_cycles": "428",
        "unsafe_shutdowns": "7"
      }
    }
  ]
}
```

欠損値は`null`とする。

ゼロと`null`を混同しない。

---

# 14. プライバシー

既定ではシリアル番号をマスクする。

```text
****9K2A
```

完全表示：

```bash
disk-smi --show-serial
```

デフォルトで表示しないもの：

* 完全なシリアル番号
* WWN
* EUI-64
* UUID
* ボリュームUUID
* ユーザー名
* ホスト名
* マウントポイント

以下は禁止。

* テレメトリー
* 外部サーバーへの送信
* SMARTデータの自動アップロード
* 利用統計の自動送信

---

# 15. 安全性

`disk-smi`は読み取り専用でなければならない。

禁止コマンド・操作：

* `diskutil eraseDisk`
* `diskutil partitionDisk`
* `diskutil repairDisk`
* `diskutil mount`
* `diskutil unmount`
* `diskutil apfs deleteContainer`
* SMART属性変更
* セルフテスト開始
* ファームウェア更新
* ディスクへの直接書き込み
* `/dev/disk*`への書き込みオープン

外部コマンド実行時に、以下を使用してはならない。

```go
exec.Command("sh", "-c", ...)
exec.Command("bash", "-c", ...)
```

---

# 16. 推奨パッケージ構成

```text
disk-smi/
├── cmd/
│   └── disk-smi/
│       └── main.go
├── internal/
│   ├── app/
│   ├── cli/
│   ├── collector/
│   ├── discovery/
│   ├── smartctl/
│   ├── model/
│   ├── health/
│   ├── render/
│   ├── i18n/
│   ├── units/
│   ├── sanitize/
│   ├── jsonout/
│   └── version/
├── docs/
│   └── spec-v0.4.md
├── testdata/
│   ├── smartctl/
│   ├── diskutil/
│   └── golden/
├── .github/
│   └── workflows/
├── AGENTS.md
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

責務：

| パッケージ       | 責務              |
| ----------- | --------------- |
| `cli`       | 引数解析            |
| `discovery` | macOS物理ディスク検出   |
| `collector` | 外部コマンドの実行管理     |
| `smartctl`  | smartctl JSON解析 |
| `model`     | 正規化データモデル       |
| `health`    | 総合状態判定          |
| `render`    | 表形式描画           |
| `i18n`      | 英語・日本語文字列       |
| `units`     | 容量、時間、温度変換      |
| `sanitize`  | 端末文字列の安全化       |
| `jsonout`   | JSON出力          |
| `app`       | 全処理の統合          |

フルスクリーンTUIフレームワークは初期段階では使用しない。静的な行指向レンダラーから開始する。

---

# 17. エラー終了コード

通常表示では、SSDが`CAUTION`や`CRITICAL`であっても、表示処理に成功した場合は終了コード0とする。

| コード | 内容           |
| --: | ------------ |
|   0 | 正常に表示完了      |
|   2 | CLI引数エラー     |
|   3 | 必須依存関係がない    |
|   4 | 権限不足         |
|   5 | 対象SSDが見つからない |
|   6 | SMART情報取得失敗  |
|   7 | 内部エラー        |

将来、監視システム向けに健康状態を終了コードへ反映する場合は、別の`--check`オプションとして実装する。

---

# 18. テスト仕様

## 18.1 描画テスト

以下の全組み合わせをテストする。

| 言語  | 罫線      |
| --- | ------- |
| 英語  | Unicode |
| 英語  | ASCII   |
| 日本語 | Unicode |
| 日本語 | ASCII   |

対象幅：

```text
60
79
80
96
100
120
```

すべての行について検証する。

```text
DisplayWidth(line) == expectedWidth
```

---

## 18.2 Unicodeテスト

最低限、以下を含める。

```text
APPLE SSD AP1024Z
非常に長い日本語SSDモデル名
é
é
温度
36°C
😀
PCIe / NVMe
ANSIエスケープ付き文字列
改行付き不正文字列
タブ付き文字列
```

---

## 18.3 SMART fixture

```text
nvme-good.json
nvme-endurance-0.json
nvme-endurance-90.json
nvme-endurance-100.json
nvme-endurance-126.json
nvme-critical-warning.json
nvme-media-errors.json
nvme-missing-temperature.json
nvme-missing-endurance.json
nvme-large-counters.json
nvme-malformed.json
sata-good.json
usb-unavailable.json
permission-denied.json
```

必須ケース：

* 使用率0% → 残量100%
* 使用率90% → CAUTION
* 使用率100% → CAUTION、残量0%
* 使用率126% → CAUTION、残量0%、使用率126%を保持
* SMART失敗 → CRITICAL
* 重大警告 → CRITICAL
* 予備領域不足 → CRITICAL
* メディアエラー → CAUTION以上
* エラーログ件数のみ非ゼロ → 自動CRITICALにしない
* 欠損値 → ゼロにしない
* 巨大カウンター → オーバーフローしない

---

## 18.4 Golden test

```text
testdata/golden/en-100-unicode.txt
testdata/golden/en-100-ascii.txt
testdata/golden/ja-100-unicode.txt
testdata/golden/ja-100-ascii.txt
```

縦線位置が1セルでも変化した場合、テストを失敗させる。

---

## 18.5 CI

CIでは実ディスクへアクセスしない。

実行項目：

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
git diff --check
```

実機統合テストは明示的な操作でのみ実行する。

```bash
DISK_SMI_INTEGRATION=1 go test ./internal/integration
```

---

# 19. 実装順序

## Milestone 1: 描画エンジン

* syntheticデータから画面表示
* 英語
* 日本語
* Unicode罫線
* ASCII罫線
* 表示幅計算
* Golden test
* 実SSDアクセスなし

## Milestone 2: smartctl JSON解析

* fixtureだけを解析
* NVMe主要フィールド
* 欠損値
* 巨大カウンター
* データユニット変換
* 実SSDアクセスなし

## Milestone 3: 健康判定

* GOOD
* CAUTION
* CRITICAL
* UNKNOWN
* 理由コード
* テーブル駆動テスト

## Milestone 4: CLI・日本語化

* `-jp`
* `--ascii`
* `--width`
* `--json`
* `--no-color`
* `--input`
* `--help`
* `--version`

## Milestone 5: macOSデバイス検出

* `diskutil list -plist`
* `diskutil info -plist`
* 物理SSDの識別
* APFSコンテナ重複排除

## Milestone 6: smartctl実行

* タイムアウト
* 権限エラー
* 非ゼロ終了コード解析
* 外付けSSDの取得失敗処理
* 実機テストは明示承認後のみ

## Milestone 7: 定期更新

* `-l 5`
* 画面再描画
* I/O速度差分
* IOPS差分
* 温度変化
* カウンターリセット検知

## Milestone 8: リリース

* README
* GitHub Actions
* バージョン埋め込み
* GitHub Release
* Homebrew Tap
* `smartmontools`依存関係

---

# 20. Homebrew仕様

最終的な利用形式：

```bash
brew install <owner>/tap/disk-smi
```

Formulaの実行時依存：

```ruby
depends_on "smartmontools"
```

ビルド時依存：

```ruby
depends_on "go" => :build
```

インストール後：

```bash
disk-smi
disk-smi -jp
```

Homebrew Formulaのテストでは、実SSDや`sudo`を使用せず、fixture入力で動作確認する。

---

# 21. v1.0完成条件

次をすべて満たした時点でv1.0とする。

1. `disk-smi`で英語表示できる
2. `disk-smi -jp`で日本語表示できる
3. 日本語でも縦線が正確に揃う
4. SSD 1台につき大型パネルを表示する
5. 総合状態と耐久残量を分離する
6. 耐久残量0%の意味を常時明記する
7. 総読み込み量を表示する
8. 総書き込み量を表示する
9. 通電時間を表示する
10. SSD電源投入回数を表示する
11. 異常電源断回数を表示する
12. 読み書きコマンド数を表示する
13. 温度と温度履歴を表示する
14. メディアエラーを表示する
15. エラーログ件数を表示する
16. 未取得値をゼロとして扱わない
17. ANSIカラーで列位置がずれない
18. UnicodeとASCII罫線に対応する
19. 長い日本語・モデル名を安全に切り詰める
20. 全描画行が指定表示幅と一致する
21. JSONキーが言語によって変化しない
22. デフォルトでシリアル番号をマスクする
23. ディスクへの書き込み操作を一切行わない
24. Apple SiliconとIntel Macでビルドできる
25. Homebrewからインストールできる

このv0.4を、リポジトリ内での実装判断における正式な基準仕様とします。
