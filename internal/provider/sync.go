package provider

// Cursor is an opaque, provider-defined pagination/delta token; callers
// persist and replay it verbatim, never parse it.
type Cursor string

type Page struct {
	Messages []Message
	Next     Cursor
	HasMore  bool
}

type UpdateKind int

const (
	MessageAdded UpdateKind = iota
	MessageChanged
	MessageRemoved
)

type Update struct {
	Kind      UpdateKind
	MessageID MessageID
}
