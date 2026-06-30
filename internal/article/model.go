package article

type Document struct {
	SourcePath string
	Title      string
	Cover      string
	Blocks     []Block
	Assets     []Asset
	Warnings   []Diagnostic
}

type Block struct {
	Type  string
	Level int
	Text  string
	Spans []Span
	Data  map[string]string
}

type Span struct {
	Offset int
	Length int
	Style  string
	URL    string
}

type Asset struct {
	Index  int
	Source string
	Alt    string
	Role   string
}

type Diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Block    int    `json:"block,omitempty"`
}
