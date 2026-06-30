package draftjs

type ContentState struct {
	// X Articles expects this shape, not DraftJS raw export's entityMap.
	Blocks   []Block  `json:"blocks"`
	Entities []Entity `json:"entities"`
}

type Block struct {
	Key               string        `json:"key,omitempty"`
	Text              string        `json:"text"`
	Type              string        `json:"type"`
	Data              interface{}   `json:"data,omitempty"`
	InlineStyleRanges []StyleRange  `json:"inline_style_ranges,omitempty"`
	EntityRanges      []EntityRange `json:"entity_ranges,omitempty"`
}

type StyleRange struct {
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Style  string `json:"style"`
}

type EntityRange struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
	Key    int `json:"key"`
}

type Entity struct {
	Key   string      `json:"key"`
	Value EntityValue `json:"value"`
}

type EntityValue struct {
	Type       string     `json:"type"`
	Mutability string     `json:"mutability"`
	Data       EntityData `json:"data"`
}

type EntityData struct {
	URL        string      `json:"url,omitempty"`
	PostID     string      `json:"post_id,omitempty"`
	Caption    string      `json:"caption,omitempty"`
	MediaItems []MediaItem `json:"media_items,omitempty"`
}

type MediaItem struct {
	MediaCategory string `json:"media_category"`
	MediaID       string `json:"media_id"`
}
