// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pageparser

import (
	"bytes"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gohugoio/hugo/parser/metadecoders"
)

func FuzzParse(f *testing.F) {
	samples := []string{
		`{{< foo >}}`,
		`{{% foo %}}`,
		`{{< foo >}} {{< bar >}}`,
		`---
title: "Front Matters"
---

This is some summary. This is some summary. This is some summary. This is some summary.

 <!--more-->

 Foo bars.
		 
		`,
	}
	for _, s := range samples {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		cfg := Config{}
		_, _ = parseBytes(b, cfg, lexIntroSection)
	})
}

func BenchmarkParse(b *testing.B) {
	start := `


---
title: "Front Matters"
description: "It really does"
---

This is some summary. This is some summary. This is some summary. This is some summary.

 <!--more-->


`
	input := []byte(start + strings.Repeat(strings.Repeat("this is text", 30)+"{{< myshortcode >}}This is some inner content.{{< /myshortcode >}}", 10))
	cfg := Config{}

	for b.Loop() {
		if _, err := parseBytes(input, cfg, lexIntroSection); err != nil {
			b.Fatal(err)
		}
	}
}

func TestFormatFromFrontMatterType(t *testing.T) {
	c := qt.New(t)
	for _, test := range []struct {
		typ    ItemType
		expect metadecoders.Format
	}{
		{TypeFrontMatterJSON, metadecoders.JSON},
		{TypeFrontMatterTOML, metadecoders.TOML},
		{TypeFrontMatterYAML, metadecoders.YAML},
		{TypeFrontMatterORG, metadecoders.ORG},
		{TypeIgnore, ""},
	} {
		c.Assert(FormatFromFrontMatterType(test.typ), qt.Equals, test.expect)
	}
}

func TestIsProbablyItemsSource(t *testing.T) {
	c := qt.New(t)

	input := ` {{< foo >}} `
	items, err := collectStringMain(input)
	c.Assert(err, qt.IsNil)

	c.Assert(IsProbablySourceOfItems([]byte(input), items), qt.IsTrue)
	c.Assert(IsProbablySourceOfItems(bytes.Repeat([]byte(" "), len(input)), items), qt.IsFalse)
	c.Assert(IsProbablySourceOfItems([]byte(`{{< foo >}}  `), items), qt.IsFalse)
	c.Assert(IsProbablySourceOfItems([]byte(``), items), qt.IsFalse)
}

func TestHasShortcode(t *testing.T) {
	c := qt.New(t)

	c.Assert(HasShortcode("{{< foo >}}"), qt.IsTrue)
	c.Assert(HasShortcode("aSDasd  SDasd aSD\n\nasdfadf{{% foo %}}\nasdf"), qt.IsTrue)
	c.Assert(HasShortcode("{{</* foo */>}}"), qt.IsFalse)
	c.Assert(HasShortcode("{{%/* foo */%}}"), qt.IsFalse)
}

func BenchmarkHasShortcode(b *testing.B) {
	withShortcode := strings.Repeat("this is text", 30) + "{{< myshortcode >}}This is some inner content.{{< /myshortcode >}}" + strings.Repeat("this is text", 30)
	withoutShortcode := strings.Repeat("this is text", 30) + "This is some inner content." + strings.Repeat("this is text", 30)
	b.Run("Match", func(b *testing.B) {
		for b.Loop() {
			HasShortcode(withShortcode)
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		for b.Loop() {
			HasShortcode(withoutShortcode)
		}
	})
}

func TestSummaryDividerStartingFromMain(t *testing.T) {
	c := qt.New(t)

	input := `aaa <!--more--> bbb`
	items, err := collectStringMain(input)
	c.Assert(err, qt.IsNil)

	c.Assert(items, qt.HasLen, 4)
	c.Assert(items[1].Type, qt.Equals, TypeLeadSummaryDivider)
}

func TestMultipleSummaryDividers(t *testing.T) {
	c := qt.New(t)

	t.Run("HTML comment multiple dividers", func(t *testing.T) {
		input := `---
title: "Test"
---
First section.
<!--more-->
Second section.
<!--more-->
Third section.`
		items, err := collectStringIntro(input)
		c.Assert(err, qt.IsNil)

		var dividerCount int
		var leadDividerCount int
		for _, item := range items {
			if item.Type == TypeLeadSummaryDivider {
				leadDividerCount++
			} else if item.Type == TypeSummaryDivider {
				dividerCount++
			}
		}
		c.Assert(leadDividerCount, qt.Equals, 1)
		c.Assert(dividerCount, qt.Equals, 1)
	})

	t.Run("Org mode multiple dividers", func(t *testing.T) {
		input := `#+title: Test
First section.
# more
Second section.
# more
Third section.`
		items, err := collectStringIntro(input)
		c.Assert(err, qt.IsNil)

		var dividerCount int
		var leadDividerCount int
		for _, item := range items {
			if item.Type == TypeLeadSummaryDivider {
				leadDividerCount++
			} else if item.Type == TypeSummaryDivider {
				dividerCount++
			}
		}
		c.Assert(leadDividerCount, qt.Equals, 1)
		c.Assert(dividerCount, qt.Equals, 1)
	})

	t.Run("Single divider backward compatible", func(t *testing.T) {
		input := `---
title: "Test"
---
First section.
<!--more-->
Second section.`
		items, err := collectStringIntro(input)
		c.Assert(err, qt.IsNil)

		var leadDividerCount int
		var otherDividerCount int
		for _, item := range items {
			if item.Type == TypeLeadSummaryDivider {
				leadDividerCount++
			} else if item.Type == TypeSummaryDivider {
				otherDividerCount++
			}
		}
		c.Assert(leadDividerCount, qt.Equals, 1)
		c.Assert(otherDividerCount, qt.Equals, 0)
	})

	t.Run("Mixed dividers with YAML front matter", func(t *testing.T) {
		input := `---
title: "Test"
---
First section.
<!--more-->
Second section.
# more
Third section.`
		items, err := collectStringIntro(input)
		c.Assert(err, qt.IsNil)

		var htmlDividers int
		var orgDividers int
		for _, item := range items {
			if item.Type == TypeLeadSummaryDivider || item.Type == TypeSummaryDivider {
				val := string(item.Val([]byte(input)))
				if strings.Contains(val, "<!--more-->") {
					htmlDividers++
				} else if strings.Contains(val, "# more") {
					orgDividers++
				}
			}
		}
		c.Assert(htmlDividers, qt.Equals, 1)
		c.Assert(orgDividers, qt.Equals, 0)
	})
}

func TestNoFrontMatterStability(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name  string
		input string
	}{
		{"Empty file", ""},
		{"Pure whitespace", "   \n\n  \t  "},
		{"BOM only", "\xef\xbb\xbf"},
		{"BOM with whitespace", "\xef\xbb\xbf   \n  "},
		{"Hash start no front matter", "# This is a heading"},
		{"Plus start no front matter", "+ just some text"},
		{"Brace start unclosed JSON", "{ not closed JSON"},
		{"No front matter just content", "Just some markdown content\n\nWith multiple lines.\n\n* List item"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cf, err := ParseFrontMatterAndContent(strings.NewReader(tc.input))
			c.Assert(err, qt.IsNil)
			c.Assert(string(cf.Content), qt.Equals, tc.input)
		})
	}
}

