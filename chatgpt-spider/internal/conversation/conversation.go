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
				return btn.CancelTimeout()
			}
		}
	}
	return nil
}

// DismissModals detects and dismisses onboarding modals, temporary chat announcements, or confirmation popups
func DismissModals(page *rod.Page) {
	buttons, err := page.Timeout(300 * time.Millisecond).Elements("button")
	if err != nil || len(buttons) == 0 {
		return
	}
	for _, b := range buttons {
		txt, _ := b.Text()
		aria, _ := b.Attribute("aria-label")
		var ariaStr string
		if aria != nil {
			ariaStr = *aria
		}
		if txt == "Continue" || txt == "Got it" || txt == "OK" || ariaStr == "Close" {
			_ = b.CancelTimeout().Click(proto.InputMouseButtonLeft, 1)
			time.Sleep(200 * time.Millisecond)
			break
		}
	}
}

// OpenTemporaryChat navigates to ChatGPT and ensures the temporary chat UI toggle is enabled
func OpenTemporaryChat(page *rod.Page) error {
	info, err := page.Info()
	if err == nil && info != nil && strings.Contains(info.URL, "?temporary-chat=true") {
		if err := page.Reload(); err != nil {
			return fmt.Errorf("failed to reload temporary chat: %w", err)
		}
	} else {
		if err := page.Navigate("https://chatgpt.com/?temporary-chat=true"); err != nil {
			return fmt.Errorf("failed to navigate to temporary chat: %w", err)
		}
	}
	_ = page.WaitLoad()
	DismissModals(page)
	
	// Physically verify and click the UI toggle if the URL trick didn't work for this account
	// Give React a moment to render the header
	time.Sleep(1 * time.Second)
	
	// If the "Turn off temporary chat" button exists, it's already active.
	// If the "Temporary chat" button exists, it's inactive, so we click it.
	btn, err := page.Timeout(1 * time.Second).Element("button[aria-label='Temporary chat']")
	if err == nil && btn != nil {
		_ = btn.CancelTimeout().Click(proto.InputMouseButtonLeft, 1)
		time.Sleep(500 * time.Millisecond) // Wait for UI transition
	}
	
	return WaitUntilReady(page, 6*time.Second)
}

