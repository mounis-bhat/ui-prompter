package figma

type FileNodesResponse struct {
	Name  string                 `json:"name"`
	Nodes map[string]NodeContent `json:"nodes"`
}

type ImagesResponse struct {
	Images map[string]string `json:"images"`
}

type NodeContent struct {
	Document Node `json:"document"`
}

type Node struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Type             string       `json:"type"`
	Characters       string       `json:"characters,omitempty"`
	Children         []Node       `json:"children,omitempty"`
	LayoutMode       string       `json:"layoutMode,omitempty"`
	PrimaryAxisAlign string       `json:"primaryAxisAlignItems,omitempty"`
	CounterAxisAlign string       `json:"counterAxisAlignItems,omitempty"`
	PaddingLeft      float64      `json:"paddingLeft,omitempty"`
	PaddingRight     float64      `json:"paddingRight,omitempty"`
	PaddingTop       float64      `json:"paddingTop,omitempty"`
	PaddingBottom    float64      `json:"paddingBottom,omitempty"`
	ItemSpacing      float64      `json:"itemSpacing,omitempty"`
	LayoutGrow       float64      `json:"layoutGrow,omitempty"`
	LayoutAlign      string       `json:"layoutAlign,omitempty"`
	Fills            []Paint      `json:"fills,omitempty"`
	Strokes          []Paint          `json:"strokes,omitempty"`
	CornerRadius     float64          `json:"cornerRadius,omitempty"`
	BoundingBox      *BoundingBox     `json:"absoluteBoundingBox,omitempty"`
	ExportSettings   []ExportSetting  `json:"exportSettings,omitempty"`
}

type ExportSetting struct {
	Format string `json:"format"`
}

type AssetInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Format string `json:"format"`
	URL    string `json:"url"`
}

type BoundingBox struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Paint struct {
	Type  string `json:"type"`
	Color *Color `json:"color,omitempty"`
}

type Color struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
	A float64 `json:"a"`
}
