package context

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type ProjectContext struct {
	Frameworks []string
	Colors     map[string]string // map[hex]name
	Components []string
	DirTree    string
}

func ScanContext(targetDir string) (*ProjectContext, error) {
	if targetDir == "" {
		return nil, nil
	}

	info, err := os.Stat(targetDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("invalid target directory: %s", targetDir)
	}

	pc := &ProjectContext{
		Colors: make(map[string]string),
	}

	// 1. Scan package.json for frameworks
	pkgPath := filepath.Join(targetDir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			detectFrameworks(pkg.Dependencies, pkg.DevDependencies, pc)
		}
	}

	// 2. Scan tailwind.config.js for colors
	twPath := filepath.Join(targetDir, "tailwind.config.js")
	if data, err := os.ReadFile(twPath); err == nil {
		parseTailwindColors(string(data), pc)
	} else {
		twPath = filepath.Join(targetDir, "tailwind.config.ts")
		if data, err := os.ReadFile(twPath); err == nil {
			parseTailwindColors(string(data), pc)
		}
	}

	// 3. Scan src/components
	compDir := filepath.Join(targetDir, "src", "components")
	if _, err := os.Stat(compDir); err == nil {
		filepath.WalkDir(compDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if ext == ".tsx" || ext == ".jsx" || ext == ".vue" || ext == ".svelte" {
				name := strings.TrimSuffix(d.Name(), ext)
				if name != "index" {
					pc.Components = append(pc.Components, name)
				}
			}
			return nil
		})
	}

	// 4. Linux Specific: Get Directory Tree Context
	if _, err := os.Stat("/usr/bin/find"); err == nil {
		srcDir := filepath.Join(targetDir, "src")
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			srcDir = targetDir
		}
		cmd := exec.Command("find", srcDir, "-maxdepth", "4", "-type", "f", "-not", "-path", "*/node_modules/*", "-not", "-path", "*/.git/*")
		if out, err := cmd.Output(); err == nil {
			pc.DirTree = string(out)
		}
	}

	return pc, nil
}

func detectFrameworks(deps, devDeps map[string]string, pc *ProjectContext) {
	frameworks := []string{"next", "react", "vue", "svelte", "tailwindcss"}
	for _, f := range frameworks {
		if _, ok := deps[f]; ok {
			pc.Frameworks = append(pc.Frameworks, f)
		} else if _, ok := devDeps[f]; ok {
			pc.Frameworks = append(pc.Frameworks, f)
		}
	}
}

func parseTailwindColors(content string, pc *ProjectContext) {
	re := regexp.MustCompile(`['"]?([a-zA-Z0-9_-]+)['"]?\s*:\s*['"](#[0-9a-fA-F]{3,8})['"]`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) == 3 {
			name := m[1]
			hexStr := strings.ToLower(m[2])
			pc.Colors[hexStr] = name
		}
	}
}

func (pc *ProjectContext) FormatForLLM() string {
	if pc == nil || (len(pc.Frameworks) == 0 && len(pc.Colors) == 0 && len(pc.Components) == 0) {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n### LOCAL PROJECT CONTEXT ###\n")
	sb.WriteString("You MUST respect these local project configurations when generating the blueprint:\n")
	if len(pc.Frameworks) > 0 {
		sb.WriteString(fmt.Sprintf("1. Frameworks in use: %s\n", strings.Join(pc.Frameworks, ", ")))
	}
	if len(pc.Colors) > 0 {
		sb.WriteString("2. Custom Tailwind Colors (Use these names instead of raw hex values!):\n")
		for hexStr, name := range pc.Colors {
			sb.WriteString(fmt.Sprintf("   - %s (hex: %s)\n", name, hexStr))
		}
	}
	if len(pc.Components) > 0 {
		sb.WriteString("3. Existing Components (If you see elements that match these names, DO NOT rebuild them, simply reference them):\n")
		sb.WriteString("   - " + strings.Join(pc.Components, ", ") + "\n")
	}
	if pc.DirTree != "" {
		sb.WriteString("\n### WORKSPACE DIRECTORY TREE (LINUX CONTEXT) ###\n")
		sb.WriteString("Review this tree to understand where files and existing components are located in the local system:\n```\n")
		tree := pc.DirTree
		if len(tree) > 2000 {
			tree = tree[:2000] + "\n... (truncated)"
		}
		sb.WriteString(tree)
		sb.WriteString("\n```\n")
	}
	return sb.String()
}
