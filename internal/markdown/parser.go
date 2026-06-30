package markdown

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/geekjourneyx/md2x/internal/article"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

type frontMatter struct {
	Title string `yaml:"title"`
	Cover string `yaml:"cover"`
}

func Parse(sourcePath string, content []byte) (*article.Document, error) {
	meta, body, err := splitFrontMatter(content)
	if err != nil {
		return nil, err
	}

	doc := &article.Document{
		SourcePath: sourcePath,
		Title:      meta.Title,
		Cover:      meta.Cover,
	}

	md := goldmark.New(goldmark.WithExtensions(extension.Strikethrough))
	root := md.Parser().Parse(text.NewReader(body))
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		appendBlock(doc, child, body)
	}

	if doc.Title == "" {
		doc.Title = firstHeadingText(doc.Blocks)
	}
	if doc.Title == "" {
		doc.Title = "Untitled Article"
	}

	return doc, nil
}

func splitFrontMatter(content []byte) (frontMatter, []byte, error) {
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		return frontMatter{}, content, nil
	}

	normalized := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	end := bytes.Index(normalized[4:], []byte("\n---\n"))
	if end < 0 {
		return frontMatter{}, nil, fmt.Errorf("unterminated frontmatter")
	}

	var meta frontMatter
	yamlBody := normalized[4 : 4+end]
	if err := yaml.Unmarshal(yamlBody, &meta); err != nil {
		return frontMatter{}, nil, err
	}
	return meta, normalized[4+end+5:], nil
}

func appendBlock(doc *article.Document, node ast.Node, source []byte) {
	switch n := node.(type) {
	case *ast.Heading:
		text, spans := flattenInline(n, source)
		doc.Blocks = append(doc.Blocks, article.Block{
			Type:  "heading",
			Level: n.Level,
			Text:  text,
			Spans: spans,
		})
	case *ast.Paragraph:
		if image, ok := paragraphImage(n, source); ok {
			appendImageBlock(doc, image, source)
			return
		}
		if containsImage(n) {
			doc.Warnings = append(doc.Warnings, article.Diagnostic{
				Severity: "warning",
				Code:     "UNSUPPORTED_MIXED_IMAGE",
				Message:  "images must be the only content in a paragraph to upload as X Article media",
				Block:    len(doc.Blocks),
			})
		}
		text, spans := flattenInline(n, source)
		if text != "" {
			doc.Blocks = append(doc.Blocks, article.Block{
				Type:  "paragraph",
				Text:  text,
				Spans: spans,
			})
		}
	case *ast.Blockquote:
		text, spans := flattenInline(n, source)
		if text != "" {
			doc.Blocks = append(doc.Blocks, article.Block{
				Type:  "blockquote",
				Text:  text,
				Spans: spans,
			})
		}
	case *ast.List:
		itemIndex := 0
		for item := n.FirstChild(); item != nil; item = item.NextSibling() {
			listItem, ok := item.(*ast.ListItem)
			if !ok {
				continue
			}
			text, spans := flattenInline(listItem, source)
			blockType := "unordered-list-item"
			data := map[string]string(nil)
			if n.IsOrdered() {
				blockType = "ordered-list-item"
				data = map[string]string{"number": strconv.Itoa(n.Start + itemIndex)}
			}
			doc.Blocks = append(doc.Blocks, article.Block{
				Type:  blockType,
				Text:  text,
				Spans: spans,
				Data:  data,
			})
			itemIndex++
		}
	case *ast.FencedCodeBlock:
		text := fencedCodeText(n, source)
		blockIndex := len(doc.Blocks)
		doc.Blocks = append(doc.Blocks, article.Block{
			Type: "code",
			Text: text,
		})
		doc.Warnings = append(doc.Warnings, article.Diagnostic{
			Severity: "warning",
			Code:     "UNSUPPORTED_CODE_BLOCK",
			Message:  "fenced code block is preserved as plain text",
			Block:    blockIndex,
		})
	case *ast.CodeBlock:
		text := codeBlockText(n, source)
		blockIndex := len(doc.Blocks)
		doc.Blocks = append(doc.Blocks, article.Block{
			Type: "code",
			Text: text,
		})
		doc.Warnings = append(doc.Warnings, article.Diagnostic{
			Severity: "warning",
			Code:     "UNSUPPORTED_CODE_BLOCK",
			Message:  "indented code block is preserved as plain text",
			Block:    blockIndex,
		})
	default:
		doc.Warnings = append(doc.Warnings, article.Diagnostic{
			Severity: "warning",
			Code:     "UNSUPPORTED_MARKDOWN_BLOCK",
			Message:  fmt.Sprintf("unsupported Markdown block %q was skipped", node.Kind().String()),
			Block:    len(doc.Blocks),
		})
	}
}

