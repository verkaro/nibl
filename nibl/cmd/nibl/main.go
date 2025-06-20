// cmd/nibl/main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"nibl/internal/builder"
	"nibl/internal/config"
	"nibl/internal/scaffold"
	"nibl/internal/server"
	"nibl/internal/story"
	"os"
	"path/filepath"
	"strings"
)

type appConfig struct {
	debug  bool
	port   int
	unsafe bool
}

const (
	contentDir  = "content"
	templateDir = "templates"
	staticDir   = "static"
	outputDir   = "public"
	configFile  = "site.yaml"
	storyFile   = "site.biff"
)

func main() {
	appCfg := appConfig{}
	// Global flags
	flag.BoolVar(&appCfg.debug, "debug", false, "Enable debug mode for verbose error output.")
	flag.IntVar(&appCfg.port, "port", 1313, "Port for the local development server.")
	flag.BoolVar(&appCfg.unsafe, "unsafe", false, "Disable HTML sanitization. Allows all raw HTML.")
	flag.Usage = printHelp
	flag.Parse()

	if err := run(appCfg); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Operation failed: %v\n", err)
		os.Exit(1)
	}
}

func run(appCfg appConfig) error {
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return nil
	}

	opts := builder.BuildOptions{
		Unsafe: appCfg.unsafe,
		Debug:  appCfg.debug,
	}

	switch args[0] {
	case "gen":
		opts.CleanDestination = true
		fmt.Println("--- Generating site from content ---")
		siteCfg := getSiteConfig()

		tmpl, err := builder.LoadTemplates(templateDir, siteCfg.Template)
		if err != nil {
			return fmt.Errorf("failed to load templates: %w", err)
		}

		pageCount, err := builder.BuildSite(outputDir, contentDir, staticDir, siteCfg, tmpl, opts)
		if err != nil {
			return fmt.Errorf("site generation failed: %w", err)
		}
		fmt.Printf("✅ Success! Generated %d pages.\n", pageCount)
		return nil

	case "story":
		// Create a new FlagSet for the "story" command
		storyCmd := flag.NewFlagSet("story", flag.ExitOnError)
		inputFile := storyCmd.String("i", storyFile, "Input story file (*.biff).")
		outputDirFlag := storyCmd.String("o", "", "Output directory for generated content. Defaults to 'content' for 'site.biff', or 'content/<story_name>' for other input files.")
		contentOnly := storyCmd.Bool("content-only", false, "Generate content structure only, do not build site.")

		storyCmd.Usage = func() {
			fmt.Println("Usage: nibl story [options]")
			fmt.Println("\nCompile a .biff file into content pages and optionally build the site.")
			fmt.Println("\nOptions:")
			storyCmd.PrintDefaults()
		}

		storyCmd.Parse(args[1:])

		// Determine the final output directory based on flags
		finalOutputDir := *outputDirFlag
		if finalOutputDir == "" {
			if *inputFile == storyFile {
				// Default behavior: compile site.biff into the root content dir
				finalOutputDir = contentDir
			} else {
				// Behavior for custom file: compile into a subdirectory
				base := filepath.Base(*inputFile)
				name := strings.TrimSuffix(base, filepath.Ext(base))
				finalOutputDir = filepath.Join(contentDir, name)
			}
		}

		// Clean the public directory only when doing a full build
		opts.CleanDestination = !(*contentOnly)
		return handleStoryCommand(*inputFile, finalOutputDir, *contentOnly, opts)

	case "serve":
		// The build function for `serve` must do a full build using default paths.
		buildFunc := func(buildOpts builder.BuildOptions) error {
			return runFullBuild(buildOpts)
		}
		return server.Run(appCfg.port, buildFunc, opts)

	case "new":
		if len(args) < 3 {
			flag.Usage()
			return nil
		}
		if args[1] == "site" {
			return scaffold.CreateNewSite(args[2])
		}
		return scaffold.CreateNewContent(args[1], args[2], configFile)

	default:
		flag.Usage()
	}

	return nil
}

// handleStoryCommand contains the new logic for the `story` command,
// handling content-only generation and full builds.
func handleStoryCommand(inputFile, storyContentDir string, contentOnly bool, opts builder.BuildOptions) error {
	siteCfg := getSiteConfig()

	fmt.Println("--- Compiling story ---")
	knotCount, err := story.Compile(inputFile, storyContentDir, siteCfg)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("story file '%s' not found", inputFile)
		}
		return fmt.Errorf("biff compilation failed: %w", err)
	}
	fmt.Printf("📖 Story: %d knots processed into %s.\n", knotCount, storyContentDir)

	if contentOnly {
		fmt.Println("✅ Success! Content-only generation complete.")
		return nil
	}

	// If not contentOnly, proceed to build the full site.
	fmt.Println("--- Building site ---")

	tmpl, err := builder.LoadTemplates(templateDir, siteCfg.Template)
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Generate the final HTML site. It reads from the main `contentDir`
	// and builds to the main `outputDir` ("public").
	pageCount, err := builder.BuildSite(outputDir, contentDir, staticDir, siteCfg, tmpl, opts)
	if err != nil {
		return fmt.Errorf("site generation failed: %w", err)
	}
	fmt.Printf("📄 Site: %d pages generated.\n", pageCount)
	fmt.Println("✅ Build successful.")
	return nil
}

// runFullBuild encapsulates the original, default build process.
// It is used by `nibl serve` to ensure consistent behavior.
func runFullBuild(opts builder.BuildOptions) error {
	fmt.Println("--- Building site ---")
	siteCfg := getSiteConfig()

	// Step 1: Compile the story from the default `site.biff`.
	knotCount, err := story.Compile(storyFile, contentDir, siteCfg)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("🔎 No 'site.biff' found, skipping story compilation.")
		} else {
			// In serve mode, we print the error but don't stop the server.
			fmt.Fprintf(os.Stderr, "\n❌ Biff compilation failed:\n   %v\n\n", err)
			return err
		}
	} else {
		fmt.Printf("📖 Story: %d knots processed.\n", knotCount)
	}

	// Step 2: Load templates.
	tmpl, err := builder.LoadTemplates(templateDir, siteCfg.Template)
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	// Step 3: Generate the final HTML site.
	pageCount, err := builder.BuildSite(outputDir, contentDir, staticDir, siteCfg, tmpl, opts)
	if err != nil {
		return fmt.Errorf("site generation failed: %w", err)
	}
	fmt.Printf("📄 Site: %d pages generated.\n", pageCount)
	fmt.Println("✅ Build successful.")
	return nil
}

func getSiteConfig() config.SiteConfig {
	siteCfg, err := config.LoadSiteConfig(configFile)
	if err != nil {
		// Using fmt.Fprintf to stderr for critical errors that halt execution.
		fmt.Fprintf(os.Stderr, "critical error: failed to load site config: %v\n", err)
		os.Exit(1)
	}
	return siteCfg
}

func printHelp() {
	fmt.Println("nibl - a quiet static site generator for interactive fiction")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  nibl [global-flags] <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  story [options]    Compile .biff file and build site. Use 'nibl story -h' for options.")
	fmt.Println("  gen                Generate site from existing content")
	fmt.Println("  serve              Run a local dev server with auto-rebuild")
	fmt.Println("  new site <name>    Create a new site scaffold")
	fmt.Println("  new <type> <title> Create new content from archetype")
	fmt.Println()
	fmt.Println("Global Flags:")
	flag.PrintDefaults()
}

