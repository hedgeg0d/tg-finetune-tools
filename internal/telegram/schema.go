package telegram

import "encoding/json"

type Export struct {
	Name string `json:"name"`
	Type string `json:"type"`
	ID   int64  `json:"id"`
}

type Entity struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Href string `json:"href"`
}

type RawMessage struct {
	ID            int64           `json:"id"`
	Type          string          `json:"type"`
	Date          string          `json:"date"`
	DateUnixtime  string          `json:"date_unixtime"`
	From          string          `json:"from"`
	FromID        string          `json:"from_id"`
	Text          json.RawMessage `json:"text"`
	TextEntities  []Entity        `json:"text_entities"`
	MediaType     string          `json:"media_type"`
	StickerEmoji  string          `json:"sticker_emoji"`
	ForwardedFrom *string         `json:"forwarded_from"`
	ViaBot        string          `json:"via_bot"`
	Action        string          `json:"action"`
	ReplyToMsgID  int64           `json:"reply_to_message_id"`
}
