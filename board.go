package s83

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"
)

const MaxBoardLen = 2217

type Board struct {
	Publisher Publisher
	timestamp time.Time
	signature Signature
	Content   []byte
}

func (b Board) String() string {
	return string(b.Content)
}

func (b Board) Key() string {
	return b.Publisher.Key()
}

func (b Board) Info() string {
	return fmt.Sprintf("verifies  : %t\ncreator   : %s\nsignature : %s\n%s", b.VerifySignature(), b.Publisher, b.signature, b.Content)
}

func (b Board) VerifySignature() bool {
	if len(b.Publisher.PublicKey) == 0 || b.signature == nil {
		return false
	}
	return ed25519.Verify(b.Publisher.PublicKey, b.Content, b.signature)
}

func (b Board) Timestamp() string {
	// TODO (?):  If-Unmodified-Since: <date and time in UTC, HTTP (RFC 5322) format>
	return b.timestamp.Format(http.TimeFormat)
}

func (b Board) Signature() string {
	return b.signature.String()
}

func (b Board) AfterBoard(other Board) bool {
	return b.timestamp.After(other.timestamp)
}

func (b Board) After(ts time.Time) bool {
	return b.timestamp.After(ts)
}

func (b Board) Eq(other Board) bool {
	sameTime := b.timestamp == other.timestamp
	sameSig := b.Signature() == other.Signature()
	sameContent := bytes.Compare(b.Content, other.Content) == 0
	samePublisher := b.Publisher.String() == other.Publisher.String()

	return sameTime && sameSig && sameContent && samePublisher
}

func NewBoard(key string, sig Signature, content []byte) (Board, error) {
	board := Board{}
	publisher, err := NewPublisherFromKey(key)
	if err != nil {
		return Board{}, err
	}
	board.Publisher = publisher

	// validate encoding requirement
	if !utf8.Valid(content) {
		return Board{}, errors.New("Invalid Board: not UTF-8")
	}
	// validate size requirement
	if len(content) > MaxBoardLen {
		return Board{}, errors.New("Invalid Board: too large")
	}
	board.Content = content

	// validate signature (can we trust the content)
	board.signature = sig
	if !board.VerifySignature() {
		return Board{}, errors.New("Invalid Signature")
	}

	// validate "last-modified meta tag"
	ts, err := ParseTimestamp(content)
	if err != nil {
		return Board{}, err
	}
	board.timestamp = ts

	// all checks pass, good board
	return board, nil
}

func BoardFromHTTP(key string, auth string, body io.ReadCloser) (Board, error) {
	// Signature
	sig, err := parseSignatureHeader(auth)
	if err != nil {
		return Board{}, err
	}
	// Content
	content, err := io.ReadAll(body)
	if err != nil {
		return Board{}, err
	}
	return NewBoard(key, sig, content)
}
