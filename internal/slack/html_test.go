package slack

import "testing"

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "empty input",
			html: "",
			want: "",
		},
		{
			name: "plain text",
			html: "Hello world",
			want: "Hello world",
		},
		{
			name: "paragraph tags",
			html: "<p>First paragraph</p><p>Second paragraph</p>",
			want: "First paragraph\n\nSecond paragraph",
		},
		{
			name: "line breaks",
			html: "Line one<br>Line two<br/>Line three",
			want: "Line one\nLine two\nLine three",
		},
		{
			name: "div tags",
			html: "<div>Block one</div><div>Block two</div>",
			want: "Block one\n\nBlock two",
		},
		{
			name: "heading h1",
			html: "<h1>Title</h1><p>Content</p>",
			want: "# Title\n\nContent",
		},
		{
			name: "heading h2",
			html: "<h2>Subtitle</h2><p>Content</p>",
			want: "## Subtitle\n\nContent",
		},
		{
			name: "heading h3",
			html: "<h3>Section</h3><p>Content</p>",
			want: "### Section\n\nContent",
		},
		{
			name: "unordered list",
			html: "<ul><li>Item one</li><li>Item two</li><li>Item three</li></ul>",
			want: "- Item one\n- Item two\n- Item three",
		},
		{
			name: "ordered list",
			html: "<ol><li>First</li><li>Second</li></ol>",
			want: "- First\n- Second",
		},
		{
			name: "bold and italic stripped",
			html: "<p>This is <b>bold</b> and <i>italic</i> text</p>",
			want: "This is bold and italic text",
		},
		{
			name: "link preserves text",
			html: `<a href="https://example.com">Click here</a>`,
			want: "Click here",
		},
		{
			name: "HTML entities",
			html: "Tom &amp; Jerry &lt;3 &gt; &quot;hello&quot;",
			want: `Tom & Jerry <3 > "hello"`,
		},
		{
			name: "nbsp entity",
			html: "Hello&nbsp;world",
			want: "Hello world",
		},
		{
			name: "nested tags",
			html: "<div><p>Inside <b>nested</b> tags</p></div>",
			want: "Inside nested tags",
		},
		{
			name: "excessive whitespace collapsed",
			html: "<p>  Lots   of   spaces  </p>",
			want: "Lots of spaces",
		},
		{
			name: "complex canvas-like content",
			html: `<h1>Project Plan</h1><p>This is the <b>main</b> plan.</p><h2>Goals</h2><ul><li>Ship feature</li><li>Write tests</li></ul><p>Done!</p>`,
			want: "# Project Plan\n\nThis is the main plan.\n\n## Goals\n\n- Ship feature\n- Write tests\n\nDone!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.html)
			if got != tt.want {
				t.Errorf("stripHTML():\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
