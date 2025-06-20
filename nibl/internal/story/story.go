// internal/story/story.go
package story

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"nibl/internal/config"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/verkaro/bigif/bigif"
	"github.com/verkaro/editml-go"
)

// preParseBiffForFrontMatter reads the raw .biff file content before compilation
// to extract front matter from comments associated with each knot.
func preParseBiffForFrontMatter(biffData []byte) (map[string]map[string]string, error) {
	data := make(map[string]map[string]string)
	var currentKnotName string
	knotRegex := regexp.MustCompile(`^\s*===\s*([\w-]+)\s*===\s*$`)

	scanner := bufio.NewScanner(bytes.NewReader(biffData))
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimFunc(line, unicode.IsSpace)

		if matches := knotRegex.FindStringSubmatch(trimmedLine); len(matches) > 1 {
			currentKnotName = matches[1]
			if data[currentKnotName] == nil {
				data[currentKnotName] = make(map[string]string)
			}
			continue
		}

		if currentKnotName != "" && strings.HasPrefix(trimmedLine, "//") {
			commentContent := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "//"))
			parts := strings.SplitN(commentContent, ":", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimSpace(parts[0]))
				value := strings.TrimSpace(parts[1])
				data[currentKnotName][key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

// processKnotContent is the central function for handling a knot's body.
// It correctly processes EditML syntax and returns clean markdown ready for rendering.
func processKnotContent(rawContent string) (string, error) {
	nodes, parseIssues := editml.Parse(rawContent)
	if len(parseIssues) > 0 && parseIssues[0].Severity == editml.SeverityError {
		return "", fmt.Errorf("editml parsing error: %s", parseIssues[0].Message)
	}
	cleanMarkdown, transformIssues := editml.TransformCleanView(nodes)
	if len(transformIssues) > 0 && transformIssues[0].Severity == editml.SeverityError {
		return "", fmt.Errorf("editml transformation error: %s", transformIssues[0].Message)
	}
	return cleanMarkdown, nil
}

// extractTitleAndContent determines the final title for a page and separates
// the H1 title hint from the rest of the body.
func extractTitleAndContent(knotName, content string, knotMeta map[string]string) (string, string) {
	var title string
	var markdownTitle string
	var finalContentLines []string

	if val, ok := knotMeta["title"]; ok {
		title = val
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimFunc(line, unicode.IsSpace)
		if strings.HasPrefix(trimmedLine, "# ") {
			if markdownTitle == "" {
				markdownTitle = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "#"))
			}
		} else {
			finalContentLines = append(finalContentLines, line)
		}
	}
	pageContent := strings.TrimSpace(strings.Join(finalContentLines, "\n"))

	if title == "" && markdownTitle != "" {
		title = markdownTitle
	}
	if title == "" {
		title = strings.Title(strings.ReplaceAll(knotName, "_", " "))
	}

	return title, pageContent
}

// Compile is the main function that drives the biff-to-markdown process.
func Compile(biffPath, contentDir string, siteCfg config.SiteConfig) (int, error) {
	biffData, err := ioutil.ReadFile(biffPath)
	if err != nil {
		return 0, err
	}

	preParsedFrontMatter, err := preParseBiffForFrontMatter(biffData)
	if err != nil {
		return 0, fmt.Errorf("failed to pre-parse biff for front matter: %w", err)
	}

	jsonBytes, err := bigif.Compile(string(biffData))
	if err != nil {
		return 0, fmt.Errorf("biff syntax error: %w", err)
	}

	var intermediate struct {
		Metadata map[string]string            `json:"metadata"`
		Graph    struct{ Nodes map[string]*bigif.StoryNode `json:"nodes"` } `json:"graph"`
	}

	if err := json.Unmarshal(jsonBytes, &intermediate); err != nil {
		return 0, fmt.Errorf("internal error: failed to unmarshal story json: %w", err)
	}

	paths := buildPaths(intermediate.Graph.Nodes, contentDir)
	filesWritten := 0
	for id, node := range intermediate.Graph.Nodes {
		targetPath := paths[id]
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return 0, fmt.Errorf("failed to create directory for story file: %w", err)
		}

		file, err := os.Create(targetPath)
		if err != nil {
			return 0, fmt.Errorf("failed to create story file %s: %w", targetPath, err)
		}
		defer file.Close()

		knotMeta := preParsedFrontMatter[node.KnotName]
		if knotMeta == nil {
			knotMeta = make(map[string]string)
		}

		displayTitle, rawPageContent := extractTitleAndContent(node.KnotName, node.Content, knotMeta)

		finalPageContent, err := processKnotContent(rawPageContent)
		if err != nil {
			return 0, fmt.Errorf("failed to process content for knot %s: %w", node.KnotName, err)
		}

		writeFrontMatter(file, &intermediate.Metadata, displayTitle, knotMeta)

		fmt.Fprintf(file, "## %s\n\n", displayTitle)
		fmt.Fprintln(file, finalPageContent)
		fmt.Fprintln(file)
		if len(node.Edges) > 0 {
			// The "## Choices" heading has been commented out as requested.
			// fmt.Fprintln(file, "## Choices")
			for _, edge := range node.Edges {
				rel, _ := filepath.Rel(filepath.Dir(targetPath), paths[edge.TargetNodeID])
				rel = filepath.ToSlash(rel)
				fmt.Fprintf(file, "* [%s](%s)\n", edge.Text, rel)
			}
		}
		filesWritten++
	}

	return filesWritten, nil
}

// writeFrontMatter writes the YAML front matter to the file.
func writeFrontMatter(f *os.File, storyMeta *map[string]string, displayTitle string, knotMeta map[string]string) {
	fmt.Fprintln(f, "---")
	fmt.Fprintf(f, "title: \"%s\"\n", strings.ReplaceAll(displayTitle, "\"", "\\\""))

	if st, ok := (*storyMeta)["title"]; ok {
		fmt.Fprintf(f, "story_title: \"%s\"\n", strings.ReplaceAll(st, "\"", "\\\""))
	}
	if sa, ok := (*storyMeta)["author"]; ok {
		fmt.Fprintf(f, "story_author: \"%s\"\n", strings.ReplaceAll(sa, "\"", "\\\""))
	}

	for key, value := range knotMeta {
		if key != "title" {
			fmt.Fprintf(f, "%s: \"%s\"\n", key, strings.ReplaceAll(value, "\"", "\\\""))
		}
	}

	fmt.Fprintln(f, "draft: false")
	fmt.Fprintln(f, "---")
}

func buildPaths(nodes map[string]*bigif.StoryNode, outDir string) map[string]string {
	paths := make(map[string]string)
	for id, node := range nodes {
		dirs := []string{outDir}
		if node.Scene != "" {
			for _, seg := range strings.Split(node.Scene, "/") {
				dirs = append(dirs, sanitize(seg))
			}
		}
		parts := []string{sanitize(node.KnotName)}
		var flags []string
		for k, v := range node.State {
			if v {
				flags = append(flags, sanitize(k))
			}
		}
		sort.Strings(flags)
		parts = append(parts, flags...)
		filename := strings.Join(parts, "-") + ".md"
		paths[id] = filepath.Join(append(dirs, filename)...)
	}
	return paths
}

func sanitize(s string) string {
	s = strings.ToLower(s)
	s = regexp.MustCompile(`[^\w- ]+`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, " ", "-")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return s
}

