tang
====

A command line tool for summarizing the results of go test, in real time.

Installation
------------

    go install github.com/ansel1/tang

Usage
-----

### Summarize go test output

Run `go test` through `tang` to get a real-time, summarized view of your tests:

    tang test ./...

You can pass any `go test` flags after `test`:

    tang test -v -count 1 -run TestMyFunc ./...

### Alternative Usage: Piped or File-based

Pipe `go test -json` into `tang`:

    go test -json ./... | tang

...or, capture the output to a file, then summarize it:

    go test -json ./... > test.out
    tang -f test.out

Advanced usage for CI:

    set -euo pipefail
    go test -json ./... 2>&1 | tang

To see help and available options:

    tang -h

## Flags

Tang's own flags go before the `test` subcommand.  Everything after `test` is passed through to `go test`:

    tang -notty test -count 1 ./...

The `-v` flag after `test` is special: it enables tang's verbose output *and* is passed through to `go test`:

    tang test -v ./...

| Flag | Default | Description                              |
| ---- | ------- | ---------------------------------------- |
| `-f` | `""`    | Read from `<filename>` instead of stdin (incompatible with `test` subcommand) |
| `-outfile` | `""` | Save all input to the specified file |
| `-jsonfile` | `""` | Output the raw json output to a file |
| `-junitfile` | `""` | Output junit xml output to a file |
| `-include-skipped` | `false` | Include skipped tests in summary |
| `-include-slow` | `false` | Include slow tests in summary |
| `-slow-threshold` | `10s` | Duration threshold for slow test detection |
| `-notty` | `false` | Don't open a tty, output to stdout |
| `-v` | `false` | Verbose output (show all test output in non-tty mode) |
| `-replay` | `false` | Replay events from file (incompatible with `test` subcommand) |
| `-rate` | `1` | Replay rate multiplier (incompatible with `test` subcommand) |
| `-no-color` | `false` | Disable all ANSI color and style escape codes |

The `NO_COLOR` environment variable is also respected. Setting `NO_COLOR=1` (or any non-empty value) has the same effect as `-no-color`. See [no-color.org](https://no-color.org) for details.

Anything piped to `tang` which doesn't appear to be `go test -json` output is just
passed directly to output, so you can pipe any output which has test output embedded in it:

    make all | tang

Why?
----

Other tools exist that do similar stuff, but most don't give real time feedback while the tests are running.  Or are
hard to use when the commands are embedded in build scripts or makefiles.  Or I just preferred a different style of formatting.  `tang`'s formatting is inspired by JetBrains Goland's test runner UI.

Some other tools you can try:

- [tparse](https://github.com/mfridman/tparse)
- [gotestsum](https://github.com/gotestyourself/gotestsum)
- [gotestfmt](https://github.com/GoTestTools/gotestfmt?tab=readme-ov-file)

License
-------

This project is licensed under the terms of the MIT license.

Contributing
------------

**Debugging**

    dlv debug --headless --api-version=2 --listen=127.0.0.1:43000 . -- [args]
