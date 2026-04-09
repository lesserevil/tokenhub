package httpapi

import (
	"testing"

	"github.com/jordanhubbard/tokenhub/internal/router"
	"github.com/jordanhubbard/tokenhub/internal/store"
)

func TestSlackMentionRequired(t *testing.T) {
	key := func(name string) *store.APIKeyRecord { return &store.APIKeyRecord{Name: name} }

	slackGroupSys := router.Message{
		Role:    "system",
		Content: "## Current Session Context\n**Source:** Slack (group: proj-channel)\n**Platform notes:** You are running inside Slack.",
	}
	slackDMSys := router.Message{
		Role:    "system",
		Content: "## Current Session Context\n**Source:** Slack (DM with Shawn)\n**Platform notes:** You are running inside Slack.",
	}
	localSys := router.Message{
		Role:    "system",
		Content: "## Current Session Context\n**Source:** Local (the machine running this agent)\n",
	}

	mention := func(name string) router.Message {
		return router.Message{Role: "user", Content: "@" + name + " can you look at this?"}
	}
	noMention := router.Message{Role: "user", Content: "can anyone look at this?"}

	tests := []struct {
		name     string
		messages []router.Message
		keyName  string
		want     bool // true = filter should block (no mention in channel)
	}{
		{
			name:     "channel message with direct mention",
			messages: []router.Message{slackGroupSys, mention("drquest")},
			keyName:  "drquest",
			want:     false, // should forward
		},
		{
			name:     "channel message with no mention",
			messages: []router.Message{slackGroupSys, noMention},
			keyName:  "drquest",
			want:     true, // should block
		},
		{
			name:     "DM message with no mention",
			messages: []router.Message{slackDMSys, noMention},
			keyName:  "drquest",
			want:     false, // DMs always forward
		},
		{
			name:     "local (non-Slack) message",
			messages: []router.Message{localSys, noMention},
			keyName:  "drquest",
			want:     false, // not Slack → forward
		},
		{
			name:     "race-bannon key, mention via short name",
			messages: []router.Message{slackGroupSys, mention("race")},
			keyName:  "race-bannon",
			want:     false, // "race" matches first segment of "race-bannon"
		},
		{
			name:     "race-bannon key, no mention at all",
			messages: []router.Message{slackGroupSys, noMention},
			keyName:  "race-bannon",
			want:     true, // should block
		},
		{
			name:     "nil key record",
			messages: []router.Message{slackGroupSys, noMention},
			keyName:  "",
			want:     false, // unknown key → allow through
		},
		{
			name:     "mention of different agent in channel",
			messages: []router.Message{slackGroupSys, mention("jonny")},
			keyName:  "drquest",
			want:     true, // jonny was mentioned, not drquest
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var rec *store.APIKeyRecord
			if tc.keyName != "" {
				rec = key(tc.keyName)
			}
			got := slackMentionRequired(tc.messages, rec)
			if got != tc.want {
				t.Errorf("slackMentionRequired() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMentionVariants(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{"drquest", []string{"drquest"}},
		{"race-bannon", []string{"race-bannon", "race", "racebannon"}},
		{"jonny", []string{"jonny"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mentionVariants(tc.name)
			if len(got) != len(tc.want) {
				t.Fatalf("mentionVariants(%q) = %v, want %v", tc.name, got, tc.want)
			}
			for i, v := range got {
				if v != tc.want[i] {
					t.Errorf("variant[%d] = %q, want %q", i, v, tc.want[i])
				}
			}
		})
	}
}
