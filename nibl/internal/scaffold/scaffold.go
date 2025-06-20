// internal/scaffold/scaffold.go
package scaffold

import (
	"bytes"
	"fmt"
	"nibl/internal/config"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// CreateNewSite is now fully restored.
func CreateNewSite(name string) error {
	fmt.Println("Scaffolding new site in:", name)
	mkdir := func(path string) error { return os.MkdirAll(filepath.Join(name, path), 0755) }
	writeFile := func(path, content string) error {
		return os.WriteFile(filepath.Join(name, path), []byte(content), 0644)
	}
	dirs := []string{"content", "static/css", "static/js", "static/images", "templates/simple", "archetypes"}
	for _, dir := range dirs {
		if err := mkdir(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	files := map[string]string{
		"site.yaml":                     siteYamlContent,
		"site.biff":                     siteBiffContent,
		"static/css/style.css":          staticCssContent,
		"templates/simple/layout.html":  templateLayoutHtmlContent,
		"templates/simple/header.html":  templateHeaderHtmlContent,
		"templates/simple/footer.html":  templateFooterHtmlContent,
		"archetypes/default.md":         archetypeDefaultMdContent,
	}
	for path, content := range files {
		if err := writeFile(path, content); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}
	fmt.Println("Site scaffolded. You can now:")
	fmt.Println("  cd", name)
	fmt.Println("  nibl story")
	fmt.Println("  nibl serve")
	return nil
}

// CreateNewContent is now fully restored.
func CreateNewContent(contentType, title, configPath string) error {
	slug := strings.ToLower(strings.ReplaceAll(title, " ", "-"))
	site, err := config.LoadSiteConfig(configPath)
	if err != nil {
		return err
	}

	path := filepath.Join("content", contentType, slug+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	archetypePath := filepath.Join("archetypes", "default.md")
	tmplBytes, err := os.ReadFile(archetypePath)
	if err != nil {
		return fmt.Errorf("could not read archetype file %s: %w", archetypePath, err)
	}

	tmpl, err := template.New("archetype").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse archetype file %s: %w", archetypePath, err)
	}

	data := struct {
		Title  string
		Author string
	}{
		Title:  title,
		Author: site.Author,
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return fmt.Errorf("failed to execute archetype template: %w", err)
	}

	if err := os.WriteFile(path, output.Bytes(), 0644); err != nil {
		return err
	}

	fmt.Println("Created:", path)
	return nil
}

// Constants for default file contents
const siteYamlContent = `title: My Interactive Story
author: Your Name
baseurl: /
description: A new story powered by nibl.
template: simple
`
const siteBiffContent = `// title: My Enchanted Garden
// author: A. Writer 
// description: A mazing site.
// STATES: has_water, has_seed
// FLAG-STATES: unlocked_gate, puzzle_solved
// LOCAL-STATES: door

=== index ===
// title: Home
You are at the start.
* Go outside -> outside

=== outside ===
// title: The Great Outdoors
- {door == true}
  You are outside. This is the end.
  Hope you had fun

END
`

const archetypeDefaultMdContent = `---
title: {{.Title}}
author: {{.Author}}
description: 
---

Write something meaningful here.
`

const staticCssContent = `body {
  font-family: sans-serif;
  max-width: 700px;
  margin: 2em auto;
  padding: 0 1em;
  line-height: 1.6;
  color: #222;
  background: #fdfdfd;
}
.header-line {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 1em;
  margin-bottom: 2em;
  flex-wrap: wrap;
}
.site-name { font-size: 0.9em; color: #777; font-style: italic; flex-grow: 1; text-align: left; }
.story-title { font-size: 1.2em; font-weight: 400; flex-grow: 1; text-align: center; }
.story-author { font-size: 0.9em; color: #777; font-style: italic; flex-grow: 1; text-align: right; }
main { margin-bottom: 3em; }
footer { text-align: center; font-size: 0.9em; color: #555; }
footer nav a { color: #444; text-decoration: none; margin: 0 0.5em; }
footer nav a:hover { text-decoration: underline; }
ul { margin-left: 1.2em; padding-left: 1.2em; list-style-type: disc; }
li { margin-bottom: 0.25em; }
hr { border: none; border-top: 1px solid #ccc; width: 33%; margin: 2em auto; }
`
const templateLayoutHtmlContent = `{{ define "main" }}
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{ .Title }} | {{ if .StoryTitle }}{{ .StoryTitle }}{{ else }}{{ .Site.Title }}{{ end }}</title>
  <link rel="stylesheet" href="{{ .BaseHref }}css/style.css">
{{ if .Description }}
  <meta name="description" content="{{ .Description }}">
{{ else }}
  <meta name="description" content="{{ .Site.Description }}">
{{ end }}
{{ if .ShowEditML }}
<style>
  .cm-add { background-color: #d4edda; color: #155724; }
  .cm-del { background-color: #f8d7da; color: #721c24; text-decoration: line-through; }
  .cm-hl { background-color: #fff3cd; color: #856404; }
  .cm-com { background-color: #eae3d3; color: #6e4c1e; font-style: italic; }
</style>
{{ end }}
</head>
<body>
  {{ template "header" . }}
  <main>
    {{ .Content }}
  </main>
  {{ template "footer" . }}
</body>
</html>
{{ end }}`

const templateHeaderHtmlContent = `{{ define "header" }}
<header>
  <div class="header-line">
    <div class="site-name">{{ .Site.Title }}</div>
    {{/* Use the global story title if it exists, otherwise fallback to site title */}}
    <div class="story-title">{{ if .StoryTitle }}{{ .StoryTitle }}{{ else }}{{ .Site.Title }}{{ end }}</div>
    {{/* Display the author, preferring the biff author over the site author */}}
    {{ if .Author }}<div class="story-author">{{ .Author }}</div>{{ end }}
  </div>
</header>
{{ end }}`

const templateFooterHtmlContent = `{{ define "footer" }}
<footer>
  <nav>
    <a href="{{ .BaseHref }}index.html">home</a>
  </nav>
  <div class="copyright">
    &copy; {{ .Site.Title }}
  </div>
</footer>
{{ end }}`

