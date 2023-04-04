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

### Gap Lines

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

### Indentation

You can also set the indent size (number of spaces) using `--indent`. Yam uses 2-space indentation by default.

```shell
yam a.yaml --indent 4
```

### Using a config file

Yam will also look for a `.yam.yaml` file in the current working directory as a source of configuration. Using a config file is optional. CLI flag values take priority over config file values. The config file can be used to configure `indent` and `gap` values only.

Example `.yam.yaml`:

```yaml
indent: 4   # Defaults to 2

gap:        # Defaults to none
- "."
- ".users"
```

## Yam's Encoder

Yam has a special YAML encoder it uses to handle formatting as it writes out YAML bytes. This encoder is configurable.

Yam bases its encoder on the YAML encoder from https://github.com/go-yaml/yaml, and uses this library's `yaml.Node` type as the input to encoding operations.

This means you're able to decode data using https://github.com/go-yaml/yaml, modify data as needed, and then encode the `yaml.Node` using **yam's encoder** instead. This is nifty if you want to write YAML data that's correctly formatted from the beginning.

For example, before:

```go
import (
    // ...
    "gopkg.in/yaml.v3"
)

func someFunction(myData yaml.Node, w io.Writer) {
    enc := yaml.NewEncoder(w)

    // use the encoder!
	_ = enc.Encode(myData)
}
```

And after:

```go
import (
    // ...
    "github.com/chainguard-dev/yam/pkg/yam/formatted"
)

func someFunction(myData yaml.Node, w io.Writer) {
    enc := formatted.NewEncoder(w)

    // use the encoder!
    _ = enc.Encode(myData)
}
```

### Configuring the encoder

Yam's encoder has chainable methods that can be used for configuring its options.

For example:

```go
enc := formatted.NewEncoder(w).SetIndent(2).SetGapExpressions(".", ".users")
```

#### Configuring the encoder with `.yam.yaml`

You can also tell the encoder to behave similarly to the `yam` command, in that a `.yam.yaml` is automatically read if available.

```go
enc := formatted.NewEncoder(w).AutomaticConfig()
```