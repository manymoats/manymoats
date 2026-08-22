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

### `manymoats credits`

What your free AI credits actually pay for, and what only looks free.

Sorted by what dies soonest, with the daily spend it would take to use a credit
before it expires. `manymoats credits covers <model>` answers the question that
actually costs people money: the same model often has two doors, and only one of
them is on your credit.

Nothing here is a guess. What can't be checked says `unknown`, and every answer
prints its own age — a claim about what a plan includes goes quiet after 14 days,
a published terms page lasts 180.

Your own figures live in `~/.manymoats/credits/holdings.json`. They are never
uploaded and never ship in the binary.

## Update

```bash
brew update && brew upgrade manymoats
```

`brew upgrade` on its own reads a cached copy of the tap and will tell you the
version you already have is the newest one. `brew update` fetches first.

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
