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
Commands:
help [<command>...]
    Show help.

time [<flags>] <file>
    Move track in time

info <file>
    Show info about GPX file

show [<flags>] [<files>...]
    Show GPX file on the map in the browser

concat [<flags>] [<files>...]
    Concat multiple GPX tracks in one

reduce [<flags>] <files>...
    Reduce number of points in track to simplify shape

route [<flags>] <file>
    Build optimal route through waypoints

runnerup list [<flags>] <db>
    List latest activities

runnerup export [<flags>] <db> <activity-id> [<output>]
    Export activity as GPX.
```