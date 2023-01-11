# yam 🍠

A sweet little formatter for YAML

## Installation

```shell
go install github.com/chainguard-dev/yam@latest
```

## Usage

### Format...

```shell
yam a.yaml
```

### Lint...

Just add `--lint` to the command:

```shell
yam a.yaml --lint
```

## Formatting/Linting Options

To expect a gap (empty line) in between child elements of a given node, just pass a `yq`-style path to the node, using `--gap`. You can use this flag as many times as needed.

```shell
yam a.yaml --gap '.'
```

```shell
yam a.yaml --gap '.foo.bar'
```

```shell
yam a.yaml --gap '.people[].address'
```

```shell
yam a.yaml --gap '.recipes[0].ingredients'
```

```shell
yam a.yaml --gap '.types.*.inputs'
```