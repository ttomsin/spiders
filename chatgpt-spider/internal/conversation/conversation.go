package conversation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// isAttachmentUploading checks if an attachment or pasted text card is actively loading/uploading
func isAttachmentUploading(page *rod.Page) bool {
	spinnerSelectors := []string{
		"[role='progressbar']",
		"[class*='animate-spin']",
		"[class*='loading']",
		"circle[class*='progress']",
		"div[class*='radial-progress']",
		"[data-testid*='upload-progress']",
		"div[class*='attachment'] svg[class*='animate-spin']",
	}

	for _, sel := range spinnerSelectors {
		elems, err := page.Elements(sel)
		if err == nil && len(elems) > 0 {
			for _, elem := range elems {
				if visible, _ := elem.Visible(); visible {
					return true
				}
			}
		}
	}
	return false
}

// checkAttachmentError checks if any attachment in the composer encountered an error
func checkAttachmentError(page *rod.Page) string {
	errSelectors := []string{
		"[data-testid*='upload-error']",
		"div[class*='text-red'][role='alert']",
		"div[class*='text-token-text-error']",
	}

	for _, sel := range errSelectors {
		elem, err := page.Timeout(50 * time.Millisecond).Element(sel)
		if err == nil && elem != nil {
			if visible, _ := elem.Visible(); visible {
				text, _ := elem.Text()
				if text != "" {
					return strings.TrimSpace(text)
				}
				return "attachment upload failed"
			}
		}
	}
	return ""
}

// getEnabledSendButton finds the Send button if it is currently enabled and clickable
func getEnabledSendButton(page *rod.Page) *rod.Element {
	sendButtonSelectors := []string{
		"button[data-testid='send-button']",
		"button[aria-label='Send prompt']",
		"button[aria-label='Send message']",
	}

	for _, btnSel := range sendButtonSelectors {
		btn, err := page.Timeout(100 * time.Millisecond).Element(btnSel)
		if err == nil && btn != nil {
			disabled, _ := btn.Attribute("disabled")
			ariaDisabled, _ := btn.Attribute("aria-disabled")
			if disabled == nil && (ariaDisabled == nil || *ariaDisabled != "true") {
				return btn
			}
		}
	}
	return nil
}

// SendPrompt enters the prompt (and optional attachment text) into ChatGPT and triggers transmission
func SendPrompt(ctx context.Context, page *rod.Page, promptText string, attachmentText string) error {
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

	// 1. If an attachment is provided, paste it first so ChatGPT creates the document pill card
	if attachmentText != "" {
		pasteJS := `(text) => {
			this.focus();
			const dt = new DataTransfer();
			dt.setData('text/plain', text);
			const pasteEvt = new ClipboardEvent('paste', {
				bubbles: true,
				cancelable: true,
				clipboardData: dt
			});
			this.dispatchEvent(pasteEvt);
		}`
		_, _ = inputElem.Eval(pasteJS, attachmentText)
		// Give ChatGPT time to convert to document pill card
		time.Sleep(600 * time.Millisecond)
	}

	// 2. Now insert the user prompt text (e.g. "look into this" or analysis instruction) into the textarea
	if promptText != "" {
		_ = inputElem.Focus()
		isEditable, _ := inputElem.Attribute("contenteditable")
		if isEditable != nil && *isEditable == "true" {
			insertJS := `(text) => {
				this.focus();
				document.execCommand('insertText', false, text);
			}`
			if _, jsErr := inputElem.Eval(insertJS, promptText); jsErr != nil {
				_ = inputElem.Input(promptText)
			}
		} else {
			_ = inputElem.Input(promptText)
		}
	} else if attachmentText == "" {
		return fmt.Errorf("empty prompt")
	}

	// Give ChatGPT event handlers time to capture input and update UI state
	time.Sleep(400 * time.Millisecond)

	// Dynamically wait for any active uploads/conversions to finish
	// We monitor actual progress indicators rather than relying on a small static sleep!
	maxWait := 5 * time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		if rem := time.Until(deadline); rem > 0 && rem < maxWait {
			maxWait = rem
		}
	}

	timeoutCh := time.After(maxWait)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	consecutiveReadyChecks := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutCh:
			return fmt.Errorf("timed out waiting for attachment / pasted text upload to complete (max wait %v exceeded)", maxWait)
		case <-ticker.C:
			// 1. Check for upload errors
			if errMsg := checkAttachmentError(page); errMsg != "" {
				return fmt.Errorf("pasted document upload failed: %s", errMsg)
			}

			// 2. Check if attachment is actively uploading / processing
			if isAttachmentUploading(page) {
				consecutiveReadyChecks = 0
				continue
			}

			// 3. If no active upload, check if Send button is enabled
			btn := getEnabledSendButton(page)
			if btn != nil {
				consecutiveReadyChecks++
				// Require 2 consecutive checks (~400ms) to ensure DOM has stabilized
				if consecutiveReadyChecks >= 2 {
					return btn.Click(proto.InputMouseButtonLeft, 1)
				}
			} else {
				consecutiveReadyChecks = 0
			}
		}
	}
}

