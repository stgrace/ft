package tool

import "fmt"

type Markdown struct {
	MarkdownDoc string
}

func (m *Markdown) AddH1(header string) *Markdown {
	m.MarkdownDoc += fmt.Sprintf("# %s\n\n", header)
	return m
}

func (m *Markdown) AddH2(header string) *Markdown {
	m.MarkdownDoc += fmt.Sprintf("## %s\n\n", header)
	return m
}

func (m *Markdown) AddDiffBlock(diff string) *Markdown {
	m.MarkdownDoc += fmt.Sprintf("```diff\n%s\n```\n\n", diff)
	return m
}