func paragraphImage(paragraph *ast.Paragraph, source []byte) (*ast.Image, bool) {
	var image *ast.Image
	for child := paragraph.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok && len(bytes.TrimSpace(textNode.Segment.Value(source))) == 0 {
			continue
		}
		nextImage, ok := child.(*ast.Image)
		if !ok || image != nil {
			return nil, false
		}
		image = nextImage
	}
	return image, image != nil
}

func containsImage(node ast.Node) bool {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if _, ok := child.(*ast.Image); ok {
			return true
		}
		if containsImage(child) {
			return true
		}
	}
	return false
}

func appendImageBlock(doc *article.Document, image *ast.Image, source []byte) {
	alt, _ := flattenInline(image, source)
	asset := article.Asset{
		Index:  len(doc.Assets),
		Source: string(image.Destination),
		Alt:    alt,
		Role:   "body",
	}
	doc.Assets = append(doc.Assets, asset)
	doc.Blocks = append(doc.Blocks, article.Block{
		Type: "image",
		Text: alt,
		Data: map[string]string{
			"asset_index": strconv.Itoa(asset.Index),
			"source":      asset.Source,
			"alt":         asset.Alt,
		},
	})
}

func flattenInline(node ast.Node, source []byte) (string, []article.Span) {
	var b strings.Builder
	var spans []article.Span
	writeInline(node, source, &b, &spans, nil)
	text := b.String()
	return strings.TrimSpace(text), adjustSpansForTrim(text, spans)
}

func writeInline(node ast.Node, source []byte, b *strings.Builder, spans *[]article.Span, inherited []spanStyle) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			writeText(n, source, b, spans, inherited)
		case *ast.String:
			writeStyledString(string(n.Value), b, spans, inherited)
		case *ast.Emphasis:
			style := "italic"
			if n.Level >= 2 {
				style = "bold"
			}
			writeInline(n, source, b, spans, append(inherited, spanStyle{Style: style}))
		case *ast.Link:
			offset := b.Len()
			writeInline(n, source, b, spans, inherited)
			if length := b.Len() - offset; length > 0 {
				*spans = append(*spans, article.Span{
					Offset: offset,
					Length: length,
					Style:  "link",
					URL:    string(n.Destination),
				})
			}
		case *ast.AutoLink:
			writeStyledString(string(n.URL(source)), b, spans, inherited)
		case *ast.Image:
			writeInline(n, source, b, spans, inherited)
		default:
			writeSeparatorBeforeContainer(child, b)
			kind := child.Kind().String()
			if kind == "Strikethrough" {
				writeInline(child, source, b, spans, append(inherited, spanStyle{Style: "strikethrough"}))
				continue
			}
			writeInline(child, source, b, spans, inherited)
		}
	}
}

func writeSeparatorBeforeContainer(node ast.Node, b *strings.Builder) {
	switch node.(type) {
	case *ast.Blockquote, *ast.List, *ast.ListItem, *ast.Paragraph:
	default:
		return
	}
	if b.Len() == 0 {
		return
	}
	text := b.String()
	if len(strings.TrimRightFunc(text, unicode.IsSpace)) < len(text) {
		return
	}
	b.WriteByte(' ')
}

type spanStyle struct {
	Style string
	URL   string
}

func writeText(n *ast.Text, source []byte, b *strings.Builder, spans *[]article.Span, styles []spanStyle) {
	value := string(n.Segment.Value(source))
	if n.SoftLineBreak() || n.HardLineBreak() {
		value += " "
	}
	writeStyledString(value, b, spans, styles)
}

func writeStyledString(value string, b *strings.Builder, spans *[]article.Span, styles []spanStyle) {
	if value == "" {
		return
	}
	offset := b.Len()
	b.WriteString(value)
	for _, style := range styles {
		*spans = append(*spans, article.Span{
			Offset: offset,
			Length: len(value),
			Style:  style.Style,
			URL:    style.URL,
		})
	}
}

func adjustSpansForTrim(text string, spans []article.Span) []article.Span {
	leftTrim := len(text) - len(strings.TrimLeftFunc(text, unicode.IsSpace))
	right := len(strings.TrimRightFunc(text, unicode.IsSpace))
	var adjusted []article.Span
	for _, span := range spans {
		start := max(span.Offset, leftTrim)
		end := min(span.Offset+span.Length, right)
		if end <= start {
			continue
		}
		span.Offset = start - leftTrim
		span.Length = end - start
		adjusted = append(adjusted, span)
	}
	return adjusted
}

func fencedCodeText(n *ast.FencedCodeBlock, source []byte) string {
	return blockLinesText(n.Lines(), source)
}

func codeBlockText(n *ast.CodeBlock, source []byte) string {
	return blockLinesText(n.Lines(), source)
}

func blockLinesText(lines *text.Segments, source []byte) string {
	var b strings.Builder
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		b.Write(line.Value(source))
	}
	return b.String()
}

func firstHeadingText(blocks []article.Block) string {
	for _, block := range blocks {
		if block.Type == "heading" && block.Text != "" {
			return block.Text
		}
	}
	return ""
}
