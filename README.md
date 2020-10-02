# statectl

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/observatorium/statectl)
[![Latest Release](https://img.shields.io/github/release/observatorium/statectl.svg?style=flat-square)](https://github.com/observatorium/statectl/releases/latest)
[![CI](https://github.com/observatorium/statectl/workflows/go/badge.svg)](https://github.com/observatorium/statectl/actions?query=workflow%3Ago)
[![Go Report Card](https://goreportcard.com/badge/github.com/observatorium/statectl)](https://goreportcard.com/report/github.com/observatorium/statectl)

CLI oriented to manage and control your deployment states.

## Example Scenario

Statectl is useful when your deployment model is seprated into 

## Requirements

* Go 1.14+
* Linux or MacOS

## Installing

```shell
go get github.com/observatorium/statectl && go mod tidy
```

or via [bingo](github.com/bwplotka/bingo) if want to pin it:

```shell
bingo get -u github.com/observatorium/statectl
```

## Usage

[embedmd]:# (statectl-help.txt $)
```$
usage: statectl [<flags>] <command> [<args> ...]

Control state of your deployments.

Flags:
  -h, --help               Show context-sensitive help (also try --help-long and
                           --help-man).
      --version            Show application version.
      --log.level=info     Log filtering level.
      --log.format=logfmt  Log format to use. Possible options: logfmt or json.

Commands:
  help [<command>...]
    Show help.

  propose
    Propose change of cluster state.


```

## Contributing

Any contributions are welcome! Just use GitHub Issues and Pull Requests as usual.
We follow [Thanos Go coding style](https://thanos.io/contributing/coding-style-guide.md/) guide.

## Initial Author

[@bwplotka](https://bwplotka.dev)
