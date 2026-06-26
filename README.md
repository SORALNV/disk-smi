# disk-smi

`disk-smi` は、Mac のSSD状態をターミナルで確認するためのツールです。`nvidia-smi` のような固定幅パネルで、SSDの健康状態、耐久残量、温度、通電時間、電源投入回数、累積I/Oなどを表示します。

実装上の正式な仕様は [docs/spec-v0.4.md](docs/spec-v0.4.md) です。

## できること

- Apple Silicon Mac の内蔵NVMe SSDを可能な限り取得
- 日本語表示
- SSDの総合状態、耐久残量、温度、通電時間、累積ホストI/Oを表示
- SMART判定、重大警告、予備領域、電源投入回数、異常電源断回数などを表示
- シリアル番号は標準でマスク表示
- JSON出力
- 複数ドライブ向けのサマリ表示
- ループ表示

`disk-smi` は読み取り専用です。ディスクの消去、修復、パーティション変更、マウント、アンマウント、書き込みは行いません。

## ダウンロードとインストール

通常はビルド不要です。GitHub Releases のビルド済みバイナリをダウンロードして使います。

### Apple Silicon Mac

M1/M2/M3/M4 などのMacはこちらです。

```bash
mkdir -p ~/.local/bin
curl -L -o ~/.local/bin/disk-smi https://github.com/SORALNV/disk-smi/releases/latest/download/disk-smi-darwin-arm64
chmod +x ~/.local/bin/disk-smi
disk-smi -jp
```