func TestOrgFrontMatterDetection(t *testing.T) {
	c := qt.New(t)

	t.Run("Org format with colon in value", func(t *testing.T) {
		input := `#+title: Test: With Colons
#+custom: key=value:another=thing
#+date: 2024-01-01

Content here.`
		items, err := collectStringIntro(input)
		c.Assert(err, qt.IsNil)

		var foundOrgFM bool
		for _, item := range items {
			if item.Type == TypeFrontMatterORG {
				foundOrgFM = true
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, "#+title:"), qt.IsTrue)
				c.Assert(strings.Contains(val, "key=value:another=thing"), qt.IsTrue)
			}
		}
		c.Assert(foundOrgFM, qt.IsTrue)
	})

	t.Run("Org format with equals in value", func(t *testing.T) {
		var d metadecoders.Decoder
		format := d.FormatFromContentString(`#+title: Test
#+options: toc:t author:t num:nil
#+custom: a=1 b=2 c=3`)
		c.Assert(format, qt.Equals, metadecoders.ORG)
	})

	t.Run("YAML not confused with Org", func(t *testing.T) {
		var d metadecoders.Decoder
		format := d.FormatFromContentString(`---
title: "Test"
---`)
		c.Assert(format, qt.Not(qt.Equals), metadecoders.ORG)
	})

	t.Run("JSON not confused with Org", func(t *testing.T) {
		var d metadecoders.Decoder
		format := d.FormatFromContentString(`{
  "title": "Test"
}`)
		c.Assert(format, qt.Not(qt.Equals), metadecoders.ORG)
	})
}

func TestShortcodeEscapePreservation(t *testing.T) {
	c := qt.New(t)

	t.Run("Backslash before asterisk preserved", func(t *testing.T) {
		input := `{{< sc param="Hello \* World" >}}`
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, `\*`), qt.IsTrue)
			}
		}
	})

	t.Run("Backslash before underscore preserved", func(t *testing.T) {
		input := `{{< sc param="Hello \_ World" >}}`
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, `\_`), qt.IsTrue)
			}
		}
	})

	t.Run("Backslash before paren preserved", func(t *testing.T) {
		input := `{{< sc param="Hello \( World" >}}`
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, `\(`), qt.IsTrue)
			}
		}
	})

	t.Run("Backslash before bracket preserved", func(t *testing.T) {
		input := `{{< sc param="Hello \[ World" >}}`
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, `\[`), qt.IsTrue)
			}
		}
	})

	t.Run("Backslash before multi-byte character", func(t *testing.T) {
		input := `{{< sc param="Hello \中文 World" >}}`
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, `\中文`), qt.IsTrue)
			}
		}
	})

	t.Run("Backslash at end of line", func(t *testing.T) {
		input := "{{< sc param=\"Hello \\\nWorld\" >}}"
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(strings.Contains(val, "\\\n"), qt.IsTrue)
			}
		}
	})

	t.Run("Backslash before backtick should error", func(t *testing.T) {
		input := "{{< sc param=\"Hello \\` World\" >}}"
		items, err := collectStringMain(input)
		// The error may be returned directly or via a tError item
		var foundError bool
		var errorMsg string
		if err != nil {
			foundError = true
			errorMsg = err.Error()
		} else {
			for _, item := range items {
				if item.Type == tError && item.Err != nil {
					foundError = true
					errorMsg = item.Err.Error()
					break
				}
			}
		}
		c.Assert(foundError, qt.IsTrue)
		c.Assert(strings.Contains(errorMsg, "unrecognized escape character"), qt.IsTrue)
	})

	t.Run("Escaped quote becomes quote", func(t *testing.T) {
		input := `{{< sc param="Hello \"World\"" >}}`
		items, err := collectStringMain(input)
		c.Assert(err, qt.IsNil)

		for _, item := range items {
			if item.Type == tScParamVal {
				val := string(item.Val([]byte(input)))
				c.Assert(val, qt.Equals, `Hello "World"`)
			}
		}
	})
}
