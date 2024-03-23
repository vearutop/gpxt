# gpxt

[![Build Status](https://github.com/vearutop/gpxt/workflows/test-unit/badge.svg)](https://github.com/vearutop/gpxt/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/vearutop/gpxt/branch/master/graph/badge.svg)](https://codecov.io/gh/vearutop/gpxt)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/vearutop/gpxt)
[![Time Tracker](https://wakatime.com/badge/github/vearutop/gpxt.svg)](https://wakatime.com/badge/github/vearutop/gpxt)
![Code lines](https://sloc.xyz/github/vearutop/gpxt/?category=code)
![Comments](https://sloc.xyz/github/vearutop/gpxt/?category=comments)

GPX Tool CLI.

## Usage

```
usage: gpxt [<flags>] <command> [<args> ...]


Flags:
  --[no-]help     Show context-sensitive help (also try --help-long and --help-man).
  --[no-]version  Show application version.

Commands:
help [<command>...]
    Show help.

move [<flags>] <file>
    When both new-start and new-end are present, the track would be stretched/shrinked to fit in new boundaries. Otherwise it would be moved to the touch new-start or
    new-end.

info <file>
    Show info about GPX file

show [<flags>] [<files>...]
    Show GPX file on the map in the browser

concat [<flags>] [<files>...]
    Concat multiple GPX tracks in one

cut [<flags>] <files>...
    Remove head and/or tail of a track

reduce [<flags>] <files>...
    Reduce number of points in track to simplify shape

route [<flags>] <file>
    Build optimal route through waypoints

runnerup list [<flags>] <db>
    List latest activities

runnerup export [<flags>] <db> <activity-id> [<output>]
    Export activity as GPX.

```