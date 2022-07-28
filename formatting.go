/*
Package formatting is a small Go library for parsing Discord markdown-like messages to an AST.
The goal is to copy the Discord apps behavior as precisely as possible. This is not a general purpose Markdown parser.

Usage

The main entrypoint to the library is the Parser type, along with its NewParser function.
A Parser is used to Parser.Parse a Discord message string into an AST represented by a Node.

A Node is a node of the message tree and has a list of Node children.
There are several types implementing the Node interface, such as TextNode.
The recommended way to consume the AST is to Walk it, with a Walker function that runs
a type switch on the Node type, to display appropriate begin/end formatting, or set internal state.
It is recommended that unknown Node types be ignored by the consumer.

For example, when writing a Discord to IRC bridge, the function passed to Walk would output an IRC bold formatting
character to the output on entering and leaving a BoldNode.

The library currently does not come with official formatters for the message AST.

Debugging

The Debug function can be used to print a node tree in a human-readable format.
*/
package formatting

import (
	"fmt"
	"regexp"
	"strings"
)

const regexpFlagDotAll = "(?s)"

var patternBlockQuote = regexp.MustCompile(regexpFlagDotAll + "^(?: *>>> +(.*)| *> +([^\\n]*\\n?))")
var patternChannelMention = regexp.MustCompile("^<#(\\d+)>")
var patternRoleMention = regexp.MustCompile("^<@&(\\d+)>")
var patternUserMention = regexp.MustCompile("^<@!?(\\d+)>")
var patternSpecialMention = regexp.MustCompile("^@(everyone|here)")

var patternCustomEmoji = regexp.MustCompile("^<(a)?:([a-zA-Z_0-9]+):(\\d+)>")
var patternNamedEmoji = regexp.MustCompile("^:([^\\s:]+?(?:::skin-tone-\\d)?):")
var patternUnescapeEmoticon = regexp.MustCompile("^(¯\\\\_\\(ツ\\)_/¯)")
var patternTimestamp = regexp.MustCompile("^<t:(-?\\d{1,17})(?::(t|T|d|D|f|F|R))?>")
var patternURL = regexp.MustCompile("^(https?://[^\\s<]+[^<.,:;\"')\\]\\s])")
var patternMaskedLink = regexp.MustCompile("^(\\[(?:\\[[^]]*]|[^]])*](?:[^\\[]*])?)\\(\\s*<?((?:[^\\s\\\\]|\\\\.)*?)>?(?:\\s+['\"]([\\s\\S]*?)['\"])?\\s*\\)")
var patternURLNoEmbed = regexp.MustCompile("^<(https?://[^\\s<]+[^<.,:;\"')\\]\\s])>")
var patternSoftHyphen = regexp.MustCompile("^\\x{00AD}")
var patternSpoiler = regexp.MustCompile("^\\|\\|([\\s\\S]+?)\\|\\|")
var patternListItem = regexp.MustCompile("^([^\\S\\r\\n]*)[*-][ \\s]+(.*)([\\n|$])?") // replaced '?' with '+'
var patternHeaderItem = regexp.MustCompile("^(\\s*(#+)[ \\t](.*) *)(?:\\n|$)")

var patternBold = regexp.MustCompile("^(\\*\\*([\\s\\S]+?)\\*\\*)(?:[^*]|$)")
var patternUnderline = regexp.MustCompile("^(__([\\s\\S]+?)__)(?:[^_]|$)")
var patternStrikethrough = regexp.MustCompile("^~~(\\S|\\S[\\s\\S]*?\\S)~~")
var patternNewline = regexp.MustCompile("^(?:\\n *)*\\n")
var patternText = regexp.MustCompile("^([\\s\\S]+?)(?:[^0-9A-Za-z\\s\\x{00c0}-\\x{ffff}]|\\n| {2,}\\n|\\w+:\\S|$)")
var patternEscape = regexp.MustCompile("^\\\\([^0-9A-Za-z\\s])")
var patternItalics = regexp.MustCompile("^(\\b_((?:__|\\\\[\\s\\S]|[^\\\\_])+?)_\\b)|^(\\*((?:\\*\\*|[^\\s*])(?:\\*\\*|\\s+(?:[^*\\s]|\\*\\*)|[^\\s*])*?)\\*)(?:[^*]|$)")

var patternCodeBlock = regexp.MustCompile(regexpFlagDotAll + "^```(?:([\\w+\\-.]+?)?(\\s*\\n))?([^\\n].*?)\\n*```")
var patternCodeInline = regexp.MustCompile(regexpFlagDotAll + "^``([^`]*)``|^`([^`]*)`")

