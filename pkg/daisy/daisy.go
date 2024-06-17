package daisy

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

func (c *Client) Components(ctx context.Context) ([]string, error) {
	// Load components page
	resp, err := c.do(ctx, "GET", "components/", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("daisy: couldn't get components: %w", err)
	}

	// Parse the document
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp))
	if err != nil {
		return nil, fmt.Errorf("daisy: couldn't parse components: %w", err)
	}

	// Extract components
	var components []string
	doc.Find("a.card").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		component := strings.TrimPrefix(href, "/components/")
		components = append(components, component)
	})
	return components, nil
}

type Snippet struct {
	Component string `json:"component"`
	Title     string `json:"title"`
	Code      string `json:"code"`
}

func (b *Browser) Snippets(parent context.Context, component string) ([]Snippet, error) {
	// Join parent and browser contexts
	ctx, cancel := context.WithCancel(b.browserContext)
	defer cancel()
	go func() {
		select {
		case <-parent.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	// Rate limit to avoid abusing the website
	unlock := b.rateLimit.Lock(ctx)
	defer unlock()

	// Navigate to component page
	if err := chromedp.Run(ctx,
		chromedp.Navigate(fmt.Sprintf("https://daisyui.com/components/%s/", component)),
		chromedp.WaitReady("Preview", chromedp.BySearch),
	); err != nil {
		return nil, fmt.Errorf("daisy: couldn't navigate: %w", err)
	}

	// Wait for buttons to be ready
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("daisy: context cancelled")
	case <-time.After(500 * time.Millisecond):
	}

	// Click on all HTML buttons
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`Array.from(document.querySelectorAll('button')).filter(btn => btn.innerText === 'HTML').map(btn => btn.click());`, nil),
	); err != nil {
		return nil, fmt.Errorf("daisy: couldn't click on HTML buttons: %w", err)
	}

	// Wait for code to be ready
	if err := chromedp.Run(ctx,
		chromedp.WaitReady("code.language-html", chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("daisy: couldn't wait for code: %w", err)
	}

	// Obtain the document
	var html string
	if err := chromedp.Run(ctx,
		chromedp.OuterHTML("html", &html),
	); err != nil {
		return nil, fmt.Errorf("daisy: couldn't get html: %w", err)
	}

	// Parse the document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("daisy: couldn't parse components: %w", err)
	}

	// Extract snippets
	var snippets []Snippet
	doc.Find(".component-preview").Each(func(i int, s *goquery.Selection) {
		text := s.Find(".component-preview-title").Text()
		code := s.Find("code.language-html").Text()
		if text == "" || code == "" {
			return
		}
		snippets = append(snippets, Snippet{
			Component: component,
			Title:     text,
			Code:      code,
		})
	})
	return snippets, nil
}
