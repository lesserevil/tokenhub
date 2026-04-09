package httpapi

import (
	"strings"

	"github.com/jordanhubbard/tokenhub/internal/router"
	"github.com/jordanhubbard/tokenhub/internal/store"
)

// slackMentionRequired returns true when the request appears to come from a
// Slack channel (not a DM) and the agent associated with keyRec is NOT
// @mentioned anywhere in the messages.
//
// When true, the caller should return a no-op response without forwarding the
// request to the model backend.  This prevents agents that use
// listen_all_messages=true from invoking the LLM for every Slack message they
// receive, even if they aren't being addressed.
//
// The check is intentionally conservative: if any signal is ambiguous (no API
// key, name can't be parsed, not a Slack request) it returns false so the
// request is always forwarded.
func slackMentionRequired(messages []router.Message, keyRec *store.APIKeyRecord) bool {
	if keyRec == nil || keyRec.Name == "" {
		return false
	}

	// Derive possible mention tokens from the API key name.
	// e.g. "race-bannon" → ["race", "race-bannon", "racebannon"]
	//      "drquest"     → ["drquest"]
	mentions := mentionVariants(keyRec.Name)

	isSlack := false
	isDM := false
	isMentioned := false

	for _, msg := range messages {
		content := msg.Content
		if content == "" {
			continue
		}

		if msg.Role == "system" {
			lower := strings.ToLower(content)
			// Hermes injects "**Source:** Slack (...)" in the system prompt.
			if strings.Contains(lower, "source:") && strings.Contains(lower, "slack") {
				isSlack = true
			}
			// DM indicator: "source: slack (dm " or "(dm with "
			if strings.Contains(lower, "(dm ") || strings.Contains(lower, "(dm\n") || strings.Contains(lower, "(dm)") {
				isDM = true
			}
		}

		if msg.Role == "user" {
			lower := strings.ToLower(content)
			for _, m := range mentions {
				if strings.Contains(lower, "@"+m) {
					isMentioned = true
					break
				}
			}
		}
	}

	// Only filter when we can positively identify this as a Slack channel message.
	if !isSlack || isDM {
		return false
	}

	return !isMentioned
}

// mentionVariants returns the lowercase @-handle variants for an API key name.
// We generate several forms to handle Slack display names that may differ
// from the internal key name (e.g. "race-bannon" → bot display name "Race").
func mentionVariants(name string) []string {
	name = strings.ToLower(strings.TrimSpace(name))
	seen := map[string]bool{name: true}
	result := []string{name}

	// First hyphen segment: "race-bannon" → "race"
	if idx := strings.IndexByte(name, '-'); idx > 0 {
		seg := name[:idx]
		if !seen[seg] {
			seen[seg] = true
			result = append(result, seg)
		}
	}

	// No-hyphen form: "race-bannon" → "racebannon"
	nohyphen := strings.ReplaceAll(name, "-", "")
	if !seen[nohyphen] {
		seen[nohyphen] = true
		result = append(result, nohyphen)
	}

	return result
}