// var patternHookedLink = regexp.MustCompile("^\\$\\[((?:\\[[^]]*]|[^]]|](?=[^\\[]*]))*)?]\\(\\s*<?((?:[^\\s\\\\]|\\\\.)*?)>?(?:\\s+['\"]([\\s\\S]*?)['\"])?\\s*\\)")

/*
Parser is an immutable object that can parse Discord messages into an AST.

A Parser should never be created manually, and should be created with the NewParser function instead.
*/
type Parser struct {
	rules []rule
}

/*
Node is a node in the Discord message tree. A Node has an ordered list of Children.

The Node type is implemented by many types, such as TextNode. It is recommended to type switch
over the Node to run specific processing depending on the node type.

Some Node types will never have children, and are called leaf nodes in the documentation.

An AST can be visited with Walk, or be printed as a debug human-readable message with Debug.
*/
type Node interface {
	Children() []Node
	addChild(node Node)
}

type node struct {
	children []Node
}

/*
Children returns the list of Children of a Node. This list should not be modified.
*/
func (n *node) Children() []Node {
	return n.children
}
func (n *node) addChild(node Node) {
	n.children = append(n.children, node)
}

/*
TextNode is the most basic leaf Node, containing text.

A TextNode does not mean unformatted text per se. For example, a TextNode that is a child of a BoldNode should be
displayed in bold, whereas a standalone TextNode could be unformatted text.
*/
type TextNode struct {
	node
	Content string
}

/*
BlockQuoteNode is a Node that introduces a block quote (possibly with a multi-line content).
It is usually input in Discord with >>>.
*/
type BlockQuoteNode struct {
	node
}

/*
CodeNode is a Node that introduces a code excerpt (either inline or in a code block).
It is usually input in Discord with ` or ```.

Non-inline code nodes can have an optional Language.
*/
type CodeNode struct {
	node
	Content  string
	Language string
}

/*
SpoilerNode is a Node that contains a spoiler.
It is usually input in Discord with ||.
*/
type SpoilerNode struct {
	node
}

/*
URLNode is a leaf Node that contains a URL.
*/
type URLNode struct {
	node
	URL string
	// Mask is an optional description of the link, found in masked links only.
	Mask string
}

/*
EmojiNode is a leaf Node that represents a custom Discord emoji.
It is usually represented in Discord with <a:text:id> or <:text:id>.
*/
type EmojiNode struct {
	node
	Animated bool
	Text     string
	ID       string
}

/*
ChannelMentionNode is a leaf Node that represents a mention of a channel.
It is usually represented in Discord with <#id>.
*/
type ChannelMentionNode struct {
	node
	ID string
}

/*
RoleMentionNode is a leaf Node that represents a mention of a role.
It is usually represented in Discord with <@&id>.
*/
type RoleMentionNode struct {
	node
	ID string
}

/*
UserMentionNode is a leaf Node that represents a mention of a user.
It is usually represented in Discord with <@!id>.
*/
type UserMentionNode struct {
	node
	ID string
}

/*
SpecialMentionNode is a leaf Node that represents a mention of a special group of users.
Currently, this is either (@) everyone or (@) here, but might contain more targets in the future.
It is usually represented in Discord with @mention.
*/
type SpecialMentionNode struct {
	node
	Mention string
}

/*
TimestampNode is a leaf Node that represents a timestamp, displayed in the local client time.
It is usually represented in Discord with <t:stamp:suffix>.
*/
type TimestampNode struct {
	node
	Stamp  string
	Suffix string
}

/*
HeaderNode is a Node that represents a Markdown header.
It is usually represented in Discord with: # header.

This node is not parsed by default and is currently used in Discord only for the first post of forums.
*/
type HeaderNode struct {
	node
	// Level is the number of hashes (#) of that header
	Level int
}

/*
BulletListNode is a Node that represents a Markdown list.
It is usually represented in Discord with: * my list.

This node is not parsed by default and is currently used in Discord only for the first post in forums.
*/
type BulletListNode struct {
	node
	NestedLevel     int
	IncludesNewline bool
}

/*
BoldNode is a Node that contains content that should be displayed in bold.
It is usually represented in Discord with **bold**.
*/
type BoldNode struct {
	node
}

/*
UnderlineNode is a Node that contains content that should be displayed with an underline.
It is usually represented in Discord with __underline__.
*/
type UnderlineNode struct {
	node
}

