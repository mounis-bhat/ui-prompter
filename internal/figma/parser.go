package figma

import (
	"fmt"
	"strings"
)

func rgbToHex(c Color) string {
	r := int(c.R * 255)
	g := int(c.G * 255)
	b := int(c.B * 255)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func getClasses(node *Node) string {
	var classes []string

	if node.LayoutMode == "HORIZONTAL" {
		classes = append(classes, "flex-row")
	} else if node.LayoutMode == "VERTICAL" {
		classes = append(classes, "flex-col")
	}

	switch node.PrimaryAxisAlign {
	case "MIN":
		classes = append(classes, "justify-start")
	case "CENTER":
		classes = append(classes, "justify-center")
	case "MAX":
		classes = append(classes, "justify-end")
	case "SPACE_BETWEEN":
		classes = append(classes, "justify-between")
	}

	switch node.CounterAxisAlign {
	case "MIN":
		classes = append(classes, "items-start")
	case "CENTER":
		classes = append(classes, "items-center")
	case "MAX":
		classes = append(classes, "items-end")
	}

	// Padding
	px := node.PaddingLeft
	if node.PaddingRight != px {
		px = -1
	}
	py := node.PaddingTop
	if node.PaddingBottom != py {
		py = -1
	}

	if px > 0 && py > 0 && px == py {
		classes = append(classes, fmt.Sprintf("p-[%.0fpx]", px))
	} else {
		if px > 0 {
			classes = append(classes, fmt.Sprintf("px-[%.0fpx]", px))
		}
		if py > 0 {
			classes = append(classes, fmt.Sprintf("py-[%.0fpx]", py))
		}
	}

	if node.ItemSpacing > 0 {
		classes = append(classes, fmt.Sprintf("gap-[%.0fpx]", node.ItemSpacing))
	}
	if node.CornerRadius > 0 {
		classes = append(classes, fmt.Sprintf("rounded-[%.0fpx]", node.CornerRadius))
	}
	if node.LayoutGrow == 1 {
		classes = append(classes, "flex-1")
	}
	if node.LayoutAlign == "STRETCH" {
		classes = append(classes, "self-stretch")
	}

	for _, fill := range node.Fills {
		if fill.Type == "SOLID" && fill.Color != nil {
			classes = append(classes, fmt.Sprintf("bg-[%s]", rgbToHex(*fill.Color)))
		}
	}
	for _, stroke := range node.Strokes {
		if stroke.Type == "SOLID" && stroke.Color != nil {
			classes = append(classes, fmt.Sprintf("border-[%s]", rgbToHex(*stroke.Color)))
		}
	}

	if len(classes) == 0 {
		return ""
	}
	return " (" + strings.Join(classes, " ") + ")"
}

func ParseNodeToMarkdown(node *Node, depth int) string {
	if node.Type == "VECTOR" || node.Type == "ELLIPSE" || node.Type == "LINE" || node.Type == "STAR" || node.Type == "BOOLEAN_OPERATION" {
		return ""
	}

	indent := strings.Repeat("  ", depth)
	
	name := node.Name
	if node.Type == "TEXT" && node.Characters != "" {
		chars := strings.ReplaceAll(node.Characters, "\n", " ")
		name = fmt.Sprintf("Text: %q", chars)
	} else {
		if strings.HasPrefix(name, "Rectangle") || strings.HasPrefix(name, "Group") || strings.HasPrefix(name, "Frame") || name == "Mask group" || name == "Layer_1" {
			name = "Container"
		}
	}

	classes := getClasses(node)
	
	prefix := "-"
	if depth == 0 {
		prefix = "#"
		indent = ""
	} else if depth == 1 {
		prefix = "##"
	} else if depth == 2 {
		prefix = "###"
	}

	childMd := ""
	validChildrenCount := 0
	for _, child := range node.Children {
		res := ParseNodeToMarkdown(&child, depth+1)
		if res != "" {
			childMd += res
			validChildrenCount++
		}
	}

	if name == "Container" && classes == "" && validChildrenCount == 0 {
		return ""
	}

	return fmt.Sprintf("%s %s %s%s\n%s", indent, prefix, name, classes, childMd)
}

func ExtractAssets(node *Node) []AssetInfo {
	var assets []AssetInfo

	isAsset := false
	format := "png"

	if len(node.ExportSettings) > 0 {
		isAsset = true
		format = strings.ToLower(node.ExportSettings[0].Format)
	} else if node.Type == "VECTOR" || node.Type == "BOOLEAN_OPERATION" || node.Type == "STAR" || node.Type == "ELLIPSE" || node.Type == "LINE" {
		isAsset = true
		format = "svg"
	} else {
		for _, fill := range node.Fills {
			if fill.Type == "IMAGE" {
				isAsset = true
				format = "png"
				break
			}
		}
	}

	if isAsset {
		name := node.Name
		if name == "" {
			name = "asset"
		}
		name = strings.ReplaceAll(name, " ", "_")
		name = strings.ReplaceAll(name, "/", "_")
		name = strings.ToLower(name)
		
		safeID := strings.ReplaceAll(node.ID, ":", "-")
		name = fmt.Sprintf("%s_%s", name, safeID)

		assets = append(assets, AssetInfo{
			ID:     node.ID,
			Name:   name,
			Format: format,
		})
	} else {
		for _, child := range node.Children {
			assets = append(assets, ExtractAssets(&child)...)
		}
	}

	return assets
}
