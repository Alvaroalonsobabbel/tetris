# Terminal Tetris

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Alvaroalonsobabbel/tetris) ![Test](https://github.com/Alvaroalonsobabbel/tetris/actions/workflows/test.yml/badge.svg) ![Latest Release](https://img.shields.io/github/v/release/Alvaroalonsobabbel/tetris?color=blue&label=Latest%20Release)

Play Tetris from the comfort of your terminal!

![example](./doc/example.gif)

⚠️ this assumes you know how to use the terminal! If you don't you can find out how [here](https://www.google.com/search?q=how+to+use+the+terminal).

## Install

For Apple computers with ARM chips you can use the provided installer. For any other OS you'll have to compile the binary yourself.

### ARM (Apple Silicon)

Open the terminal and run:

```bash
curl -sSL https://raw.githubusercontent.com/Alvaroalonsobabbel/tetris/main/bin/install.sh | bash
```

- You'll be required to enter your admin password.
- You might be required to allow the program to run in the _System Settings - Privavacy & Security_ tab.

### Compiling the binary yourself

1. [Install Go](https://go.dev/doc/install)
2. Clone the repo `git clone git@github.com:Alvaroalonsobabbel/tetris.git`
3. CD into the repo `cd tetris`
4. Run the program `make run-tetris`

## Options

Disables Ghost piece.

```bash
tetris -noghost
```

Enables debug logs into `~/.tetrisLog`.

```bash
tetris -debug
```

Prints current version.

```bash
tetris -version
```