/*
ItalicsNode is a Node that contains content that should be displayed in italics.
It is usually represented in Discord with *italics*.
*/
type ItalicsNode struct {
	node
}

/*
StrikethroughNode is a Node that contains content that should be displayed with a strikethrough line.
It is usually represented in Discord with ~~strikethrough~~.
*/
type StrikethroughNode struct {
	node
}

type parseSpec struct {
	node     Node
	matchEnd int
	start    int
	end      int
}
type rule struct {
	pattern    *regexp.Regexp
	block      bool
	parser     func(match match) parseSpec
	blockQuote bool
}
type match struct {
	parser *Parser
	match  string
	groups []int
}

func (m *match) group(i int) string {
	start := m.groups[i*2]
	if start == -1 {
		return ""
	}
	end := m.groups[i*2+1]
	return m.match[start:end]
}

func (m *match) start(i int) int {
	return m.groups[i*2]
}

func (m *match) end(i int) int {
	return m.groups[i*2+1]
}

/*
ParserOptions is a configuration object used for creating a Parser with NewParser.

DefaultParserOptions contains the default options that should be used for parsing. An empty ParserOptions is not the same as DefaultParserOptions!
*/
type ParserOptions struct {
	EnableBlockQuote    bool
	EnableMaskedLinks   bool
	EnableMentions      bool
	EnableForumMarkdown bool
}

/*
DefaultParserOptions is the default parser configurations for usual message parsing.
It should be used for most use cases.
*/
var DefaultParserOptions = ParserOptions{
	EnableBlockQuote: true,
	EnableMentions:   true,
}

