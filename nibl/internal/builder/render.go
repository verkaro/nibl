// internal/builder/render.go
package builder

import (
	"bytes"
	"fmt"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
)

var (
	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(newMDLinkTransformer(), 100),
			),
		),
		goldmark.WithRendererOptions(
			// The WithHardWraps() option that caused incorrect line breaks has been removed.
			html.WithUnsafe(),
		),
	)
	htmlSanitizer = bluemonday.UGCPolicy()
)

// processContent now has a simplified pipeline. It expects the markdown body
// to have been pre-processed and is only responsible for rendering it to HTML.
func processContent(rawContent []byte, opts BuildOptions) (PageMeta, string, error) {
	meta := PageMeta{}

	// Step 1: Separate front matter from the markdown body.
	parts := bytes.SplitN(rawContent, []byte("---"), 3)
	var body string

	if len(parts) >= 3 {
		if err := yaml.Unmarshal(parts[1], &meta); err != nil {
			return PageMeta{}, "", fmt.Errorf("failed to parse front matter: %w", err)
		}
		body = string(parts[2])
	} else {
		body = string(rawContent)
	}

	// Step 2: Render the markdown body to HTML using Goldmark.
	var htmlBuffer bytes.Buffer
	if err := markdownRenderer.Convert([]byte(body), &htmlBuffer); err != nil {
		return meta, "", fmt.Errorf("failed to render markdown with goldmark: %w", err)
	}

	// Step 3: Sanitize the final HTML unless the --unsafe flag is used.
	if !opts.Unsafe {
		sanitizedHTML := htmlSanitizer.SanitizeBytes(htmlBuffer.Bytes())
		return meta, string(sanitizedHTML), nil
	}

	// The return statement is now corrected to match the function signature.
	return meta, htmlBuffer.String(), nil
}

