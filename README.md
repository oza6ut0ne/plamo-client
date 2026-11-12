# plamo-client

[![PkgGoDev](https://pkg.go.dev/badge/github.com/oza6ut0ne/plamo-client)](https://pkg.go.dev/github.com/oza6ut0ne/plamo-client)
[![Release](https://github.com/oza6ut0ne/plamo-client/actions/workflows/release.yml/badge.svg)](https://github.com/oza6ut0ne/plamo-client/actions/workflows/release.yml)

[Ollama](https://ollama.com/) でローカルで動かしている [PLaMo 3 Translate](https://huggingface.co/pfnet/plamo-3-translate-0.8b-GGUF) に OpenAI 互換 API (`/v1/completions`) 経由で翻訳させるクライアントプログラム。

## Getting Started

1. [Ollama](https://ollama.com/download) をインストールする

2. PLaMo 3 Translate のモデルを取得する

    ```console
    $ ollama pull hf.co/pfnet/plamo-3-translate-0.8b-GGUF
    ```

    2B モデルを使いたい場合は次のモデルを取得する。

    ```console
    $ ollama pull hf.co/pfnet/plamo-3-translate-2b-GGUF
    ```

3. plamo-client をインストールする

    ```console
    $ go install github.com/oza6ut0ne/plamo-client@latest
    ```

    もしくはリリースページから[ビルド済みバイナリ](https://github.com/oza6ut0ne/plamo-client/releases/latest)をダウンロードしてもよい。

4. Ollama のサーバーが起動していることを確認する

    既定では `127.0.0.1:11434` で待ち受けている。

5. plamo-client を使って翻訳する

    ```console
    $ plamo-client "A client for the PLaMo 3 Translate running on Ollama"
    Ollama上で動作するPLaMo 3 Translateのクライアント

    $ plamo-client "Ollamaで動作するPLaMo 3 Translateのクライアント"
    PLaMo 3 Translate client running on Ollama
    ```

    plamo-client は 4 で確認した Ollama の OpenAI 互換 API にアクセスして翻訳をする。

## Usage

1. 引数に翻訳する文を指定する

    ```console
    $ plamo-client '引数に翻訳する文を指定する'
    Specify the sentence to translate as an argument.
    ```

2. 標準入力から翻訳する文を渡す

    ```console
    $ echo '標準入力から翻訳する文を渡す' | plamo-client
    Pass a sentence to be translated from standard input
    ```

    パイプやリダイレクトで渡された入力は、全体をひとつのテキストとしてまとめて翻訳する。

3. 引数を指定せずに起動すると、インタラクティブに翻訳する

    ```console
    $ plamo-client
    plamo> 引数に翻訳する文を指定する
    Specify the sentence to translate as an argument.
    plamo> 引数を指定せずに起動すると、インタラクティブに翻訳する
    When launched without arguments, it translates interactively.
    plamo>
    ```

    インタラクティブモードは `<CTRL+D>` もしくは `<CTRL+C>` で終了できる。

    貼り付けた複数行のテキストは、改行を保ったままひとつの入力として翻訳する。

    入力中は次のショートカットが使える。

    | キー | 動作 |
    | --- | --- |
    | `↑` / `↓` | カーソルを前 / 次の行へ移動し、端では入力履歴をたどる |
    | `←` / `→` | カーソルを前 / 次へ移動 |
    | `Home` / `End` | 入力の先頭 / 末尾へ移動 |
    | `Backspace` / `Delete` | 前 / 次の文字を削除 |
    | `<CTRL+A>` / `<CTRL+E>` | 入力の先頭 / 末尾へ移動 |
    | `<CTRL+B>` / `<CTRL+F>` | カーソルを前 / 次へ移動 |
    | `<CTRL+P>` / `<CTRL+N>` | 入力履歴を前 / 次へたどる |
    | `<CTRL+U>` / `<CTRL+K>` | カーソルから行頭 / 行末まで削除 |
    | `<CTRL+L>` | 画面をクリア |
    | `<CTRL+D>` | 入力が空なら終了、そうでなければ次の文字を削除 |

    `-iteration` オプションはインタラクティブモードでは作用しない。

## Options

* `-verbose` デバッグ用のメッセージを表示する
* `-host {HOST}` Ollama のホストを指定する。既定は `127.0.0.1`。
* `-port {PORT}` Ollama のポートを指定する。既定は `11434`。
* `-lang {LANG}` [訳出先の言語](https://huggingface.co/pfnet/plamo-3-translate-2b-GGUF/blob/main/README_ja.md#%E5%AF%BE%E5%BF%9C%E8%A8%80%E8%AA%9E)を指定する。

    特に指定しない場合は自動判定で、翻訳対象の文中の ASCII 文字が75%以上なら日本語への翻訳となり、そうでなければ英語への翻訳となる。
    大文字小文字の区別はしない。

* `-form {FORM}` 訳出文のスタイルを指定する。

    デフォルトは `none` 。有効な値は `none`, `plain`, `polite` の3通り。

* `-2b` 2B モデル (`hf.co/pfnet/plamo-3-translate-2b-GGUF`) を使う。

    `-model` で既定の 0.8B モデルを指定している場合にのみ作用する。

* `-model {MODEL}` 使用するモデル名を指定する。既定は `hf.co/pfnet/plamo-3-translate-0.8b-GGUF`。
* `-n_predict {N}` 生成するトークン数の上限を指定する。既定は `-1` (無制限)。
* `-repeat_penalty {P}` repeat penalty を指定する。既定は `1.0`。
* `-temperature {T}` temperature を指定する。既定は `0.0`。
* `-top_p {P}` top P を指定する。既定は `0.0`。
* `-iteration {COUNT}` 反復翻訳回数を指定する。反復のたびに翻訳先の言語は逆転する。

    デフォルトは0で、1回限りの翻訳をする。
    1以上を指定した場合、その回数、翻訳を反復する。
    -1を指定した場合、翻訳文の履歴に一致する文章が訳出されるまで、反復翻訳を繰り返す。

## Build

```console
$ make build
```

Nix を使う場合は Flake からビルドできる。

```console
$ nix build
```