// getAssistantElements returns all assistant message articles currently in the DOM
func getAssistantElements(page *rod.Page) []*rod.Element {
	articles, err := page.Elements("article")
	if err == nil && len(articles) > 0 {
		var assistants []*rod.Element
		for _, art := range articles {
			roleAttr, _ := art.Attribute("data-message-author-role")
			if roleAttr != nil && *roleAttr == "assistant" {
				assistants = append(assistants, art)
				continue
			}
			if asst, _ := art.Element("[data-message-author-role='assistant']"); asst != nil {
				assistants = append(assistants, art)
			}
		}
		if len(assistants) > 0 {
			return assistants
		}
	}

	// Fallback to direct attribute selector
	elems, err := page.Elements("[data-message-author-role='assistant']")
	if err == nil && len(elems) > 0 {
		return elems
	}

	return nil
}

// CountAssistantMessages returns the number of assistant message blocks currently in the DOM
func CountAssistantMessages(page *rod.Page) int {
	return len(getAssistantElements(page))
}

// StreamCompletion polls the DOM and invokes onDelta with each newly rendered token chunk
func StreamCompletion(ctx context.Context, page *rod.Page, initialCount int, maxWait time.Duration, onDelta func(delta string)) (string, error) {
	if maxWait <= 0 {
		maxWait = 5 * time.Minute
	}
	deadline := time.Now().Add(maxWait)

	var lastObservedText string
	unchangedCount := 0
	hasSeenStreaming := false

	// Give ChatGPT up to 15s to begin generating
	startDeadline := time.Now().Add(15 * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		var isGenerating bool
		stopBtn, _ := page.Timeout(50 * time.Millisecond).Element("button[data-testid='stop-button'], button[aria-label='Stop streaming'], button[aria-label='Stop generating']")
		if stopBtn != nil {
			if vis, _ := stopBtn.Visible(); vis {
				isGenerating = true
				hasSeenStreaming = true
			}
		}

		// Also check if Send button has reappeared (meaning generation is definitely done)
		sendBtn := getEnabledSendButton(page)

		assistantMsgs := getAssistantElements(page)
		if len(assistantMsgs) > initialCount {
			lastElem := assistantMsgs[len(assistantMsgs)-1]

			var text string
			if md, err := lastElem.Element(".markdown, div[class*='whitespace-pre-wrap']"); err == nil && md != nil {
				text, _ = md.Text()
			} else {
				text, _ = lastElem.Text()
			}

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
					// If send button has reappeared and text has stabilized, generation is finished
					if sendBtn != nil && unchangedCount >= 2 {
						return strings.TrimSpace(lastObservedText), nil
					}
					// If we observed streaming and the stop button is no longer visible, finish
					if hasSeenStreaming && !isGenerating && unchangedCount >= 2 {
						return strings.TrimSpace(lastObservedText), nil
					}
					// Fallback: if not generating and text has stabilized for 6 polls (~720ms)
					if !isGenerating && unchangedCount >= 6 {
						return strings.TrimSpace(lastObservedText), nil
					}
				}
			}
		} else if time.Now().After(startDeadline) && !hasSeenStreaming {
			// In case the selector differed, check fallback
			fallbacks, _ := page.Elements(".markdown")
			if len(fallbacks) > initialCount {
				lastElem := fallbacks[len(fallbacks)-1]
				text, _ := lastElem.Text()
				if text != "" {
					return strings.TrimSpace(text), nil
				}
			}
			return "", fmt.Errorf("timeout waiting for assistant to begin generating")
		}

		time.Sleep(120 * time.Millisecond)
	}

	if lastObservedText != "" {
		return strings.TrimSpace(lastObservedText), nil
	}

	return "", fmt.Errorf("timeout waiting for response")
}