/*
NewParser creates a new parser from a ParserOptions configuration.

The options parameter should be DefaultParserOptions / nil in most cases.
As a special case, passing nil is equivalent to passing DefaultParserOptions.

Passing an empty ParserOptions struct is not the same as passing DefaultParserOptions / nil.
This should be avoided unless you do want a ParserOptions with all fields set to false.

The Parser returned by NewParser can be reused for parsing multiple concurrent messages. There is no internal state.
*/
func NewParser(options *ParserOptions) *Parser {
	if options == nil {
		options = &DefaultParserOptions
	}

	rules := make([]rule, 0, 16)
	rules = append(rules, rule{
		pattern: patternSoftHyphen,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &TextNode{
					Content: "",
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternEscape,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &TextNode{
					Content: match.group(1),
				},
			}
		},
	})
	if options.EnableBlockQuote {
		rules = append(rules, rule{
			pattern: patternBlockQuote,
			block:   true,
			parser: func(match match) parseSpec {
				var i int
				if len(match.group(1)) > 0 {
					i = 1
				} else {
					i = 2
				}
				return parseSpec{
					node:  &BlockQuoteNode{},
					start: match.start(i),
					end:   match.end(i),
				}
			},
			blockQuote: true,
		})
	}
	rules = append(rules, rule{
		pattern: patternCodeBlock,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &CodeNode{
					Content:  match.group(3),
					Language: match.group(1),
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternCodeInline,
		parser: func(match match) parseSpec {
			i := 1
			if len(match.group(2)) > 0 {
				i = 2
			}
			return parseSpec{
				node: &CodeNode{
					Content: match.group(i),
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternSpoiler,
		parser: func(match match) parseSpec {
			return parseSpec{
				node:  &SpoilerNode{},
				start: match.start(1),
				end:   match.end(1),
			}
		},
	})
	if options.EnableMaskedLinks {
		rules = append(rules, rule{
			pattern: patternMaskedLink,
			parser: func(match match) parseSpec {
				// intentionally not implementing the pathological masked link attack workaround here.
				mask := match.group(1)
				mask = mask[1 : len(mask)-1]
				return parseSpec{
					node: &URLNode{
						URL:  match.group(2),
						Mask: mask,
					},
				}
			},
		})
	}
	rules = append(rules, rule{
		pattern: patternURLNoEmbed,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &URLNode{
					URL: match.group(1),
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternURL,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &URLNode{
					URL: match.group(1),
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternCustomEmoji,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &EmojiNode{
					Animated: len(match.group(1)) > 0,
					Text:     match.group(2),
					ID:       match.group(3),
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternNamedEmoji,
		parser: func(match match) parseSpec {
			emojiName := match.group(0)
			// TODO: parse the emoji data into the actual unicode emoji
			return parseSpec{
				node: &TextNode{
					Content: emojiName,
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternUnescapeEmoticon,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &TextNode{
					Content: match.group(1),
				},
			}
		},
	})
	if options.EnableMentions {
		rules = append(rules, rule{
			pattern: patternChannelMention,
			parser: func(match match) parseSpec {
				return parseSpec{
					node: &ChannelMentionNode{
						ID: match.group(1),
					},
				}
			},
		})
		rules = append(rules, rule{
			pattern: patternRoleMention,
			parser: func(match match) parseSpec {
				return parseSpec{
					node: &RoleMentionNode{
						ID: match.group(1),
					},
				}
			},
		})
		rules = append(rules, rule{
			pattern: patternUserMention,
			parser: func(match match) parseSpec {
				return parseSpec{
					node: &UserMentionNode{
						ID: match.group(1),
					},
				}
			},
		})
		rules = append(rules, rule{
			pattern: patternSpecialMention,
			parser: func(match match) parseSpec {
				return parseSpec{
					node: &SpecialMentionNode{
						Mention: match.group(1),
					},
				}
			},
		})
	}
	// TODO: dynamic unicodeEmoji pattern
	rules = append(rules, rule{
		pattern: patternTimestamp,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &TimestampNode{
					Stamp:  match.group(1),
					Suffix: match.group(2),
				},
			}
		},
	})
	if options.EnableForumMarkdown {
		rules = append(rules, rule{
			pattern: patternHeaderItem,
			block:   true,
			parser: func(match match) parseSpec {
				n := 1
				if len(match.group(2)) > 0 {
					n = len(match.group(2))
				}
				return parseSpec{
					node: &HeaderNode{
						Level: n,
					},
					start:    match.start(3),
					end:      match.end(3),
					matchEnd: match.end(1),
				}
			},
		})
		rules = append(rules, rule{
			pattern: patternListItem,
			parser: func(match match) parseSpec {
				level := 1
				if len(match.group(1)) > 0 {
					level = 2
				}
				return parseSpec{
					node: &BulletListNode{
						NestedLevel:     level,
						IncludesNewline: len(match.group(3)) > 0,
					},
					start: match.start(2),
					end:   match.end(2),
				}
			},
		})
	}
	rules = append(rules, rule{
		pattern: patternNewline,
		block:   true,
		parser: func(match match) parseSpec {
			return parseSpec{
				node: &TextNode{
					Content: "\n",
				},
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternBold,
		parser: func(match match) parseSpec {
			return parseSpec{
				node:     &BoldNode{},
				start:    match.start(2),
				end:      match.end(2),
				matchEnd: match.end(1),
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternUnderline,
		parser: func(match match) parseSpec {
			return parseSpec{
				node:     &UnderlineNode{},
				start:    match.start(2),
				end:      match.end(2),
				matchEnd: match.end(1),
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternItalics,
		parser: func(match match) parseSpec {
			content := 2
			if len(match.group(4)) > 0 {
				content = 4
			}
			total := 1
			if len(match.group(3)) > 0 {
				total = 3
			}
			return parseSpec{
				node:     &ItalicsNode{},
				start:    match.start(content),
				end:      match.end(content),
				matchEnd: match.end(total),
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternStrikethrough,
		parser: func(match match) parseSpec {
			return parseSpec{
				node:  &StrikethroughNode{},
				start: match.start(1),
				end:   match.end(1),
			}
		},
	})
	rules = append(rules, rule{
		pattern: patternText,
		parser: func(match match) parseSpec {
			// TODO: replace the passed string with replaceEmojiSurrogates,
			// then parse it with rules={namedEmojiRule, patternTextRule}
			return parseSpec{
				node: &TextNode{
					Content: match.group(1),
				},
				matchEnd: match.end(1),
			}
		},
	})
	return &Parser{
		rules: rules,
	}
}

/*
Parse parses the passed Discord message into an AST. The root Node of the tree is returned.

The root Node is always a private node structure that contains a list of Node children.

Walk can be used to process the AST returned by this tree.
*/
func (p *Parser) Parse(source string) Node {
	remainingParses := make([]parseSpec, 0, 16)
	topLevelRootNode := &node{}
	lastCapture := ""

	if len(source) > 0 {
		remainingParses = append(remainingParses, parseSpec{
			node:  topLevelRootNode,
			start: 0,
			end:   len(source),
		})
	}

	// TODO: do not nest multiple block quotes
	blockQuoteEnd := 0

	for len(remainingParses) > 0 {
		builder := remainingParses[len(remainingParses)-1]
		remainingParses = remainingParses[:len(remainingParses)-1]
		if builder.start >= builder.end {
			break
		}
		inspectionSource := source[builder.start:builder.end]
		offset := builder.start

		var rule rule
		var groups []int
		for _, r := range p.rules {
			if r.block && lastCapture != "" && !strings.HasSuffix(lastCapture, "\n") {
				continue
			}
			if r.blockQuote && builder.start < blockQuoteEnd {
				continue
			}
			g := r.pattern.FindStringSubmatchIndex(inspectionSource)
			if g == nil {
				continue
			}
			rule = r
			groups = g
			break
		}
		if len(groups) == 0 {
			panic(fmt.Sprintf("failed to find rule to match source: %s", source))
		}

		newBuilder := rule.parser(match{
			parser: p,
			match:  inspectionSource,
			groups: groups,
		})
		if newBuilder.matchEnd == 0 {
			newBuilder.matchEnd = groups[1]
		}
		parent := builder.node
		parent.addChild(newBuilder.node)

		matcherSourceEnd := newBuilder.matchEnd + offset
		if matcherSourceEnd != builder.end {
			remainingParses = append(remainingParses, parseSpec{
				node:  parent,
				start: matcherSourceEnd,
				end:   builder.end,
			})
		}

		if newBuilder.start != 0 || newBuilder.end != 0 {
			newBuilder.start += offset
			newBuilder.end += offset
			remainingParses = append(remainingParses, newBuilder)
		}
		if rule.blockQuote {
			blockQuoteEnd = newBuilder.end
		}

		lastCapture = inspectionSource[:newBuilder.matchEnd]
	}

	return topLevelRootNode
}

/*
Walker is the visiting callback used by Walk.
*/
type Walker func(n Node, entering bool)

/*
Walk walks the passed AST represented by its root Node, with a Walker function.
The walk algorithm parses the tree in a depth-first manner.

The Walker function is called on entering and leaving each node.
*/
func Walk(n Node, w Walker) {
	w(n, true)
	for _, child := range n.Children() {
		Walk(child, w)
	}
	w(n, false)
}

/*
Debug prints an AST to a human-readable string for debugging purposes.

The format of this string is unspecified and should not be parsed.
For programmatically processing an AST, use Walk.
*/
func Debug(n Node) string {
	noSpace := true
	var sb strings.Builder
	Walk(n, func(nn Node, entering bool) {
		if entering {
			if noSpace {
				noSpace = false
			} else {
				sb.WriteString(" ")
			}
			sb.WriteString("[")
			switch n := nn.(type) {
			case *TextNode:
				sb.WriteString(fmt.Sprintf("text %q", n.Content))
			case *BlockQuoteNode:
				sb.WriteString(fmt.Sprintf("blockquote"))
			case *CodeNode:
				sb.WriteString(fmt.Sprintf("code %q %q", n.Language, n.Content))
			case *SpoilerNode:
				sb.WriteString(fmt.Sprintf("spoiler"))
			case *URLNode:
				sb.WriteString(fmt.Sprintf("url %q %q", n.Mask, n.URL))
			case *EmojiNode:
				sb.WriteString(fmt.Sprintf("emoji %v %q %q", n.Animated, n.Text, n.ID))
			case *ChannelMentionNode:
				sb.WriteString(fmt.Sprintf("channelmention %q", n.ID))
			case *RoleMentionNode:
				sb.WriteString(fmt.Sprintf("rolemention %q", n.ID))
			case *UserMentionNode:
				sb.WriteString(fmt.Sprintf("usermention %q", n.ID))
			case *SpecialMentionNode:
				sb.WriteString(fmt.Sprintf("specialmention %q", n.Mention))
			case *TimestampNode:
				sb.WriteString(fmt.Sprintf("timestamp %q %q", n.Stamp, n.Suffix))
			case *HeaderNode:
				sb.WriteString(fmt.Sprintf("header %d", n.Level))
			case *BulletListNode:
				sb.WriteString(fmt.Sprintf("list %d %v", n.NestedLevel, n.IncludesNewline))
			case *BoldNode:
				sb.WriteString(fmt.Sprintf("bold"))
			case *UnderlineNode:
				sb.WriteString(fmt.Sprintf("underline"))
			case *ItalicsNode:
				sb.WriteString(fmt.Sprintf("italics"))
			case *StrikethroughNode:
				sb.WriteString(fmt.Sprintf("strikethrough"))
			case *node:
				noSpace = true
			default:
				panic(fmt.Sprintf("invalid node type: %T", n))
			}
		} else {
			sb.WriteString("]")
		}
	})
	return sb.String()
}
