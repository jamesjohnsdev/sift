package tui

// Placeholder in-memory data until the provider/storage layers exist.

type tag struct {
	name string
}

type message struct {
	from    string
	subject string
	date    string
	body    string
}

func placeholderTags() []tag {
	return []tag{
		{name: "Inbox"},
		{name: "Sent"},
		{name: "Drafts"},
		{name: "Work/Projects"},
		{name: "Trash"},
	}
}

func placeholderMessages() map[string][]message {
	return map[string][]message{
		"Inbox": {
			{
				from:    "alice@example.com",
				subject: "Q3 roadmap review",
				date:    "09:14",
				body:    "Hey,\n\nCan we push the roadmap review to Thursday? Still waiting on numbers from finance.\n\n- Alice",
			},
			{
				from:    "github@notifications.example.com",
				subject: "[sift] New issue: sync stalls on large mailboxes",
				date:    "08:02",
				body:    "A new issue was opened:\n\nSync appears to stall past ~200k messages during initial backfill.",
			},
			{
				from:    "bob@example.com",
				subject: "Re: Outlook OAuth scopes",
				date:    "Yesterday",
				body:    "Confirmed - Mail.ReadWrite and Mail.Send cover what we need for v1.",
			},
		},
		"Sent": {
			{
				from:    "me@example.com",
				subject: "Re: Q3 roadmap review",
				date:    "Yesterday",
				body:    "Sounds good, Thursday works.",
			},
		},
		"Drafts": {
			{
				from:    "me@example.com",
				subject: "Draft: vacation notice",
				date:    "Mon",
				body:    "I'll be out from ...",
			},
		},
		"Work/Projects": {
			{
				from:    "carol@example.com",
				subject: "Plugin API surface proposal",
				date:    "Mon",
				body:    "Attaching a first pass at the Lua hook surface for review.",
			},
		},
		"Trash": {},
	}
}
