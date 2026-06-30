package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2x/internal/article"
)

func readFixture(t *testing.T, name string) (string, []byte) {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "articles", name)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return path, content
}

func TestParseMetadataAndBlocks(t *testing.T) {
	sourcePath, content := readFixture(t, "simple.md")

	doc, err := Parse(sourcePath, content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	var _ = (*article.Document)(doc)

	if doc.Title != "My first Article" {
		t.Fatalf("Title = %q, want %q", doc.Title, "My first Article")
	}
	if doc.Cover != "./cover.png" {
		t.Fatalf("Cover = %q, want %q", doc.Cover, "./cover.png")
	}
	if got := len(doc.Blocks); got != 2 {
		t.Fatalf("len(Blocks) = %d, want 2", got)
	}
	if doc.Blocks[0].Type != "heading" {
		t.Fatalf("first block Type = %q, want heading", doc.Blocks[0].Type)
	}
	if doc.Blocks[0].Level != 1 {
		t.Fatalf("first block Level = %d, want 1", doc.Blocks[0].Level)
	}
	if doc.Blocks[1].Text != "Hello from md2x." {
		t.Fatalf("second block Text = %q, want %q", doc.Blocks[1].Text, "Hello from md2x.")
	}
	if got := countSpans(doc.Blocks[1].Spans, "bold"); got != 1 {
		t.Fatalf("bold spans = %d, want 1", got)
	}
}

func TestParseFormattingBlocks(t *testing.T) {
	sourcePath, content := readFixture(t, "formatting.md")

	doc, err := Parse(sourcePath, content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	for _, blockType := range []string{"blockquote", "unordered-list-item", "ordered-list-item"} {
		if !hasBlockType(doc.Blocks, blockType) {
			t.Fatalf("expected block type %q in %#v", blockType, doc.Blocks)
		}
	}

	paragraph, ok := findParagraphWithPrefix(doc.Blocks, "Paragraph with")
	if !ok {
		t.Fatalf("paragraph beginning %q not found in %#v", "Paragraph with", doc.Blocks)
	}

	wantText := "Paragraph with bold, italic, strike, and link."
	if paragraph.Text != wantText {
		t.Fatalf("paragraph Text = %q, want %q", paragraph.Text, wantText)
	}

	if !hasSpanText(paragraph, "bold", "bold", "") {
		t.Fatalf("missing bold span for %q in %#v", "bold", paragraph.Spans)
	}
	if !hasSpanText(paragraph, "italic", "italic", "") {
		t.Fatalf("missing italic span for %q in %#v", "italic", paragraph.Spans)
	}
	if !hasSpanText(paragraph, "strikethrough", "strike", "") {
		t.Fatalf("missing strikethrough span for %q in %#v", "strike", paragraph.Spans)
	}
	if !hasSpanText(paragraph, "link", "link", "https://example.com") {
		t.Fatalf("missing link span for %q with URL %q in %#v", "link", "https://example.com", paragraph.Spans)
	}

	var orderedNumbers []string
	for _, block := range doc.Blocks {
		if block.Type == "ordered-list-item" {
			orderedNumbers = append(orderedNumbers, block.Data["number"])
		}
	}
	wantNumbers := []string{"1", "2"}
	if strings.Join(orderedNumbers, ",") != strings.Join(wantNumbers, ",") {
		t.Fatalf("ordered list item numbers = %#v, want %#v", orderedNumbers, wantNumbers)
	}
}

func TestParseUnicodeTrimKeepsSpanRangesValid(t *testing.T) {
	doc, err := Parse("unicode.md", []byte("\u00a0　**styled**　\u00a0\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := len(doc.Blocks); got != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", got)
	}
	block := doc.Blocks[0]
	if block.Text != "styled" {
		t.Fatalf("block Text = %q, want %q", block.Text, "styled")
	}
	for _, span := range block.Spans {
		if span.Offset < 0 || span.Length < 0 || span.Offset+span.Length > len(block.Text) {
			t.Fatalf("span out of range for text length %d: %#v", len(block.Text), span)
		}
	}
	if !hasSpanText(block, "bold", "styled", "") {
		t.Fatalf("missing valid bold span for styled text in %#v", block.Spans)
	}
}

func TestParseBlockquoteSeparatesParagraphs(t *testing.T) {
	doc, err := Parse("quote.md", []byte("> first\n>\n> second\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := len(doc.Blocks); got != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", got)
	}
	block := doc.Blocks[0]
	if block.Type != "blockquote" {
		t.Fatalf("block Type = %q, want blockquote", block.Type)
	}
	if strings.Contains(block.Text, "firstsecond") {
		t.Fatalf("blockquote text concatenated without separator: %q", block.Text)
	}
	if block.Text != "first second" {
		t.Fatalf("blockquote Text = %q, want %q", block.Text, "first second")
	}
}

func TestParseMixedFormattingLinkUsesOneLinkSpan(t *testing.T) {
	doc, err := Parse("link.md", []byte("[a **b** c](https://example.com)\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := len(doc.Blocks); got != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", got)
	}
	block := doc.Blocks[0]
	if block.Text != "a b c" {
		t.Fatalf("block Text = %q, want %q", block.Text, "a b c")
	}
	if got := countSpans(block.Spans, "link"); got != 1 {
		t.Fatalf("link spans = %d, want 1: %#v", got, block.Spans)
	}
	if !hasSpanText(block, "link", "a b c", "https://example.com") {
		t.Fatalf("missing single link span over full text in %#v", block.Spans)
	}
	if !hasSpanText(block, "bold", "b", "") {
		t.Fatalf("missing bold span for %q in %#v", "b", block.Spans)
	}
}

func TestParseImageAssets(t *testing.T) {
	sourcePath, content := readFixture(t, "images.md")

	doc, err := Parse(sourcePath, content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if doc.Cover != "./cover.png" {
		t.Fatalf("Cover = %q, want %q", doc.Cover, "./cover.png")
	}
	if got := len(doc.Assets); got != 1 {
		t.Fatalf("len(Assets) = %d, want 1", got)
	}
	if doc.Assets[0].Source != "./diagram.png" {
		t.Fatalf("asset Source = %q, want %q", doc.Assets[0].Source, "./diagram.png")
	}
	if doc.Assets[0].Alt != "Diagram" {
		t.Fatalf("asset Alt = %q, want %q", doc.Assets[0].Alt, "Diagram")
	}
	if doc.Assets[0].Index != 0 {
		t.Fatalf("asset Index = %d, want 0", doc.Assets[0].Index)
	}
	if doc.Assets[0].Role != "body" {
		t.Fatalf("asset Role = %q, want %q", doc.Assets[0].Role, "body")
	}

	var imageBlock article.Block
	var foundImageBlock bool
	for _, block := range doc.Blocks {
		if block.Type == "image" {
			imageBlock = block
			foundImageBlock = true
			break
		}
	}
	if !foundImageBlock {
		t.Fatalf("image block not found in %#v", doc.Blocks)
	}
	if imageBlock.Type != "image" {
		t.Fatalf("image block Type = %q, want %q", imageBlock.Type, "image")
	}
	if imageBlock.Text != "Diagram" {
		t.Fatalf("image block Text = %q, want %q", imageBlock.Text, "Diagram")
	}
	if imageBlock.Data["asset_index"] != "0" {
		t.Fatalf("image block asset_index = %q, want %q", imageBlock.Data["asset_index"], "0")
	}
	if imageBlock.Data["source"] != "./diagram.png" {
		t.Fatalf("image block source = %q, want %q", imageBlock.Data["source"], "./diagram.png")
	}
	if imageBlock.Data["alt"] != "Diagram" {
		t.Fatalf("image block alt = %q, want %q", imageBlock.Data["alt"], "Diagram")
	}
}

func TestParseMixedImageParagraphWarns(t *testing.T) {
	doc, err := Parse("mixed-image.md", []byte("Before ![Diagram](./diagram.png) after.\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := len(doc.Assets); got != 0 {
		t.Fatalf("len(Assets) = %d, want 0 for mixed image paragraph", got)
	}
	if !hasDiagnostic(doc.Warnings, "UNSUPPORTED_MIXED_IMAGE") {
		t.Fatalf("missing mixed image warning in %#v", doc.Warnings)
	}
}

func TestParseUnsupportedBlockWarns(t *testing.T) {
	doc, err := Parse("html.md", []byte("<div>raw html</div>\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if !hasDiagnostic(doc.Warnings, "UNSUPPORTED_MARKDOWN_BLOCK") {
		t.Fatalf("missing unsupported block warning in %#v", doc.Warnings)
	}
}

func countSpans(spans []article.Span, style string) int {
	var count int
	for _, span := range spans {
		if span.Style == style {
			count++
		}
	}
	return count
}

func hasBlockType(blocks []article.Block, blockType string) bool {
	for _, block := range blocks {
		if block.Type == blockType {
			return true
		}
	}
	return false
}

func hasDiagnostic(diagnostics []article.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func findParagraphWithPrefix(blocks []article.Block, prefix string) (article.Block, bool) {
	for _, block := range blocks {
		if block.Type == "paragraph" && strings.HasPrefix(block.Text, prefix) {
			return block, true
		}
	}
	return article.Block{}, false
}

func hasSpanText(block article.Block, style, text, url string) bool {
	for _, span := range block.Spans {
		if span.Style != style || spanText(block, span) != text {
			continue
		}
		if url == "" || span.URL == url {
			return true
		}
	}
	return false
}

func spanText(block article.Block, span article.Span) string {
	if span.Offset < 0 || span.Length < 0 || span.Offset+span.Length > len(block.Text) {
		return ""
	}
	return block.Text[span.Offset : span.Offset+span.Length]
}