`disk-smi: command not found` になる場合は、`~/.local/bin` をPATHに追加します。

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
disk-smi -jp
```

### Intel Mac

Intel Macはこちらです。

```bash
mkdir -p ~/.local/bin
curl -L -o ~/.local/bin/disk-smi https://github.com/SORALNV/disk-smi/releases/latest/download/disk-smi-darwin-amd64
chmod +x ~/.local/bin/disk-smi
disk-smi -jp
```

### 手元のMacがどちらか分からない場合

次で確認できます。

```bash
uname -m
```

- `arm64`: Apple Silicon Mac なので `disk-smi-darwin-arm64`
- `x86_64`: Intel Mac なので `disk-smi-darwin-amd64`

### 外付けSSDも詳しく見たい場合

外付けSSD、USB接続、SATA、特殊なケースでは `smartmontools` を入れると取得できる情報が増える場合があります。

```bash
brew install smartmontools
```

その上で、必要なら `smartctl` backend を指定します。

```bash
disk-smi -jp --backend smartctl
```

### ソースからビルドしたい場合

通常は不要です。開発する場合だけ使います。

```bash
brew install go
git clone https://github.com/SORALNV/disk-smi.git
cd disk-smi
go build -o disk-smi ./cmd/disk-smi
./disk-smi -jp
```

## 使い方

一番よく使うコマンドはこれです。

```bash
disk-smi -jp
```

表示例:

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ APPLE SSD AP1024Z / 1.00 TB / NVMe / 内蔵 / disk0                                                │
├──────────────────────────────────────────────────────────────────────────────────────────────────┤
│       総合状態              温度             通電時間                   累積ホストI/O            │
│         正常                36°C            1,500時間              総書き込み量  4.22 TB         │
│     耐久残量 100%                            62日12時間            総読み込み量  7.18 TB         │
├─ 健康・耐久 ───────────────────────────────────┼─ 電源・使用状況 ────────────────────────────────┤
│ 耐久使用率                                  0% │ SSD電源投入回数                           115回 │
│ SMART判定                                 合格 │ 異常電源断回数                              9回 │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

実際の表示内容は、Mac、SSD、macOS、権限、取得できるSMART/NVMe情報によって変わります。取得できない項目はターミナル表示では省略されるか、必要に応じて `不明`、`権限不足` などで表示されます。

## よく使うオプション

```text
-jp                 日本語表示。Apple Silicon Macではnative backendを標準使用
disk0               disk0だけを見る
/dev/disk0          /dev/disk0だけを見る
--summary           複数ドライブをコンパクトに一覧表示
-l 2                2秒ごとに更新
--loop 5            5秒ごとに更新
--json              JSONで出力
--json-pretty       読みやすいJSONで出力
--iec               KiB/MiB/GiB/TiB単位で表示
--show-serial       シリアル番号をマスクせず表示
--width 120         表示幅を120セルにする
--ascii             ASCII罫線で表示
--no-color          色を無効化
--debug             取得に使ったコマンドや失敗理由をstderrに表示
```

バックエンドを指定したい場合:

```text
--backend auto      smartctlが使える場合はsmartctl、必要に応じてnative fallback
--backend native    macOS内蔵API/IOKit系だけで取得
--backend smartctl  smartctlから取得
```

Apple Silicon Mac の内蔵SSDを見たいだけなら、まずはこれで十分です。

```bash
disk-smi -jp
```

## 権限が足りない場合

一部のSMART情報は、macOS側の権限や環境によって取得できないことがあります。その場合は次を試してください。

```bash
sudo disk-smi -jp
```

ただし、`disk-smi` 自体は `sudo` を自動では使いません。

## 表示される値について

### 耐久残量

`耐久残量` は、SSDが報告する推定寿命情報から計算した目安です。

`耐久残量 0%` は「メーカーが想定する公称耐久を消費した目安」であり、即時故障や使用不能を意味しません。ただし、バックアップや交換検討の目安にはなります。

### 累積ホストI/O

`disk-smi` の `総書き込み量` と `総読み込み量` は、SSD自身が報告する累積ホストI/Oです。

アクティビティモニタの「読み込まれたデータ」「書き込まれたデータ」は、macOS側のI/O統計です。現在の起動セッションやOS/driver側の集計に近い値で、SSDが持っている生涯累積SMART/NVMeカウンタとは別物です。

そのため、`disk-smi` とアクティビティモニタの値は一致しないことがあります。

簡単に言うと:

- `disk-smi`: SSDの健康状態と、生涯累積に近いカウンタを見る
- アクティビティモニタ: 今のMac上で発生しているI/Oを見る

## 取得できる情報と限界

Apple Silicon Mac の内蔵NVMe SSDでは、native backend が次の情報をできる限り取得します。

- `diskutil` のSMART情報
- IOKit経由のNVMe SMART Log
- IOKit経由のNVMe Identify情報
- OCP Cloud SMART Log Page `0xC0`
- HID温度センサー
- `system_profiler` のNVMeメタデータ
- IORegistry のコントローラ情報

ただし、すべてのSSDやmacOS環境で全項目が取れるわけではありません。SSDやコントローラが公開していない情報、macOSから見えない情報、権限が必要な情報は表示できません。

外付けSSD、USB接続、SATA、特殊なケースでは `smartctl` のほうが多く取れる場合があります。

```bash
brew install smartmontools
disk-smi -jp --backend smartctl
```

## JSON出力

機械処理したい場合はJSONを使えます。

```bash
disk-smi --json-pretty
```

シリアル番号は標準ではマスクされます。完全なシリアル番号をJSONに含めたい場合だけ、明示的に指定します。

```bash
disk-smi --json-pretty --show-serial
```

## GitHub Releasesから入れる場合

Releasesには、macOS向けのビルド済みバイナリを置きます。

```text
disk-smi-darwin-arm64
disk-smi-darwin-amd64
```

Apple Silicon Mac なら通常は `darwin-arm64` を選びます。

GitHubの画面からダウンロードした場合は、実行権限を付けてから使います。

```bash
mv disk-smi-darwin-arm64 disk-smi
chmod +x disk-smi
mkdir -p ~/.local/bin
cp disk-smi ~/.local/bin/
disk-smi -jp
```

## Homebrew Formulaについて

Formulaのテンプレートは [Formula/disk-smi.rb](Formula/disk-smi.rb) にあります。

現時点では `OWNER` や `sha256` がプレースホルダーのため、そのまま公開tapとして使う状態ではありません。リリースを作成したあと、次のヘルパーでFormulaを更新します。

```bash
ruby scripts/update_formula.rb --owner SORALNV --version <version> --sha256 <sha256>
```

## 開発者向け

標準チェック:

```bash
scripts/check_release_ready.sh
```

このスクリプトは次を実行します。

- `gofmt`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- 空白チェック
- Ruby/Formulaの構文チェック
- Formula更新ヘルパーのテスト
- macOS `amd64` / `arm64` のクロスビルド

CIと単体テストはfixtureを使い、実ディスクにはアクセスしません。

## 安全性

`disk-smi` は読み取り専用ツールです。

- ディスクを消去しません
- ディスクを修復しません
- パーティションを変更しません
- マウント/アンマウントしません
- SMARTセルフテストを開始しません
- shell経由で任意コマンドを実行しません

外部コマンドを使う場合も、固定引数の `exec.CommandContext` で実行します。