// WaitUntilReady dynamically waits for ChatGPT prompt textarea to become interactive without static sleeps
func WaitUntilReady(page *rod.Page, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	deadline := time.Now().Add(timeout)
	selectors := []string{
		"#prompt-textarea",
		"div[contenteditable='true']",
		"textarea[placeholder*='Message']",
	}

	for time.Now().Before(deadline) {
		DismissModals(page)
		for _, sel := range selectors {
			elem, err := page.Timeout(100 * time.Millisecond).Element(sel)
			if err == nil && elem != nil {
				if vis, _ := elem.Visible(); vis {
					return nil
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for ChatGPT prompt input area to become ready")
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
		elem, findErr := page.Timeout(1 * time.Second).Element(sel)
		if findErr == nil && elem != nil {
			inputElem = elem.CancelTimeout()
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
		time.Sleep(350 * time.Millisecond)
	}

	// 2. Now insert the user prompt text (e.g. "look into this" or analysis instruction) into the textarea
	if promptText != "" {
		_ = inputElem.Focus()
		isEditable, _ := inputElem.Attribute("contenteditable")
		if isEditable != nil && *isEditable == "true" {
			insertJS := `(text) => {
				this.focus();
				document.execCommand('selectAll', false, null);
				document.execCommand('insertText', false, text);
				this.dispatchEvent(new Event('input', { bubbles: true }));
				this.dispatchEvent(new Event('change', { bubbles: true }));
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

	// Dynamic wait for any active uploads/conversions to finish
	maxWait := 5 * time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		if rem := time.Until(deadline); rem > 0 && rem < maxWait {
			maxWait = rem
		}
	}

	timeoutCh := time.After(maxWait)
	ticker := time.NewTicker(80 * time.Millisecond)
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
				// If no file attachment, fire immediately; if attachment was present, wait 2 polls (~160ms) for stability
				targetChecks := 1
				if attachmentText != "" {
					targetChecks = 2
				}
				if consecutiveReadyChecks >= targetChecks {
					return btn.Timeout(5 * time.Second).Click(proto.InputMouseButtonLeft, 1)
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

// InjectSSEInterceptor overrides window.fetch to natively capture ChatGPT's Server-Sent Events (SSE) stream.
// This allows us to extract the pure, raw Markdown directly from the LLM, bypassing the React UI.
func InjectSSEInterceptor(page *rod.Page) {
	injectJS := `() => {
		if (window._sseInterceptorActive) return;
		window._sseInterceptorActive = true;
		window._chatgptTokens = "";
		window._chatgptStreamFinished = false;
		
		const originalFetch = window.fetch;
		window.fetch = async function(...args) {
			const url = args[0];
			const urlStr = typeof url === "string" ? url : (url && url.url ? url.url : "");
			
			// Only intercept the exact conversation endpoints to avoid resetting on /prepare or other APIs
			const isConversationEndpoint = urlStr.endsWith("/backend-api/conversation") || urlStr.endsWith("/backend-api/f/conversation");
			
			if (isConversationEndpoint) {
				// Reset stream buffers for the new generation
				window._chatgptTokens = "";
				window._chatgptStreamFinished = false;
				
				const response = await originalFetch.apply(this, args);
				const cloned = response.clone();
				
				(async () => {
					try {
						const reader = cloned.body.getReader();
						const decoder = new TextDecoder();
						while (true) {
							const { done, value } = await reader.read();
							if (done) break;
							const chunk = decoder.decode(value);
							
							// Parse the SSE chunk lines
							let lines = chunk.split("\n");
							for (let line of lines) {
								if (line.startsWith("data: ")) {
									let dataStr = line.substring(6).trim();
									if (dataStr === "[DONE]") {
										window._chatgptStreamFinished = true;
										continue;
									}
									try {
										let data = JSON.parse(dataStr);
										window._debugSSELogs = (window._debugSSELogs || "") + "\nCHUNK: " + dataStr;
										
										// Try direct format
										if (data.message && data.message.content && data.message.content.parts && data.message.content.parts.length > 0) {
											window._chatgptTokens = data.message.content.parts[0];
										}
										// Try "v" wrapper format (delta encoding)
										else if (data.v && data.v.message && data.v.message.content && data.v.message.content.parts && data.v.message.content.parts.length > 0) {
											window._chatgptTokens = data.v.message.content.parts[0];
										}
										
										// Try JSON Patch formatting (o: "append", p: "/message/content/parts/0", v: "chunk")
										if (data.o === "append" && data.p === "/message/content/parts/0" && typeof data.v === "string") {
											window._chatgptTokens += data.v;
										}
										// Try bare "v" delta chunks ({"v": "..."})
										if (typeof data.v === "string" && !data.message && !data.o && !data.p) {
											window._chatgptTokens += data.v;
										}
										// Try array of JSON Patches (o: "patch", v: [...])
										if (data.o === "patch" && Array.isArray(data.v)) {
											for (let op of data.v) {
												if (op.o === "append" && op.p === "/message/content/parts/0" && typeof op.v === "string") {
													window._chatgptTokens += op.v;
												}
											}
										}
									} catch (e) {}
								}
							}
						}
					} catch(e) {}
					window._chatgptStreamFinished = true;
				})();
				return response;
			}
			return originalFetch.apply(this, args);
		};
	}`
	_, _ = page.Eval(injectJS)
}

// CountAssistantMessages returns the number of assistant message blocks currently in the DOM
func CountAssistantMessages(page *rod.Page) int {
	return len(getAssistantElements(page))
}

// StreamCompletion polls the SSE interceptor (or DOM fallback) and invokes onDelta with each chunk
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

		// Check the SSE interceptor first
		sseRes, err := page.Eval(`() => {
			if (window._sseInterceptorActive && window._chatgptTokens && window._chatgptTokens !== "") {
				return { text: window._chatgptTokens, finished: window._chatgptStreamFinished === true };
			}
			return null;
		}`)
		
		var text string
		var sseFinished bool
		usedSSE := false
		
		if err == nil && sseRes != nil && !sseRes.Value.Nil() {
			textVal := sseRes.Value.Get("text")
			if textVal.String() != "" && textVal.String() != "<nil>" {
				text = textVal.String()
				sseFinished = sseRes.Value.Get("finished").Bool()
				usedSSE = true
				if text != "" {
					hasSeenStreaming = true
				}
			}
		} else {
			// Fallback to DOM polling if SSE isn't active
			assistantMsgs := getAssistantElements(page)
			if len(assistantMsgs) > initialCount {
				lastElem := assistantMsgs[len(assistantMsgs)-1]
				if md, err := lastElem.Element(".markdown, div[class*='whitespace-pre-wrap']"); err == nil && md != nil {
					text, _ = md.Text()
				} else {
					text, _ = lastElem.Text()
				}
			}
		}

		if text != "" && len(text) > len(lastObservedText) {
			// Debug log to console
			fmt.Printf("[DEBUG] Extracted text (SSE=%v): %s\n", usedSSE, text[len(lastObservedText):])
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
					// Fast path termination if SSE says [DONE]
					if usedSSE && sseFinished {
						if dbgRes, _ := page.Eval(`() => window._debugSSELogs`); dbgRes != nil { fmt.Printf("\n[DEBUG SSE RAW LOGS]\n%s\n[END DEBUG SSE RAW LOGS]\n", dbgRes.Value.String()) }
						return strings.TrimSpace(lastObservedText), nil
					}
					// If send button has reappeared and text has stabilized, generation is finished
					if sendBtn != nil && unchangedCount >= 2 {
						if dbgRes, _ := page.Eval(`() => window._debugSSELogs`); dbgRes != nil { fmt.Printf("\n[DEBUG SSE RAW LOGS]\n%s\n[END DEBUG SSE RAW LOGS]\n", dbgRes.Value.String()) }
						return strings.TrimSpace(lastObservedText), nil
					}
					// If we observed streaming and the stop button is no longer visible, finish
					if hasSeenStreaming && !isGenerating && unchangedCount >= 2 {
						if dbgRes, _ := page.Eval(`() => window._debugSSELogs`); dbgRes != nil { fmt.Printf("\n[DEBUG SSE RAW LOGS]\n%s\n[END DEBUG SSE RAW LOGS]\n", dbgRes.Value.String()) }
						return strings.TrimSpace(lastObservedText), nil
					}
					// DOM-only fallback: if not generating and no SSE active, wait for text to stabilize
					// Don't use this when SSE is active — ChatGPT can pause mid-stream for seconds
					if !usedSSE && !isGenerating && unchangedCount >= 6 {
						if dbgRes, _ := page.Eval(`() => window._debugSSELogs`); dbgRes != nil { fmt.Printf("\n[DEBUG SSE RAW LOGS]\n%s\n[END DEBUG SSE RAW LOGS]\n", dbgRes.Value.String()) }
						return strings.TrimSpace(lastObservedText), nil
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

		time.Sleep(5 * time.Millisecond)
	}

	if lastObservedText != "" {
		if dbgRes, _ := page.Eval(`() => window._debugSSELogs`); dbgRes != nil {
			fmt.Printf("\n[DEBUG SSE RAW LOGS]\n%s\n[END DEBUG SSE RAW LOGS]\n", dbgRes.Value.String())
		}
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
	info, _ := page.Info()
	// If already on the base chat page without an active conversation, just ensure ready
	if info != nil && !strings.Contains(info.URL, "/c/") && CountAssistantMessages(page) == 0 {
		return WaitUntilReady(page, 5*time.Second)
	}

	newChatSelectors := []string{
		"a[data-testid='navigation-item-new-chat']",
		"button[aria-label='New chat']",
		"a[href='/']",
	}

	for _, sel := range newChatSelectors {
		if btn, err := page.Timeout(1 * time.Second).Element(sel); err == nil && btn != nil {
			_ = btn.CancelTimeout().Click(proto.InputMouseButtonLeft, 1)
			_ = WaitUntilReady(page, 5*time.Second)
			time.Sleep(300 * time.Millisecond)
			return nil
		}
	}

	// Direct navigation fallback
	if err := page.Navigate("https://chatgpt.com"); err != nil {
		return err
	}
	_ = page.WaitLoad()
	return WaitUntilReady(page, 5*time.Second)
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
	_ = WaitUntilReady(page, 6*time.Second)
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
			let authHeader = {};
			try {
				const sRes = await fetch('/api/auth/session');
				if (sRes.ok) {
					const sData = await sRes.json();
					if (sData && sData.accessToken) {
						authHeader['Authorization'] = 'Bearer ' + sData.accessToken;
					}
				}
			} catch (e) {}

			const headers = { 'Content-Type': 'application/json', ...authHeader };

			// First attempt: Call ChatGPT's internal backend API directly from page context
			const res = await fetch('/backend-api/conversation/' + id, {
				method: 'PATCH',
				headers: headers,
				body: JSON.stringify({ is_visible: false })
			});
			if (res.ok) return { success: true };
			
			// Fallback attempt: Standard DELETE method
			const resDel = await fetch('/backend-api/conversation/' + id, {
				method: 'DELETE',
				headers: headers
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