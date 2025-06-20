// internal/builder/models.go
package builder

import (
	"html/template"
	"nibl/internal/config"
)

// PageMeta holds metadata from front matter. It now includes a map
// for arbitrary parameters defined in the source markdown or biff file.
type PageMeta struct {
	Title       string                 `yaml:"title"`
	Author      string                 `yaml:"author"` // Per-page author (fallback)
	Draft       bool                   `yaml:"draft"`
	Description string                 `yaml:"description"`
	ShowEditML  bool                   `yaml:"showEditML"`
	StoryTitle  string                 `yaml:"story_title"`  // Global story title from biff
	StoryAuthor string                 `yaml:"story_author"` // Global story author from biff
	Params      map[string]interface{} `yaml:",inline"`
}

// PageData is the struct passed to templates. It now includes the
// arbitrary parameters, making them available in templates via `.Params`.
type PageData struct {
	Content     template.HTML
	Title       string // The title of the specific page/knot
	BaseHref    string
	Author      string // The final author to be displayed
	Description string
	Site        config.SiteConfig
	ShowEditML  bool
	StoryTitle  string // The global title of the story
	Params      map[string]interface{}
}

