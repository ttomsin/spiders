package format

import (
	"fmt"
	"strings"
)

// ApplyFormatConstraint appends strict response formatting rules to the prompt
func ApplyFormatConstraint(promptText string, formatSpec string) string {
	formatSpec = strings.TrimSpace(formatSpec)
	if formatSpec == "" {
		return promptText
	}

	lower := strings.ToLower(formatSpec)
	var constraint string

	switch lower {
	case "json":
		constraint = "CRITICAL CONSTRAINT: You MUST respond strictly in valid, parseable raw JSON. Do NOT include any conversational text, explanations, or markdown code fences (like ```json). Output pure JSON only."
	case "csv":
		constraint = "CRITICAL CONSTRAINT: You MUST respond strictly in raw CSV format with a valid header row. Do NOT include any conversational preamble, commentary, or markdown fences. Output pure CSV only."
	case "xml":
		constraint = "CRITICAL CONSTRAINT: You MUST respond strictly in valid, well-formed XML. Do NOT include any conversational preamble, commentary, or text outside the XML tags. Output pure XML only."
	case "yaml", "yml":
		constraint = "CRITICAL CONSTRAINT: You MUST respond strictly in valid YAML. Do NOT include any conversational commentary or text outside the YAML block. Output pure YAML only."
	case "markdown", "md":
		constraint = "CRITICAL CONSTRAINT: Format your response strictly in clean GitHub-Flavored Markdown."
	default:
		// Custom format specification (e.g. "json array of {id, sentiment, reason}")
		constraint = fmt.Sprintf("CRITICAL CONSTRAINT: You MUST strictly adhere to the following output format constraint: %s. Do NOT include conversational filler or text outside this specified format.", formatSpec)
	}

	return fmt.Sprintf("%s\n\n[%s]", promptText, constraint)
}
