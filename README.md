# discord-formatting [![GoDoc](https://godoc.org/github.com/delthas/discord-formatting?status.svg)](https://godoc.org/github.com/delthas/discord-formatting) [![stability-experimental](https://img.shields.io/badge/stability-experimental-orange.svg)](https://github.com/emersion/stability-badges#experimental)

A small Go library for parsing Discord markdown-like messages to an AST.

The goal is to copy the Discord apps behavior as precisely as possible. This is not a general purpose Markdown parser.

## Usage

The API is well-documented in its [![GoDoc](https://godoc.org/github.com/delthas/discord-formatting?status.svg)](https://godoc.org/github.com/delthas/discord-formatting)

### Example

```go
package main
import "github.com/delthas/discord-formatting"

func main() {
    parser := formatting.NewParser(nil)
    ast := parser.Parse("*hi* @everyone <:smile:12345> __what__ **is** `up`?")
    formatting.Walk(ast, func(n formatting.Node, entering bool) {
        switch nn := n.(type) {
        case *formatting.TextNode:
            if entering {
                fmt.Print(nn.Content)
            }
        }
    })
    fmt.Println(formatting.Debug(ast))
}
```

## Status

Used daily in a small-scale deployment.

The API could be slightly changed in backwards-incompatible ways for now.

- [X] Nearly all the formatting
- [ ] Replacing Unicode named emoji with their actual emoji codepoints

## License

MIT
