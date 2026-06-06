package clean

type Reason int

const (
	Kept Reason = iota
	DropService
	DropUnknownSender
	DropForwarded
	DropViaBot
	DropMedia
	DropEmpty
	DropShort
	NumReasons
)

type ReasonInfo struct {
	Reason Reason
	Label  string
}

var DropReasons = []ReasonInfo{
	{DropMedia, "media (no text)"},
	{DropService, "service"},
	{DropEmpty, "empty after clean"},
	{DropShort, "too short"},
	{DropForwarded, "forwarded"},
	{DropViaBot, "via_bot"},
	{DropUnknownSender, "unknown sender"},
}