// WaitForCompletion polls the DOM to detect when response generation finishes and returns immediately
func WaitForCompletion(ctx context.Context, page *rod.Page, initialCount int, maxWait time.Duration) (string, error) {
	return StreamCompletion(ctx, page, initialCount, maxWait, nil)
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

// DeleteConversation deletes a conversation by its UUID from ChatGPT using the authenticated session
func DeleteConversation(page *rod.Page, convID string) error {
	cleanID := strings.TrimSpace(convID)
	cleanID = strings.TrimPrefix(cleanID, "/")
	cleanID = strings.TrimPrefix(cleanID, "c/")
	if cleanID == "" {
		return fmt.Errorf("empty conversation ID")
	}

	deleteJS := `async (id) => {
		try {
			// First attempt: Call ChatGPT's internal backend API directly from page context
			const res = await fetch('/backend-api/conversation/' + id, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ is_visible: false })
			});
			if (res.ok) return { success: true };
			
			// Fallback attempt: Standard DELETE method
			const resDel = await fetch('/backend-api/conversation/' + id, {
				method: 'DELETE'
			});
			if (resDel.ok) return { success: true };

			return { success: false, status: res.status, statusDel: resDel.status };
		} catch (err) {
			return { success: false, error: err.message };
		}
	}`

	res, err := page.Eval(deleteJS, cleanID)
	if err != nil {
		return fmt.Errorf("failed to execute conversation delete: %w", err)
	}

	// Also navigate back to clean home page or trigger NewChat so UI resets
	_ = NewChat(page)

	if res != nil && res.Value.Get("success").Bool() {
		return nil
	}
	return nil
}

// ConversationTurn represents a single user or assistant exchange in the conversation
type ConversationTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GetConversationHistory extracts the visible message turns of the currently loaded conversation
func GetConversationHistory(page *rod.Page) ([]ConversationTurn, error) {
	var turns []ConversationTurn

	// Wait up to 5 seconds for messages to render after navigation
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		arts, _ := page.Elements("article, [data-message-author-role]")
		if len(arts) > 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	articles, err := page.Elements("article")
	if err == nil && len(articles) > 0 {
		for _, art := range articles {
			role := "user"
			roleAttr, _ := art.Attribute("data-message-author-role")
			if roleAttr != nil && *roleAttr != "" {
				role = *roleAttr
			} else if asst, _ := art.Element("[data-message-author-role='assistant']"); asst != nil {
				role = "assistant"
			}

			// Try getting markdown / text container first
			msgBody, _ := art.Element(".markdown, [data-message-author-role], div[class*='whitespace-pre-wrap']")
			var text string
			if msgBody != nil {
				text, _ = msgBody.Text()
			} else {
				text, _ = art.Text()
			}

			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				turns = append(turns, ConversationTurn{
					Role:    role,
					Content: trimmed,
				})
			}
		}
	}

	// Fallback to data-message-author-role elements directly if article is not used in DOM
	if len(turns) == 0 {
		elems, err := page.Elements("[data-message-author-role]")
		if err == nil && len(elems) > 0 {
			for _, elem := range elems {
				role := "user"
				if roleAttr, _ := elem.Attribute("data-message-author-role"); roleAttr != nil && *roleAttr != "" {
					role = *roleAttr
				}
				text, _ := elem.Text()
				trimmed := strings.TrimSpace(text)
				if trimmed != "" {
					turns = append(turns, ConversationTurn{
						Role:    role,
						Content: trimmed,
					})
				}
			}
		}
	}

	return turns, nil
}