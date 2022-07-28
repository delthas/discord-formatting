package formatting

import (
	"fmt"
	"testing"
)

func test(t *testing.T, text string, want string) {
	got := Debug(NewParser(&ParserOptions{
		EnableBlockQuote:    true,
		EnableMaskedLinks:   true,
		EnableMentions:      true,
		EnableForumMarkdown: true,
	}).Parse(text))
	if got != want {
		t.Errorf("error parsing %q: want %q, got %q", text, want, got)
	}
}

func TestFormatting(t *testing.T) {
	test(t, ">>> hi", `[[blockquote [text "hi"]]]`)
	test(t, "<#1234>", `[[channelmention "1234"]]`)
	test(t, "<@&1234>", `[[rolemention "1234"]]`)
	test(t, "<@!1234>", `[[usermention "1234"]]`)
	test(t, "@everyone", `[[specialmention "everyone"]]`)
	test(t, "@here", `[[specialmention "here"]]`)
	test(t, "<a:that:1234>", `[[emoji true "that" "1234"]]`)
	test(t, "<:that:1234>", `[[emoji false "that" "1234"]]`)
	test(t, ":grin:", `[[text ":grin:"]]`)
	test(t, `¯\_(ツ)_/¯`, `[[text "¯\\_(ツ)_/¯"]]`) // double \\ because of go %q
	test(t, `<t:1234567890:t>`, `[[timestamp "1234567890" "t"]]`)
	test(t, `https://example.com`, `[[url "" "https://example.com"]]`)
	test(t, `[example](https://example.com)`, `[[url "example" "https://example.com"]]`)
	test(t, `<https://example.com>`, `[[url "" "https://example.com"]]`)
	test(t, "\u00AD", `[[text ""]]`)
	test(t, "||flushed||", `[[spoiler [text "flushed"]]]`)
	test(t, "- list", `[[list 1 false [text "list"]]]`)
	test(t, "### header", `[[header 3 [text "header"]]]`)
	test(t, "**bold**", `[[bold [text "bold"]]]`)
	test(t, "*hi*", `[[italics [text "hi"]]]`)
	test(t, "_hi_", `[[italics [text "hi"]]]`)
	test(t, "__hi__", `[[underline [text "hi"]]]`)
	test(t, "~~hi~~", `[[strikethrough [text "hi"]]]`)
	test(t, "\n \n", `[[text "\n"]]`)
	test(t, "hi", `[[text "hi"]]`)
	test(t, `\*hi\*`, `[[text "*"] [text "hi"] [text "*"]]`)
	test(t, "`hello`", `[[code "" "hello"]]`)
	test(t, "```sx\nhello\n```", `[[code "sx" "hello"]]`)
}

func TestSimple(t *testing.T) {
	p := NewParser(nil)
	ast := p.Parse("*hi\u00ADmom__underline__* ~~strike~~ \\~~strike~~! `my code` \n```shell\nmy epic code\nyes\n```")
	fmt.Println(Debug(ast))
}

func Example() {
	parser := NewParser(nil)
	ast := parser.Parse("*hi* @everyone <:smile:12345> __what__ **is** `up`?")
	Walk(ast, func(n Node, entering bool) {
		switch nn := n.(type) {
		case *TextNode:
			if entering {
				fmt.Print(nn.Content)
			}
		}
	})
	fmt.Println(Debug(ast))
}
