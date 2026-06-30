package draftjs

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/geekjourneyx/md2x/internal/article"
)

func TestRenderNilDocumentReturnsError(t *testing.T) {
	state, err := Render(nil)
	if err == nil {
		t.Fatal("Render(nil) returned nil error, want error")
	}
	if state != nil {
		t.Fatalf("Render(nil) state = %#v, want nil", state)
	}
}

func TestRenderUsesDeterministicBase36BlockKeys(t *testing.T) {
	doc := &article.Document{
		Blocks: make([]article.Block, 10),
	}
	for i := range doc.Blocks {
		doc.Blocks[i] = article.Block{Type: "paragraph", Text: strconv.Itoa(i + 1)}
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	for i, want := range []string{"1", "2", "3", "a"} {
		blockIndex := i
		if want == "a" {
			blockIndex = 9
		}
		if state.Blocks[blockIndex].Key != want {
			t.Fatalf("block %d Key = %q, want %q", blockIndex, state.Blocks[blockIndex].Key, want)
		}
	}
}

func TestRenderMapsBlockTypes(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{Type: "heading", Level: 2, Text: "H2"},
			{Type: "heading", Level: 3, Text: "H3"},
			{Type: "unordered-list-item", Text: "Bullet"},
			{Type: "ordered-list-item", Text: "Numbered"},
			{Type: "blockquote", Text: "Quote"},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	wantTypes := []string{
		"header-two",
		"header-three",
		"unordered-list-item",
		"ordered-list-item",
		"blockquote",
	}
	for i, want := range wantTypes {
		if state.Blocks[i].Type != want {
			t.Fatalf("block %d Type = %q, want %q", i, state.Blocks[i].Type, want)
		}
	}
}

func TestRenderItalicStyleRange(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "paragraph",
				Text: "Plain italic text",
				Spans: []article.Span{
					{Offset: len("Plain "), Length: len("italic"), Style: "italic"},
				},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	ranges := state.Blocks[0].InlineStyleRanges
	if got := len(ranges); got != 1 {
		t.Fatalf("len(InlineStyleRanges) = %d, want 1: %#v", got, ranges)
	}
	want := StyleRange{Offset: 6, Length: 6, Style: "italic"}
	if ranges[0] != want {
		t.Fatalf("italic range = %#v, want %#v", ranges[0], want)
	}
}

func TestRenderHeadingParagraphAndLink(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{Type: "heading", Level: 1, Text: "Title"},
			{
				Type: "paragraph",
				Text: "Visit X",
				Spans: []article.Span{
					{Offset: 0, Length: len("Visit"), Style: "bold"},
					{Offset: len("Visit "), Length: len("X"), Style: "link", URL: "https://x.com"},
				},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if got := len(state.Blocks); got != 2 {
		t.Fatalf("len(Blocks) = %d, want 2", got)
	}
	if state.Blocks[0].Type != "header-one" {
		t.Fatalf("first block Type = %q, want header-one", state.Blocks[0].Type)
	}
	paragraph := state.Blocks[1]
	if paragraph.Type != "unstyled" {
		t.Fatalf("paragraph Type = %q, want unstyled", paragraph.Type)
	}
	if got := len(paragraph.InlineStyleRanges); got != 1 {
		t.Fatalf("len(InlineStyleRanges) = %d, want 1: %#v", got, paragraph.InlineStyleRanges)
	}
	if paragraph.InlineStyleRanges[0].Style != "bold" {
		t.Fatalf("style = %q, want bold", paragraph.InlineStyleRanges[0].Style)
	}
	if got := len(state.Entities); got != 1 {
		t.Fatalf("len(Entities) = %d, want 1: %#v", got, state.Entities)
	}
	if state.Entities[0].Value.Type != "link" {
		t.Fatalf("entity Type = %q, want link", state.Entities[0].Value.Type)
	}
	if state.Entities[0].Value.Data.URL != "https://x.com" {
		t.Fatalf("entity URL = %q, want https://x.com", state.Entities[0].Value.Data.URL)
	}
	if got := len(paragraph.EntityRanges); got != 1 {
		t.Fatalf("len(EntityRanges) = %d, want 1: %#v", got, paragraph.EntityRanges)
	}
	if paragraph.EntityRanges[0].Key != 0 {
		t.Fatalf("entity range Key = %d, want 0", paragraph.EntityRanges[0].Key)
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal content state: %v", err)
	}
	if strings.Contains(string(data), `"data":null`) {
		t.Fatalf("content state JSON contains data:null: %s", data)
	}
}

func TestRenderUsesUTF16Offsets(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "paragraph",
				Text: "A 😀 bold",
				Spans: []article.Span{
					{Offset: 7, Length: 4, Style: "bold"},
				},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	ranges := state.Blocks[0].InlineStyleRanges
	if got := len(ranges); got != 1 {
		t.Fatalf("len(InlineStyleRanges) = %d, want 1: %#v", got, ranges)
	}
	if ranges[0].Offset != 5 {
		t.Fatalf("UTF-16 offset = %d, want 5", ranges[0].Offset)
	}
	if ranges[0].Length != 4 {
		t.Fatalf("UTF-16 length = %d, want 4", ranges[0].Length)
	}
}

func TestRenderImageBlockWithMediaID(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "image",
				Text: "Alt text",
				Data: map[string]string{
					"media_id":       "media-123",
					"media_category": "tweet_image",
				},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if got := len(state.Blocks); got != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", got)
	}
	block := state.Blocks[0]
	if block.Type != "atomic" {
		t.Fatalf("block Type = %q, want atomic", block.Type)
	}
	if block.Text != " " {
		t.Fatalf("block Text = %q, want single space placeholder", block.Text)
	}
	if got := len(state.Entities); got != 1 {
		t.Fatalf("len(Entities) = %d, want 1: %#v", got, state.Entities)
	}
	entity := state.Entities[0]
	if entity.Value.Type != "image" {
		t.Fatalf("entity Type = %q, want image", entity.Value.Type)
	}
	if entity.Value.Data.Caption != "Alt text" {
		t.Fatalf("entity caption = %q, want original image text", entity.Value.Data.Caption)
	}
	if got := len(entity.Value.Data.MediaItems); got != 1 {
		t.Fatalf("len(MediaItems) = %d, want 1: %#v", got, entity.Value.Data.MediaItems)
	}
	if entity.Value.Data.MediaItems[0].MediaID != "media-123" {
		t.Fatalf("MediaID = %q, want media-123", entity.Value.Data.MediaItems[0].MediaID)
	}
	if got := len(block.EntityRanges); got != 1 {
		t.Fatalf("len(EntityRanges) = %d, want 1: %#v", got, block.EntityRanges)
	}
	if block.EntityRanges[0] != (EntityRange{Offset: 0, Length: 1, Key: 0}) {
		t.Fatalf("EntityRange = %#v, want offset 0 length 1 key 0", block.EntityRanges[0])
	}
	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal image block: %v", err)
	}
	if strings.Contains(string(data), `"data"`) {
		t.Fatalf("image block JSON contains internal data: %s", data)
	}
}

