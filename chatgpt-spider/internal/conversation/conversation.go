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

// WaitForCompletion polls the DOM as a fallback to detect when response generation finishes
func WaitForCompletion(page *rod.Page, maxWait time.Duration) (string, error) {
	deadline := time.Now().Add(maxWait)

	time.Sleep(1500 * time.Millisecond)

	for time.Now().Before(deadline) {
		stopBtn, _ := page.Timeout(500 * time.Millisecond).Element("button[data-testid='stop-button'], button[aria-label='Stop streaming'], button[aria-label='Stop generating']")
		if stopBtn == nil {
			assistantMsg, err := page.Element("article [data-message-author-role='assistant'], div[data-message-author-role='assistant'], .markdown")
			if err == nil && assistantMsg != nil {
				text, _ := assistantMsg.Text()
				if text != "" {
					return text, nil
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("timeout waiting for response")
}