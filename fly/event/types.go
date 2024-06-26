package event

import (
	"context"
	"time"
)

type DuplicateVaa struct {
	VaaID            string     `json:"vaaId"`
	ChainID          uint16     `json:"chainId"`
	Version          uint8      `json:"version"`
	GuardianSetIndex uint32     `json:"guardianSetIndex"`
	Vaa              []byte     `json:"vaas"`
	Digest           string     `json:"digest"`
	ConsistencyLevel uint8      `json:"consistencyLevel"`
	Timestamp        *time.Time `json:"timestamp"`
}

type event struct {
	TrackID string `json:"trackId"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	Data    any    `json:"data"`
}

type EventDispatcher interface {
	NewDuplicateVaa(ctx context.Context, e DuplicateVaa) error
}
