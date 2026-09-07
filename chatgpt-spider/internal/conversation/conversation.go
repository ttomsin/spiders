package conversation

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// SendPrompt enters the prompt into ChatGPT and triggers transmission
func SendPrompt(page *rod.Page, promptText string) error {
	selectors := []string{
		"#prompt-textarea",
		"div[contenteditable='true']",
		"textarea[placeholder*='Message']",
		"textarea[data-id='root']",
		"textarea",
	}

	var inputElem *rod.Element

	for _, sel := range selectors {
		elem, findErr := page.Timeout(2 * time.Second).Element(sel)
		if findErr == nil && elem != nil {
			inputElem = elem
			break
		}
	}

	if inputElem == nil {
		return fmt.Errorf("could not find ChatGPT prompt input area. Ensure you are on the chat page and logged in")
	}

	// Focus
	_ = inputElem.Focus()

	isEditable, _ := inputElem.Attribute("contenteditable")
	if isEditable != nil && *isEditable == "true" {
		escaped := strings.ReplaceAll(promptText, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, "`", "\\`")
		escaped = strings.ReplaceAll(escaped, "$", "\\$")
		js := fmt.Sprintf(`() => {
			this.focus();
			document.execCommand('insertText', false, %q);
		}`, escaped)
		if _, jsErr := inputElem.Eval(js); jsErr != nil {
			_ = inputElem.Input(promptText)
		}
	} else {
		_ = inputElem.SelectAllText()
		_ = inputElem.Input(promptText)
	}

	time.Sleep(300 * time.Millisecond)

	// Try clicking Send button
	sendButtonSelectors := []string{
		"button[data-testid='send-button']",
		"button[aria-label='Send prompt']",
		"button[aria-label='Send message']",
	}

	for _, btnSel := range sendButtonSelectors {
		if btn, err := page.Timeout(1 * time.Second).Element(btnSel); err == nil && btn != nil {
			disabled, _ := btn.Attribute("disabled")
			if disabled == nil {
				_ = btn.Click(proto.InputMouseButtonLeft, 1)
				return nil
			}
		}
	}

	// Fallback to Enter key
	return page.KeyActions().Press(input.Enter).Do()
}

// WaitForCompletion polls the DOM to detect when response generation finishes and returns immediately
func WaitForCompletion(page *rod.Page, maxWait time.Duration) (string, error) {
	deadline := time.Now().Add(maxWait)

	// Wait briefly for streaming indicator or first tokens to appear
	time.Sleep(500 * time.Millisecond)

	var lastObservedText string
	unchangedCount := 0

	for time.Now().Before(deadline) {
		// Check if stop streaming button is visible
		stopBtn, _ := page.Timeout(200 * time.Millisecond).Element("button[data-testid='stop-button'], button[aria-label='Stop streaming'], button[aria-label='Stop generating']")

		// Find assistant response elements - always inspect the LAST assistant message on the page
		assistantMsgs, err := page.Elements("article [data-message-author-role='assistant'], div[data-message-author-role='assistant'], .markdown")
		if err == nil && len(assistantMsgs) > 0 {
			// Grab the latest assistant message
			lastElem := assistantMsgs[len(assistantMsgs)-1]
			text, _ := lastElem.Text()
			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				if stopBtn == nil {
					// Stop button gone and we have content - finished!
					return trimmed, nil
				}
				if trimmed == lastObservedText {
					unchangedCount++
					if unchangedCount >= 4 { // stable for ~1.2 seconds without stop button
						return trimmed, nil
					}
				} else {
					lastObservedText = trimmed
					unchangedCount = 0
				}
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	if lastObservedText != "" {
		return lastObservedText, nil
	}

	return "", fmt.Errorf("timeout waiting for response")
}

// NewChat triggers a fresh conversation tab
func NewChat(page *rod.Page) error {
	// Try clicking the "New chat" button in the sidebar or header
	newChatSelectors := []string{
		"a[href='/']",
		"button[aria-label='New chat']",
		"a[data-testid='navigation-item-new-chat']",
	}

	for _, sel := range newChatSelectors {
		if btn, err := page.Timeout(1 * time.Second).Element(sel); err == nil && btn != nil {
			_ = btn.Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(1 * time.Second)
			return nil
		}
	}

// Direct navigation fallback
	if err := page.Navigate("https://chatgpt.com"); err != nil {
		return err
	}
	_ = page.WaitLoad()
	time.Sleep(2 * time.Second)
	return nil
}

// OpenConversation navigates to an existing conversation thread by ID or URL
func OpenConversation(page *rod.Page, sessionIDOrURL string) error {
	raw := strings.TrimSpace(sessionIDOrURL)
	if raw == "" {
		return nil
	}

	targetURL := raw
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		// Clean ID if path is passed like "/c/xxx"
		cleanID := strings.TrimPrefix(raw, "/")
		cleanID = strings.TrimPrefix(cleanID, "c/")
		targetURL = fmt.Sprintf("https://chatgpt.com/c/%s", cleanID)
	}

	if err := page.Navigate(targetURL); err != nil {
		return fmt.Errorf("failed to navigate to conversation %s: %w", targetURL, err)
	}
	_ = page.WaitLoad()
	time.Sleep(3 * time.Second)
	return nil
}

// GetCurrentConversationID extracts the active conversation ID from the page URL
func GetCurrentConversationID(page *rod.Page) string {
	info, err := page.Info()
	if err != nil || info == nil {
		return ""
	}
	urlStr := info.URL
	if strings.Contains(urlStr, "/c/") {
		parts := strings.Split(urlStr, "/c/")
		if len(parts) >= 2 {
			id := strings.Split(parts[1], "?")[0]
			id = strings.Split(id, "#")[0]
			return strings.TrimSpace(id)
		}
	}
	return ""
}