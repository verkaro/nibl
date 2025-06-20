// internal/builder/builder.go
package builder

import (
	"fmt"
	"html/template"
	"io"
	"nibl/internal/config"
	"nibl/internal/util"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type BuildOptions struct {
	CleanDestination bool
	Unsafe           bool
	Debug            bool
}

// BuildSite processes content files, renders them into HTML pages, and copies static assets.
func BuildSite(outputDir, contentDir, staticDir string, site config.SiteConfig, tmpl *template.Template, opts BuildOptions) (int, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, err
	}

	if opts.CleanDestination {
		fmt.Println("Cleaning destination directory...")
		entries, err := os.ReadDir(outputDir)
		if err != nil {
			return 0, err
		}
		for _, entry := range entries {
			if err := os.RemoveAll(filepath.Join(outputDir, entry.Name())); err != nil {
				return 0, err
			}
		}
	}

	pagesGenerated := 0
	if err := filepath.Walk(contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext != ".html" && ext != ".md" {
			return nil
		}

		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}
		if !utf8.Valid(contentBytes) {
			return fmt.Errorf("content file is not valid UTF-8: %s", path)
		}

		meta, htmlOut, parseErr := processContent(contentBytes, opts)
		if parseErr != nil {
			return fmt.Errorf("failed to process content for %s: %w", path, parseErr)
		}

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}

		if meta.Draft && !isExceptionPage(strings.TrimSuffix(relPath, ext)) {
			return nil
		}

		outputPath := filepath.Join(outputDir, strings.TrimSuffix(relPath, ext)+".html")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		pageData := PageData{
			Content:     template.HTML(htmlOut),
			Title:       meta.Title,
			BaseHref:    util.ComputeBaseHref(relPath),
			Description: meta.Description,
			Site:        site,
			ShowEditML:  meta.ShowEditML,
			StoryTitle:  meta.StoryTitle,
			Params:      meta.Params, // Pass arbitrary params to the template
		}

		if meta.StoryAuthor != "" {
			pageData.Author = meta.StoryAuthor
		} else {
			pageData.Author = site.Author
		}
		if pageData.Description == "" {
			pageData.Description = site.Description
		}

		if err := renderPage(tmpl, outputPath, pageData); err != nil {
			return fmt.Errorf("failed to render page %s: %w", path, err)
		}
		pagesGenerated++
		return nil
	}); err != nil {
		return 0, err
	}

	if err := copyStaticAssets(staticDir, outputDir); err != nil {
		return 0, err
	}
	return pagesGenerated, nil
}

// copyStaticAssets copies files from the static directory to the output directory.
func copyStaticAssets(staticDir, outputDir string) error {
	// This map defines the file extensions that are considered "static assets".
	// You can add or remove extensions here as needed (e.g., ".woff", ".woff2").
	allowedExts := map[string]bool{
		".css": true, ".js": true, ".txt": true, ".svg": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	}
	return filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Skip files with extensions that are not in our allowed list.
		if !allowedExts[filepath.Ext(info.Name())] {
			return nil
		}

		rel, err := filepath.Rel(staticDir, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(outputDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer dst.Close()
		_, err = io.Copy(dst, src)
		return err
	})
}

// isExceptionPage checks for pages that should not be considered drafts.
func isExceptionPage(slug string) bool {
	// This can be expanded with other "special" pages like "404" or "sitemap".
	return slug == "index" || slug == "about" || slug == "menu"
}

// renderPage executes the Go template and writes the output to a file.
func renderPage(tmpl *template.Template, outPath string, data PageData) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	// "main" is the name of the template defined within our layout file.
	return tmpl.ExecuteTemplate(outFile, "main", data)
}

// LoadTemplates parses all necessary template files from a given theme directory.
func LoadTemplates(templateDir, templateName string) (*template.Template, error) {
	path := filepath.Join(templateDir, templateName)
	// This function assumes a specific structure for templates:
	// a layout file and partials for header/footer.
	tmpl, err := template.ParseFiles(
		filepath.Join(path, "layout.html"),
		filepath.Join(path, "header.html"),
		filepath.Join(path, "footer.html"),
	)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