func TestRenderRejectsSpanOffsetInsideMultibyteRune(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "paragraph",
				Text: "A 😀 bold",
				Spans: []article.Span{
					{Offset: len("A ") + 1, Length: len("😀"), Style: "bold"},
				},
			},
		},
	}

	if _, err := Render(doc); err == nil {
		t.Fatal("Render returned nil error, want invalid span boundary error")
	}
}

func TestRenderClampsNegativeSpanOffsetToStart(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "paragraph",
				Text: "Hello",
				Spans: []article.Span{
					{Offset: -2, Length: 4, Style: "bold"},
				},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	ranges := state.Blocks[0].InlineStyleRanges
	if got := len(ranges); got != 1 {
		t.Fatalf("len(InlineStyleRanges) = %d, want 1: %#v", got, ranges)
	}
	if ranges[0] != (StyleRange{Offset: 0, Length: 2, Style: "bold"}) {
		t.Fatalf("range = %#v, want clamped UTF-16 range", ranges[0])
	}
}

func TestRenderClampsOverlongSpanRangeToTextEnd(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "paragraph",
				Text: "A 😀",
				Spans: []article.Span{
					{Offset: len("A "), Length: 100, Style: "italic"},
				},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	ranges := state.Blocks[0].InlineStyleRanges
	if got := len(ranges); got != 1 {
		t.Fatalf("len(InlineStyleRanges) = %d, want 1: %#v", got, ranges)
	}
	if ranges[0] != (StyleRange{Offset: 2, Length: 2, Style: "italic"}) {
		t.Fatalf("range = %#v, want clamped UTF-16 emoji range", ranges[0])
	}
}

