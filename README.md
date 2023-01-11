# yam 🍠

A sweet little formatter for YAML

## Installation

```shell
go install github.com/chainguard-dev/yam@latest
```

## Usage

### Format...

...all `.yaml` and `.yml` files in the current directory:

```shell
yam
```

...one file:

```shell
yam a.yaml
```

...two files:

```shell
yam a.yaml b.yaml
```

...THREE FILES!!! 😱

```shell
yam a.yaml b.yaml c.yaml
```

### Lint...

Just add `--lint` to the command:

```shell
yam --fix
```

## Formatting/Linting Options

To expect a gap (empty line) in between child elements of a given node, just pass a `yq`-style path to the node, using `--gap`. You can use this flag as many times as needed.

```shell
yam --gap '.'
```

```shell
yam --gap '.foo.bar'
```

```shell
yam --gap '.people[].address'
```

```shell
yam --gap '.recipes[0].ingredients'
```

```shell
yam --gap '.types.*.inputs'
```