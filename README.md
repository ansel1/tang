tang
====

A command line tool for summarizing the results of go test, in real time.

**Status:** Phase 1 Complete ✅ - See SPEC.md for implementation phases.

Installation
------------

    go install github.com/ansel1/tang

Usage
-----

Pipe `go test -json` into `tang`:

    go test -json ./... | tang

...or, capture the output `go test -json` to a file, then summarize it:

    go test -json ./... > test.out
    tang -f test.out

Advanced usage, good for CI, handles some edge cases:

    set -euo pipefail
    go test -json ./... 2>&1 | tang

To see help and available options, like highlighting slow tests:

    tang -h

## Flags

| Flag | Default | Description                              |
| ---- | ------- | ---------------------------------------- |
| `-f` | `""`    | Read from `<filename>` instead of stdin |
| `-outfile` | `""` | Save all input to the specified file |
| `-jsonfile` | | `""` | Output the raw json output to a file |
| `-junitfile` | | `""` | Output junit xml output to a file |
| `-include-passed` | `false` | Include passed tests in summary |
| `-include-slow` | `false` | Include slow tests in summary |
| `-slow-threshold` | `1s` | Set slow test threshold |
| `-include-skipped` | `true` | Include skipped tests in summary |
| `-notty` | `false` | Don't open a tty (not typically needed) |
| `-replay` | `false` | Use with -f, replay events with pauses to simulate original test run |
| `-rate` | `1` | Use with -replay, set rate to replay<br>Defaults to 1 (original speed), 0.5 = double speed, 0 = no pauses |
| `-renderer` | `default` | Select the renderer (default, simple) |

Anything piped to `tang` which doesn't appear to be `go test -json` output is just
passed directly to output, so you can pipe any output which has test output embedded in it:

    make all | gotestpretty

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
