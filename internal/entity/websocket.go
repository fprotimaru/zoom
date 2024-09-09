package entity

import "encoding/json"

type WebSocketMessageType uint8

const (
	ICECandidateType WebSocketMessageType = iota + 1
	Offer
	AnswerType
	OfferRequest
)

type WebSocketMessage struct {
	Type WebSocketMessageType `json:"type"`
	Data json.RawMessage      `json:"data"`
}