func TestMarshalUsesDraftJSSnakeCaseKeys(t *testing.T) {
	state := ContentState{
		Blocks: []Block{
			{
				Key:  "block-1",
				Text: "Hello image",
				Type: "atomic",
				InlineStyleRanges: []StyleRange{
					{Offset: 0, Length: 5, Style: "bold"},
				},
				EntityRanges: []EntityRange{
					{Offset: 6, Length: 1, Key: 0},
				},
			},
		},
		Entities: []Entity{
			{
				Key: "0",
				Value: EntityValue{
					Type:       "image",
					Mutability: "IMMUTABLE",
					Data: EntityData{
						MediaItems: []MediaItem{
							{MediaCategory: "tweet_image", MediaID: "media-123"},
						},
					},
				},
			},
		},
	}

	payload, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	body := string(payload)

	for _, want := range []string{
		`"inline_style_ranges"`,
		`"entity_ranges"`,
		`"media_items"`,
		`"media_id"`,
		`"media_category"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("marshaled JSON missing %s: %s", want, body)
		}
	}

	for _, unwanted := range []string{
		`"Offset"`,
		`"Length"`,
		`"Style"`,
		`"MediaItems"`,
		`"MediaID"`,
		`"MediaCategory"`,
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("marshaled JSON contains Go field name %s: %s", unwanted, body)
		}
	}
}

func TestRenderRegressionStylesLinksAndImageFallback(t *testing.T) {
	doc := &article.Document{
		Blocks: []article.Block{
			{
				Type: "paragraph",
				Text: "A 😀 link",
				Spans: []article.Span{
					{Offset: len("A "), Length: len("😀"), Style: "strikethrough"},
					{Offset: len("A 😀 "), Length: len("link"), Style: "link", URL: "https://example.com"},
				},
			},
			{
				Type: "image",
				Text: "No uploaded media",
				Data: map[string]string{"source": "./image.png"},
			},
		},
	}

	state, err := Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	ranges := state.Blocks[0].InlineStyleRanges
	if got := len(ranges); got != 1 {
		t.Fatalf("len(InlineStyleRanges) = %d, want 1: %#v", got, ranges)
	}
	if ranges[0] != (StyleRange{Offset: 2, Length: 2, Style: "strikethrough"}) {
		t.Fatalf("strikethrough range = %#v, want UTF-16 emoji range", ranges[0])
	}
	entityRanges := state.Blocks[0].EntityRanges
	if got := len(entityRanges); got != 1 {
		t.Fatalf("len(EntityRanges) = %d, want 1: %#v", got, entityRanges)
	}
	if entityRanges[0].Offset != 5 || entityRanges[0].Length != 4 {
		t.Fatalf("link range = %#v, want UTF-16 offset 5 length 4", entityRanges[0])
	}
	if state.Blocks[1].Type != "unstyled" {
		t.Fatalf("image without media_id Type = %q, want unstyled", state.Blocks[1].Type)
	}
	if got := len(state.Blocks[1].EntityRanges); got != 0 {
		t.Fatalf("image without media_id EntityRanges = %d, want 0", got)
	}
	if got := len(state.Entities); got != 1 {
		t.Fatalf("len(Entities) = %d, want only link entity", got)
	}
}
