package s83

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func (b Board) SameAs(other Board) bool {
	sameTime := b.timestamp == other.timestamp
	sameSig := b.Signature() == other.Signature()
	sameContent := bytes.Compare(b.Content, other.Content) == 0
	samePublisher := b.Publisher.String() == other.Publisher.String()

	return sameTime && sameSig && sameContent && samePublisher
}

// TODO: this clobbers old files, which matches the ephemeral nature of the protocol
// but may want to provide an option for keeping around old boards.
func (b Board) Save(dir string) error {
	path := filepath.Join(dir, fmt.Sprintf("%s.s83", b.Publisher.String()))
	data := append([]byte(b.Signature()+"\n"), b.Content...)
	// TODO: move actual write out to client, maybe move the whole thing out.
	// the goal was to make the client and server data stores similar.
	return os.WriteFile(path, data, 0600)
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

func BoardFromPath(path string) (Board, error) {

	// TODO: validate path is in store
	data, err := os.ReadFile(path)
	if err != nil {
		return Board{}, err
	}
	line := bytes.Index(data, []byte("\n"))

	// first line stores the signature
	sig, err := hex.DecodeString(string(data[:line]))
	if err != nil {
		return Board{}, err
	}
	// everything else is content
	content := data[line+1:]

	// validate on creation
	return NewBoard(keyFromPath(path), sig, content)
}

func keyFromPath(path string) string {
	// extract publisher key from file name
	return strings.TrimSuffix(filepath.Base(path), ".s83")
}
