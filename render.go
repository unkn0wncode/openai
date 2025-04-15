// Package openai / render.go implements rendering of URLs as images.
package openai

import (
	"fmt"
	openaiInternal "macbot/openai/internal"

	"github.com/playwright-community/playwright-go"
)

// RenderURL renders a given URL as an image, returns bytes of the image file.
func RenderURL(url string) ([]byte, error) {
	if err := playwright.Install(); err != nil {
		return nil, fmt.Errorf("failed to install browsers: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to launch playwright: %w", err)
	}
	defer func() {
		if err := pw.Stop(); err != nil {
			openaiInternal.LogStd.Println("Error stopping Playwright:", err)
		}
	}()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			openaiInternal.LogStd.Println("Error closing browser:", err)
		}
	}()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create new page: %w", err)
	}

	if err := page.SetViewportSize(430, 932); err != nil {
		return nil, fmt.Errorf("failed to set viewport size: %w", err)
	}

	if _, err := page.Goto(url); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	screenshotBytes, err := page.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to render screenshot: %w", err)
	}

	return screenshotBytes, nil
}
