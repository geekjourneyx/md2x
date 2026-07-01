package draftjs

import (
	"fmt"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/geekjourneyx/md2x/internal/article"
)

func Render(doc *article.Document) (*ContentState, error) {
	if doc == nil {
		return nil, fmt.Errorf("draftjs: nil document")
	}

	state := &ContentState{
		Blocks:   make([]Block, 0, len(doc.Blocks)),
		Entities: make([]Entity, 0),
	}
	for i, source := range doc.Blocks {
		block := Block{
			Key:  strconv.FormatInt(int64(i+1), 36),
			Text: source.Text,
			Type: blockType(source),
		}
		if includeBlockData(source) {
			block.Data = source.Data
		}

		if source.Type == "image" {
			if mediaID := source.Data["media_id"]; mediaID != "" {
				block.Type = "atomic"
				block.Text = " "
				block.Data = nil
				entityKey := len(state.Entities)
				mediaCategory := source.Data["media_category"]
				if mediaCategory == "" {
					mediaCategory = "tweet_image"
				}
				state.Entities = append(state.Entities, Entity{
					Key: strconv.Itoa(entityKey),
					Value: EntityValue{
						Type:       "image",
						Mutability: "immutable",
						Data: EntityData{
							Caption: source.Text,
							MediaItems: []MediaItem{
								{MediaCategory: mediaCategory, MediaID: mediaID},
							},
						},
					},
				})
				block.EntityRanges = append(block.EntityRanges, EntityRange{Offset: 0, Length: 1, Key: entityKey})
			}
			state.Blocks = append(state.Blocks, block)
			continue
		}

		for _, span := range source.Spans {
			offset, length, ok, err := utf16Range(source.Text, span.Offset, span.Length)
			if err != nil {
				return nil, fmt.Errorf("draftjs: block %d span %q has invalid byte range: %w", i, span.Style, err)
			}
			if !ok {
				continue
			}
			switch span.Style {
			case "bold":
				block.InlineStyleRanges = append(block.InlineStyleRanges, StyleRange{Offset: offset, Length: length, Style: "bold"})
			case "italic":
				block.InlineStyleRanges = append(block.InlineStyleRanges, StyleRange{Offset: offset, Length: length, Style: "italic"})
			case "strikethrough":
				block.InlineStyleRanges = append(block.InlineStyleRanges, StyleRange{Offset: offset, Length: length, Style: "strikethrough"})
			case "link":
				entityKey := len(state.Entities)
				state.Entities = append(state.Entities, Entity{
					Key: strconv.Itoa(entityKey),
					Value: EntityValue{
						Type:       "link",
						Mutability: "mutable",
						Data:       EntityData{URL: span.URL},
					},
				})
				block.EntityRanges = append(block.EntityRanges, EntityRange{Offset: offset, Length: length, Key: entityKey})
			}
		}
		state.Blocks = append(state.Blocks, block)
	}

	return state, nil
}

func includeBlockData(block article.Block) bool {
	return block.Type == "image" && len(block.Data) > 0
}

func blockType(block article.Block) string {
	switch block.Type {
	case "heading":
		switch block.Level {
		case 1:
			return "header-one"
		case 2:
			return "header-two"
		default:
			return "header-three"
		}
	case "unordered-list-item", "ordered-list-item", "blockquote":
		return block.Type
	default:
		return "unstyled"
	}
}

func utf16Range(text string, byteOffset, byteLength int) (int, int, bool, error) {
	start, ok := validBoundary(text, byteOffset)
	if !ok {
		return 0, 0, false, fmt.Errorf("offset %d is inside a UTF-8 rune", byteOffset)
	}
	endByte := byteOffset + byteLength
	end, ok := validBoundary(text, endByte)
	if !ok {
		return 0, 0, false, fmt.Errorf("end offset %d is inside a UTF-8 rune", endByte)
	}
	if end <= start {
		return 0, 0, false, nil
	}

	offset := utf16Length(text[:start])
	length := utf16Length(text[start:end])
	if length <= 0 {
		return 0, 0, false, nil
	}
	return offset, length, true, nil
}

func validBoundary(text string, index int) (int, bool) {
	if index <= 0 {
		return 0, true
	}
	if index >= len(text) {
		return len(text), true
	}
	if !utf8.RuneStart(text[index]) {
		return 0, false
	}
	return index, true
}

func utf16Length(text string) int {
	var length int
	for _, r := range text {
		if r <= utf8.RuneSelf {
			length++
			continue
		}
		length += len(utf16.Encode([]rune{r}))
	}
	return length
}
