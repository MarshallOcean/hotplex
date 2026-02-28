package slack

import (
	"testing"
)

func TestMrkdwnFormatter_Format(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Empty input
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},

		// Headings
		{
			name:     "h1 heading",
			input:    "# Heading 1",
			expected: "*Heading 1*",
		},
		{
			name:     "h2 heading",
			input:    "## Heading 2",
			expected: "*Heading 2*",
		},
		{
			name:     "h3 heading",
			input:    "### Heading 3",
			expected: "*Heading 3*",
		},
		{
			name:     "multiple headings",
			input:    "# H1\n## H2\n### H3",
			expected: "*H1*\n*H2*\n*H3*",
		},
		{
			name:     "heading in code block preserved",
			input:    "```\n# Not a heading\n```",
			expected: "```\n# Not a heading\n```",
		},

		// Lists
		{
			name:     "unordered list with dash",
			input:    "- item 1\n- item 2",
			expected: "• item 1\n• item 2",
		},
		{
			name:     "unordered list with asterisk",
			input:    "* item 1\n* item 2",
			expected: "• item 1\n• item 2",
		},
		{
			name:     "unordered list with plus",
			input:    "+ item 1\n+ item 2",
			expected: "• item 1\n• item 2",
		},
		{
			name:     "ordered list",
			input:    "1. First\n2. Second\n3. Third",
			expected: "• First\n• Second\n• Third",
		},
		{
			name:     "mixed lists",
			input:    "- Item A\n1. Item B\n+ Item C",
			expected: "• Item A\n• Item B\n• Item C",
		},
		{
			name:     "list in code block preserved",
			input:    "```\n- Not a list\n```",
			expected: "```\n- Not a list\n```",
		},

		// Links
		{
			name:     "simple link",
			input:    "[Google](https://google.com)",
			expected: "<https://google.com|Google>",
		},
		{
			name:     "link with special chars",
			input:    "[Example](https://example.com?a=1&b=2)",
			expected: "<https://example.com?a=1&b=2|Example>",
		},
		{
			name:     "multiple links",
			input:    "[Link1](http://a.com) and [Link2](http://b.com)",
			expected: "<http://a.com|Link1> and <http://b.com|Link2>",
		},

		// Bold
		{
			name:     "bold with asterisks",
			input:    "**bold text**",
			expected: "*bold text*", // Slack mrkdwn uses *text* for bold
		},
		{
			name:     "bold with underscores",
			input:    "__bold text__",
			expected: "*bold text*", // Slack mrkdwn uses *text* for bold
		},
		{
			name:     "bold in sentence",
			input:    "This is **bold** text",
			expected: "This is *bold* text", // Slack mrkdwn uses *text* for bold
		},

		// Italic
		{
			name:     "italic with underscore",
			input:    "_italic text_",
			expected: "_italic text_",
		},
		{
			name:     "italic with asterisk",
			input:    "*italic text*",
			expected: "*italic text*", // *text* is bold in Slack, not italic (Markdown uses *text* for italic)
		},
		{
			name:     "italic in sentence",
			input:    "This is _italic_ text",
			expected: "This is _italic_ text",
		},

		// Strikethrough
		{
			name:     "strikethrough",
			input:    "~~deleted text~~",
			expected: "~deleted text~",
		},

		// Blockquotes
		{
			name:     "blockquote",
			input:    "> This is a quote",
			expected: "> This is a quote",
		},
		{
			name:     "multiple blockquote lines",
			input:    "> Line 1\n> Line 2",
			expected: "> Line 1\n> Line 2",
		},

		// Special characters escaping
		{
			name:     "escape ampersand",
			input:    "A & B",
			expected: "A &amp; B",
		},
		{
			name:     "escape less than",
			input:    "A < B",
			expected: "A &lt; B",
		},
		{
			name:     "escape greater than",
			input:    "A > B",
			expected: "A &gt; B",
		},
		{
			name:     "escape mixed",
			input:    "A & B < C > D",
			expected: "A &amp; B &lt; C &gt; D",
		},
		{
			name:     "preserve slack syntax user mention",
			input:    "<@U123>",
			expected: "<@U123>",
		},
		{
			name:     "preserve slack syntax channel mention",
			input:    "<#C123>",
			expected: "<#C123>",
		},
		{
			name:     "preserve slack syntax special mention",
			input:    "<!here>",
			expected: "<!here>",
		},
		{
			name:     "preserve slack syntax link",
			input:    "<https://example.com|text>",
			expected: "<https://example.com|text>",
		},

		// Code blocks
		{
			name:     "inline code preserved",
			input:    "Use `code` here",
			expected: "Use `code` here",
		},
		{
			name:     "inline code with special chars",
			input:    "Use `a & b < c` here",
			expected: "Use `a & b < c` here",
		},
		{
			name:     "code block with special chars",
			input:    "```\ncode & more < test >\n```",
			expected: "```\ncode & more < test >\n```",
		},

		// Complex combinations
		{
			name:     "complex markdown",
			input:    "# Title\n\nThis is **bold** and _italic_ text.\n\n- List item 1\n- List item 2\n\n> A quote\n\n[Link](https://example.com)",
			expected: "*Title*\n\nThis is *bold* and _italic_ text.\n\n• List item 1\n• List item 2\n\n> A quote\n\n<https://example.com|Link>", // Slack link format is <url|text>
		},
	}

	formatter := NewMrkdwnFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.input)
			if result != tt.expected {
				t.Errorf("Format(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_FormatHeadings(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"h1", "# Title", "*Title*"},
		{"h2", "## Title", "*Title*"},
		{"h3", "### Title", "*Title*"},
		{"h4", "#### Title", "*Title*"},
		{"h5", "##### Title", "*Title*"},
		{"h6", "###### Title", "*Title*"},
		{"not heading", "#NoSpace", "#NoSpace"},
		{"not heading hashes", "####### Too Many", "####### Too Many"}, // 7+ hashes treated as too many, not a valid heading
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.convertHeadings(tt.input)
			if result != tt.expected {
				t.Errorf("convertHeadings(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_FormatLists(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"dash list", "- item", "• item"},
		{"asterisk list", "* item", "• item"},
		{"plus list", "+ item", "• item"},
		{"ordered list", "1. First", "• First"},
		{"ordered list multi-digit", "10. Item", "• Item"},
		{"not a list", "-no space", "-no space"},
		{"not a list dot", "1.no space", "1.no space"},
		{"indented list", "  - item", "  • item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.convertLists(tt.input)
			if result != tt.expected {
				t.Errorf("convertLists(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_FormatBlockquotes(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	// Test the full Format function, not convertBlockquotes directly
	// because convertBlockquotes uses BLOCKQUOTE_START marker that gets replaced later
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple quote", "> quote", "> quote"},
		{"quote no space", ">quote", "> quote"},
		{"quote with spaces", ">  multiple spaces", "> multiple spaces"},
		{"not in code", "```\n> quote\n```", "```\n> quote\n```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.input)
			if result != tt.expected {
				t.Errorf("Format(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_EscapeSpecialChars(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ampersand", "A&B", "A&amp;B"},
		{"less than", "A<B", "A&lt;B"},
		{"greater than", "A>B", "A&gt;B"},
		{"all special", "A&B<C>D", "A&amp;B&lt;C&gt;D"},
		{"preserve mention", "<@U123>", "<@U123>"},
		{"preserve channel", "<#C123|general>", "<#C123|general>"},
		{"preserve here", "<!here>", "<!here>"},
		{"preserve channel all", "<!channel>", "<!channel>"},
		{"preserve everyone", "<!everyone>", "<!everyone>"},
		{"preserve link", "<http://example.com|text>", "<http://example.com|text>"},
		{"preserve mailto", "<mailto:test@example.com|email>", "<mailto:test@example.com|email>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.escapeSpecialChars(tt.input)
			if result != tt.expected {
				t.Errorf("escapeSpecialChars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_ConvertLinks(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple link", "[text](http://example.com)", "<http://example.com|text>"},
		{"https link", "[secure](https://example.com)", "<https://example.com|secure>"},
		{"link with path", "[page](http://example.com/path)", "<http://example.com/path|page>"},
		{"link with query", "[search](http://example.com?q=test)", "<http://example.com?q=test|search>"},
		{"multiple links", "[a](http://a.com)[b](http://b.com)", "<http://a.com|a><http://b.com|b>"},
		{"not a link", "[no url", "[no url"},
		{"partial link", "[text]()", "[text]()"}, // Empty URL, should not convert
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.convertLinks(tt.input)
			if result != tt.expected {
				t.Errorf("convertLinks(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_ConvertBold(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"asterisk bold", "**bold**", "*bold*"},
		{"underscore bold", "__bold__", "*bold*"},
		{"bold in text", "hello **world**", "hello *world*"},
		{"multiple bold", "**a** and **b**", "*a* and *b*"},
		{"bold in code", "```**not bold**```", "```**not bold**```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.convertBold(tt.input)
			if result != tt.expected {
				t.Errorf("convertBold(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_ConvertItalic(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"underscore italic", "_italic_", "_italic_"},
		{"asterisk italic", "*italic*", "*italic*"}, // *text* is not converted (it's Markdown italic but Slack bold)
		{"italic in text", "hello _world_", "hello _world_"},
		{"not italic", "*no closing", "*no closing"},
		{"italic in code", "```_not italic_```", "```_not italic_```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.convertItalic(tt.input)
			if result != tt.expected {
				t.Errorf("convertItalic(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMrkdwnFormatter_ConvertStrikethrough(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strikethrough", "~~deleted~~", "~deleted~"},
		{"strikethrough in text", "hello ~~world~~", "hello ~world~"},
		{"multiple strikethrough", "~~a~~ and ~~b~~", "~a~ and ~b~"},
		{"strikethrough in code", "```~~not struck~~```", "```~~not struck~~```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.convertStrikethrough(tt.input)
			if result != tt.expected {
				t.Errorf("convertStrikethrough(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatCodeBlock(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		code     string
		language string
		expected string
	}{
		{"no language", "code", "", "```\ncode\n```"},
		{"go language", "fmt.Println()", "go", "```go\nfmt.Println()\n```"},
		{"python language", "print('hi')", "python", "```python\nprint('hi')\n```"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatCodeBlock(tt.code, tt.language)
			if result != tt.expected {
				t.Errorf("FormatCodeBlock(%q, %q) = %q, want %q", tt.code, tt.language, result, tt.expected)
			}
		})
	}
}

// Test edge cases and potential bugs
func TestMrkdwnFormatter_EdgeCases(t *testing.T) {
	formatter := NewMrkdwnFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Nested formatting
		{
			name:     "bold and italic combined",
			input:    "***bold italic***",
			expected: "**bold italic**", // ** makes *bold*, then *...* is left as-is since it looks like converted bold
		},
		{
			name:     "heading with bold",
			input:    "# **Heading**",
			expected: "**Heading**", // Heading converts # **Heading** to ***Heading***, bold converts ***Heading*** to **Heading**
		},
		// Code block boundaries
		{
			name:     "code block then list",
			input:    "```\ncode\n```\n\n- list",
			expected: "```\ncode\n```\n\n• list",
		},
		// Multiple special chars
		{
			name:     "multiple ampersands",
			input:    "A & B & C",
			expected: "A &amp; B &amp; C",
		},
		// Empty lines
		{
			name:     "empty lines preserved",
			input:    "line1\n\nline2",
			expected: "line1\n\nline2",
		},
		// List after heading
		{
			name:     "heading then list",
			input:    "# Title\n- item",
			expected: "*Title*\n• item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.input)
			if result != tt.expected {
				t.Errorf("Format(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Integration Tests - Real-world Markdown to mrkdwn conversion
// =============================================================================

func TestMrkdwnFormatter_Integration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		desc     string
	}{
		{
			name: "AI response with code",
			input: `# File Analysis Complete

I've analyzed the codebase. Here are the findings:

## Summary
- **Total files**: 25
- **Lines of code**: ~3,500

## Key Issues
1. Missing error handling in pool.go
2. Potential race condition in session.go

> Recommendation: Add mutex protection

[View Dashboard](https://example.com/dashboard)`,
			expected: `*File Analysis Complete*

I've analyzed the codebase. Here are the findings:

*Summary*
• *Total files*: 25
• *Lines of code*: ~3,500

*Key Issues*
• Missing error handling in pool.go
• Potential race condition in session.go

> Recommendation: Add mutex protection

<https://example.com/dashboard|View Dashboard>`,
			desc: "Complex AI response with headings, lists, code, and links",
		},
		{
			name:     "Tool output with special chars",
			input:    "Command output:\n```\nError: Connection failed & retry exhausted\nDetails: a < b > c\n```\n\n> This is a critical error",
			expected: "Command output:\n```\nError: Connection failed & retry exhausted\nDetails: a < b > c\n```\n\n> This is a critical error",
			desc:     "Tool output with code block and blockquote",
		},
		{
			name:     "Mixed inline formatting",
			input:    "This is **bold**, _italic_, ~~deleted~~, and `code` text.",
			expected: "This is *bold*, _italic_, ~deleted~, and `code` text.",
			desc:     "All inline formatting types",
		},
		{
			name: "Multiple links",
			input: `Check these resources:
- [Documentation](https://docs.example.com)
- [API Reference](https://api.example.com)
- [GitHub](https://github.com/example)`,
			expected: `Check these resources:
• <https://docs.example.com|Documentation>
• <https://api.example.com|API Reference>
• <https://github.com/example|GitHub>`,
			desc: "List with links",
		},
		{
			name:     "Slack mentions preserved",
			input:    "Ping <@U123> and <!here> for review",
			expected: "Ping <@U123> and <!here> for review",
			desc:     "Slack mentions should not be escaped",
		},
		{
			name:     "Complex URL with params",
			input:    "[Search](https://example.com/search?q=hello+world&page=1)",
			expected: "<https://example.com/search?q=hello+world&page=1|Search>",
			desc:     "URL with query parameters",
		},
		{
			name:     "Nested code and quotes",
			input:    "> Here's the code:\n> ```\n> func main() {\n>     fmt.Println(\"Hello\")\n> }\n> ```",
			expected: "> Here's the code:\n> ```\n> func main() {\n> fmt.Println(\"Hello\")\n> }\n> ```",
			desc:     "Blockquote containing code block (note: leading spaces in blockquote lines are trimmed)",
		},
		{
			name: "Heading levels",
			input: `# H1 Main Title
## H2 Section
### H3 Subsection
#### H4 Minor Section

Content here.`,
			expected: `*H1 Main Title*
*H2 Section*
*H3 Subsection*
*H4 Minor Section*

Content here.`,
			desc: "All heading levels",
		},
	}

	formatter := NewMrkdwnFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.input)
			if result != tt.expected {
				t.Errorf("Format(%q) = %q, want %q\n\nDesc: %s", tt.input, result, tt.expected, tt.desc)
			}
		})
	}
}

// TestMrkdwnFormatter_InputOutputVerification provides comprehensive input/output
// verification for production scenarios
func TestMrkdwnFormatter_InputOutputVerification(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Empty and whitespace
		{"empty", "", ""},
		{"whitespace", "   ", "   "},
		{"newline only", "\n", "\n"},

		// Single element conversions
		{"h1 only", "# Hello", "*Hello*"},
		{"h2 only", "## Hello", "*Hello*"},
		{"h6 only", "###### Hello", "*Hello*"},
		{"bold only", "**bold", "**bold"},
		{"italic only", "_italic_", "_italic_"},
		{"strikethrough only", "~~gone~~", "~gone~"},
		{"code only", "`code`", "`code`"},
		{"link only", "[text](http://x.com)", "<http://x.com|text>"},
		{"list single", "- item", "• item"},
		{"ordered single", "1. first", "• first"},
		{"quote single", "> quote", "> quote"},

		// Real-world message patterns
		{
			name:     "error message",
			input:    "**Error**: Connection timeout\n\n> The server didn't respond within 30s\n\n```json\n{\"error\": \"ETIMEDOUT\"}\n```",
			expected: "*Error*: Connection timeout\n\n> The server didn't respond within 30s\n\n```json\n{\"error\": \"ETIMEDOUT\"}\n```",
		},
		{
			name:     "success message",
			input:    "✅ **Build Successful**\n\nTests: _all passing_\nDuration: 2.5s\n\n[View Details](http://ci.example.com/build/123)",
			expected: "✅ *Build Successful*\n\nTests: _all passing_\nDuration: 2.5s\n\n<http://ci.example.com/build/123|View Details>",
		},
		{
			name:     "progressive disclosure",
			input:    "### Step 1/3: Installing dependencies\n\n```bash\nnpm install\n```\n\n**Status**: In progress...",
			expected: "*Step 1/3: Installing dependencies*\n\n```bash\nnpm install\n```\n\n*Status*: In progress...",
		},
	}

	formatter := NewMrkdwnFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.Format(tt.input)
			if result != tt.expected {
				t.Errorf("\nInput:    %q\nExpected: %q\nGot:      %q", tt.input, tt.expected, result)
			}
		})
	}
}
