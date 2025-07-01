package project

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hammie/rubrduck/internal/config"
)

type FileInfo struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
}

type ProjectInfo struct {
	Languages  map[string]int `json:"languages"`
	Frameworks []string       `json:"frameworks"`
	Configs    []string       `json:"configs"`
	Files      []FileInfo     `json:"files"`
}

// AnalyzeProject scans the given directory and returns information about the codebase
func AnalyzeProject(basePath string, cfg config.ProjectConfig) (*ProjectInfo, error) {
	info := &ProjectInfo{
		Languages: make(map[string]int),
	}

	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil
		}
		if shouldIgnore(rel, cfg.IgnorePaths) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(rel))
		lang := languageFromExt(ext, cfg.CodeExtensions)
		if lang != "" {
			info.Languages[lang]++
		}
		basename := filepath.Base(rel)
		switch basename {
		case "package.json":
			info.Frameworks = append(info.Frameworks, detectNodeFrameworks(path)...)
			info.Configs = append(info.Configs, rel)
		case "go.mod":
			info.Frameworks = append(info.Frameworks, "go")
			info.Configs = append(info.Configs, rel)
		case "pyproject.toml", "requirements.txt":
			info.Frameworks = append(info.Frameworks, "python")
			info.Configs = append(info.Configs, rel)
		case "composer.json":
			info.Frameworks = append(info.Frameworks, "php")
			info.Configs = append(info.Configs, rel)
		}
		stat, err := os.Stat(path)
		if err == nil {
			info.Files = append(info.Files, FileInfo{Path: rel, Size: stat.Size(), Language: lang})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	info.Frameworks = uniqueStrings(info.Frameworks)
	info.Configs = uniqueStrings(info.Configs)
	return info, nil
}

func shouldIgnore(path string, patterns []string) bool {
	for _, p := range patterns {
		if strings.HasPrefix(path, p) {
			return true
		}
		if ok, _ := filepath.Match(p, filepath.Base(path)); ok {
			return true
		}
	}
	return false
}

func languageFromExt(ext string, allowed []string) string {
	if ext == "" {
		return ""
	}
	for _, a := range allowed {
		if ext == a {
			switch ext {
			case ".go":
				return "Go"
			case ".js", ".ts":
				return "JavaScript"
			case ".py":
				return "Python"
			case ".java":
				return "Java"
			case ".cpp", ".c", ".h":
				return "C/C++"
			case ".rs":
				return "Rust"
			case ".rb":
				return "Ruby"
			case ".php":
				return "PHP"
			}
		}
	}
	return ""
}

func detectNodeFrameworks(path string) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var data struct {
		Dependencies map[string]string `json:"dependencies"`
		DevDeps      map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil
	}
	frameworks := []string{"node"}
	for dep := range data.Dependencies {
		switch dep {
		case "react", "vue", "angular", "svelte", "express":
			frameworks = append(frameworks, dep)
		}
	}
	return frameworks
}

func uniqueStrings(in []string) []string {
	m := make(map[string]struct{})
	var out []string
	for _, s := range in {
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
