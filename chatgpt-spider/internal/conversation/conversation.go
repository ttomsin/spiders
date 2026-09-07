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

// StreamCompletion polls the DOM and invokes onDelta with each newly rendered token chunk
func StreamCompletion(page *rod.Page, maxWait time.Duration, onDelta func(delta string)) (string, error) {
	deadline := time.Now().Add(maxWait)

	var lastObservedText string
	unchangedCount := 0
	hasSeenStreaming := false

	// Wait up to 5 seconds for streaming indicator or first tokens
	initDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(initDeadline) {
		stopBtn, _ := page.Timeout(200 * time.Millisecond).Element("button[data-testid='stop-button'], button[aria-label='Stop streaming'], button[aria-label='Stop generating']")
		if stopBtn != nil {
			hasSeenStreaming = true
			break
		}
		time.Sleep(150 * time.Millisecond)
	}

	for time.Now().Before(deadline) {
		stopBtn, _ := page.Timeout(150 * time.Millisecond).Element("button[data-testid='stop-button'], button[aria-label='Stop streaming'], button[aria-label='Stop generating']")
		if stopBtn != nil {
			hasSeenStreaming = true
		}

		assistantMsgs, err := page.Elements("article [data-message-author-role='assistant'], div[data-message-author-role='assistant'], .markdown")
		if err == nil && len(assistantMsgs) > 0 {
			lastElem := assistantMsgs[len(assistantMsgs)-1]
			text, _ := lastElem.Text()
			if text != "" {
				if len(text) > len(lastObservedText) {
					delta := text[len(lastObservedText):]
					if onDelta != nil && delta != "" {
						onDelta(delta)
					}
					lastObservedText = text
					unchangedCount = 0
				} else if text == lastObservedText {
					unchangedCount++
					if (hasSeenStreaming && stopBtn == nil && unchangedCount >= 3) || unchangedCount >= 6 {
						return strings.TrimSpace(lastObservedText), nil
					}
				}
			}
		}

		time.Sleep(120 * time.Millisecond)
	}

	if lastObservedText != "" {
		return strings.TrimSpace(lastObservedText), nil
	}

	return "", fmt.Errorf("timeout waiting for response")
}

// WaitForCompletion polls the DOM to detect when response generation finishes and returns immediately
func WaitForCompletion(page *rod.Page, maxWait time.Duration) (string, error) {
	return StreamCompletion(page, maxWait, nil)
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

// ConversationTurn represents a single user or assistant exchange in the conversation
type ConversationTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GetConversationHistory extracts the visible message turns of the currently loaded conversation
func GetConversationHistory(page *rod.Page) ([]ConversationTurn, error) {
	var turns []ConversationTurn

	articles, err := page.Elements("article")
	if err != nil || len(articles) == 0 {
		return turns, nil
	}

	for _, art := range articles {
		role := "user"
		roleAttr, _ := art.Attribute("data-message-author-role")
		if roleAttr != nil && *roleAttr != "" {
			role = *roleAttr
		} else {
			if asst, _ := art.Element("[data-message-author-role='assistant']"); asst != nil {
				role = "assistant"
			}
		}

		text, _ := art.Text()
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			turns = append(turns, ConversationTurn{
				Role:    role,
				Content: trimmed,
			})
		}
	}

	return turns, nil
}