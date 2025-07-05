package project

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo holds basic metadata about a file in the project
// including the detected programming language.
type FileInfo struct {
	Path     string
	Language string
}

// Analysis holds aggregate information about the project
// such as detected languages, frameworks, configuration files
// and an index of files.
type Analysis struct {
	Root        string
	Languages   map[string]int
	Frameworks  []string
	ConfigFiles []string
	Files       []FileInfo
}

// Analyze walks the project directory and collects context information.
// root should be the path to the project root directory.
func Analyze(root string) (*Analysis, error) {
	a := &Analysis{
		Root:      root,
		Languages: make(map[string]int),
	}

	// Track frameworks discovered via config files
	frameworks := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip common directories we don't want to index
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}

		// Detect config files and frameworks
		switch filepath.Base(path) {
		case "go.mod":
			a.ConfigFiles = append(a.ConfigFiles, rel)
			detectGoFrameworks(path, frameworks)
		case "package.json":
			a.ConfigFiles = append(a.ConfigFiles, rel)
			detectNodeFrameworks(path, frameworks)
		case "requirements.txt", "Gemfile", "pom.xml":
			a.ConfigFiles = append(a.ConfigFiles, rel)
		}

		lang := languageFromExt(filepath.Ext(path))
		if lang != "" {
			a.Languages[lang]++
			a.Files = append(a.Files, FileInfo{Path: rel, Language: lang})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for f := range frameworks {
		a.Frameworks = append(a.Frameworks, f)
	}

	return a, nil
}

// languageFromExt maps file extensions to language names
func languageFromExt(ext string) string {
	switch ext {
	case ".go":
		return "Go"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".java":
		return "Java"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".c", ".h":
		return "C"
	case ".cpp", ".cc", ".cxx":
		return "C++"
	case ".rs":
		return "Rust"
	default:
		return ""
	}
}

// detectGoFrameworks looks for common Go frameworks in go.mod
func detectGoFrameworks(path string, fw map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	content := string(data)
	if contains(content, "github.com/gin-gonic/gin") {
		fw["gin"] = true
	}
	if contains(content, "github.com/labstack/echo") {
		fw["echo"] = true
	}
}

// detectNodeFrameworks reads package.json and checks dependencies
func detectNodeFrameworks(path string, fw map[string]bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var pkg struct {
		Dependencies    map[string]interface{} `json:"dependencies"`
		DevDependencies map[string]interface{} `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}
	for dep := range pkg.Dependencies {
		switch dep {
		case "react", "next", "vue":
			fw[dep] = true
		case "express":
			fw["express"] = true
		}
	}
	for dep := range pkg.DevDependencies {
		switch dep {
		case "react", "next", "vue":
			fw[dep] = true
		case "express":
			fw["express"] = true
		}
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
