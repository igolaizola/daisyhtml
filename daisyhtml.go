package daisyhtml

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/igolaizola/daisyhtml/pkg/daisy"
)

// Generate generates HTML files for each snippet of each daisyUI component
func Generate(ctx context.Context, output string) error {
	log.Println("running")
	defer log.Println("finished")

	// Create daisy client
	client := daisy.New(&daisy.Config{
		Wait:  1 * time.Second,
		Debug: false,
	})

	// Get components
	components, err := client.Components(ctx)
	if err != nil {
		return err
	}
	if len(components) == 0 {
		return fmt.Errorf("no components found")
	}

	// Create browser
	browser := daisy.NewBrowser(&daisy.BrowserConfig{
		Wait:     1 * time.Second,
		Headless: true,
	})
	if err := browser.Start(ctx); err != nil {
		return err
	}
	defer func() { _ = browser.Stop() }()

	// Get snippets for each component
	for _, component := range components {
		snippets, err := browser.Snippets(ctx, component)
		if err != nil {
			return err
		}
		log.Printf("component %q (%d files)\n", component, len(snippets))
		if len(snippets) == 0 {
			continue
		}

		// Create snippets folder
		htmlFolder := filepath.Join(output, "html", component)
		jsxFolder := filepath.Join(output, "jsx", component)
		if err := os.MkdirAll(htmlFolder, 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(jsxFolder, 0755); err != nil {
			return err
		}

		// Write snippets to files
		for _, snippet := range snippets {
			name := strings.ToLower(snippet.Title)
			name = strings.ReplaceAll(name, " ", "-")
			name = strings.ReplaceAll(name, "/", "-")
			htmlPath := filepath.Join(htmlFolder, fmt.Sprintf("%s.html", name))
			if err := os.WriteFile(htmlPath, []byte(snippet.HTML), 0644); err != nil {
				return err
			}
			jsxPath := filepath.Join(jsxFolder, fmt.Sprintf("%s.jsx", name))
			if err := os.WriteFile(jsxPath, []byte(snippet.JSX), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
