# manymoats

Terminal tools. One binary, one install.

```bash
brew install manymoats/tap/manymoats
```

Then:

```bash
manymoats
```

## What's in it

### `manymoats orch`

A status bar for your coding agents. Who's working right now, which model, which
account, what it's costing you.

It reads Claude Code session files and the Cursor workspace database on your own
machine and shows you what it finds. Nothing leaves your computer — there is no
server, no telemetry, no account.

Colour tells you *who*: each provider keeps its own. Treatment tells you *what*:
a live agent breathes, a stalled one goes flat. Agents that aren't doing anything
aren't shown.

```
1-4   views          m  minimal
n     name mode      h  machines
a     show idle      q  quit
```

## Update

```bash
brew upgrade manymoats
```

## Build from source

Needs Go 1.22+.

```bash
git clone https://github.com/manymoats/manymoats
cd manymoats
go build -o manymoats .
./manymoats
```

`go test ./...` runs the suite.

## Config

Lives in `~/.orch/`. Written by `manymoats orch setup`; nothing there is required
to run.

## License

MIT
