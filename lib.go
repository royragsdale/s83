package s83

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// SpringVersion for use in http headers
const SpringVersion = "83"

const keyLen = 64
const sigLen = 128

const maxNumBoards = 10_000_000

const maxBoardLen = 2217

const yearBase = 2000 // update in year 3000

const TestPublic = "ab589f4dde9fce4180fcf42c7b05185b0a02a5d682e353fa39177995083e0583"
const TestPrivate = "3371f8b011f51632fea33ed0a3688c26a45498205c6097c352bd4d079d224419"

const TimeFormat8601 = "2006-01-02T15:04:05Z" //"YYYY-MM-DDTHH:MM:SSZ"

type Creator struct {
	PrivateKey ed25519.PrivateKey
	Publisher
}

func genCreator() (Creator, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	return Creator{priv, Publisher{pub}}, err
}

func NewCreator() (Creator, int, error) {
	var err error
	cnt := 0
	c := Creator{}
	for !c.Publisher.valid() {
		c, err = genCreator()
		if err != nil {
			return c, cnt, err
		}
		cnt += 1
	}
	return c, cnt, nil
}

func NewCreatorFromKey(privateKeyHex string) (Creator, error) {
	if len(privateKeyHex) != keyLen {
		return Creator{}, errors.New("Invalid key length")
	}
	// "crypto/ed2551's private key representation includes a public key suffix...
	// refers to the RFC 8032 private key as the “seed”"
	seed, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return Creator{}, err
	}
	privateKey := ed25519.NewKeyFromSeed(seed)

	publicKey := privateKey.Public().(ed25519.PublicKey)

	return Creator{privateKey, Publisher{publicKey}}, nil
}

func (c Creator) String() string {
	return c.Publisher.String()
}

func (c Creator) ExportPrivateKey() string {
	// crypto/ed25519 keys include a public key suffix (strip for consistency with spec)
	return hex.EncodeToString(c.PrivateKey)[:64]
}

func (c Creator) Valid() bool {
	return c.Publisher.valid()
}

func timeElem(t time.Time) string {
	tStr := t.UTC().Format(TimeFormat8601)
	return fmt.Sprintf(`<time datetime="%s">`, tStr)
}

func (c Creator) NewBoard(content []byte) (Board, error) {

	// check board doesn't already have a timestamp
	ts, err := ParseTimestamp(content)
	if err != nil {
		// TODO: consider other error cases (e.g. unparsable/multiple)
		// no good timestamp, so helpfully prepend one
		tElem := []byte(timeElem(time.Now().UTC()))
		content = append(tElem, content...)
	} else if ts.After(time.Now().UTC()) {
		// check the timestamp provided is not in the future
		return Board{}, errors.New("time element timestamp is in the future")
	}

	// timestamp is good.

	// create signature
	sig := ed25519.Sign(c.PrivateKey, content)
	return NewBoard(c.Publisher.String(), sig, content)
}

type Publisher struct {
	PublicKey ed25519.PublicKey
}

func NewPublisherFromKey(publicKeyHex string) (Publisher, error) {
	if len(publicKeyHex) != keyLen {
		return Publisher{}, errors.New("Invalid key length")
	}
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return Publisher{}, err
	}
	return Publisher{publicKey}, nil
}

func (p Publisher) String() string {
	return hex.EncodeToString(p.PublicKey)
}

// TODO: add difficulty check
func (p Publisher) valid() bool {
	// ensures a key conforms to the correct format
	// final seven hex characters must be 83e followed by four characters, interpreted as MMYY
	reValidKey := regexp.MustCompile(`83e(0[1-9]|1[0-2])(\d\d)$`)
	if !reValidKey.MatchString(p.String()) {
		return false
	}

	// the key is only valid in the two years preceding it,
	// and expires at the end of the last day of the month specified
	yearStr := p.String()[keyLen-2:]
	keyYear, err := strconv.Atoi(yearStr)
	if err != nil {
		return false
	}

	monthStr := p.String()[keyLen-4 : keyLen-2]
	keyMonth, err := strconv.Atoi(monthStr)
	if err != nil {
		return false
	}

	keyDate := time.Date(yearBase+keyYear, time.Month(keyMonth), 0, 0, 0, 0, 0, time.UTC)
	keyExpiry := keyDate.AddDate(0, 1, 0) // valid for the entire month of expiration
	keyStart := keyDate.AddDate(-2, 0, 0) // valid for two years preceding
	now := time.Now().UTC()

	return keyStart.Before(now) && keyExpiry.After(now)
}

type Signature []byte

func (s Signature) String() string {
	return hex.EncodeToString(s)
}

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

func (b Board) After(other Board) bool {
	return b.timestamp.After(other.timestamp)
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
	if len(content) > maxBoardLen {
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

func NewBoardFromHTTP(key string, auth string, body io.ReadCloser) (Board, error) {
	// Authorization
	sig, err := parseAuthorizationHeader(auth)
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

func parseAuthorizationHeader(auth string) (Signature, error) {
	//Authorization: Spring-83 Signature=<signature>
	reSig := regexp.MustCompile(`^Spring-83 Signature=([0-9A-Fa-f]{128}?)$`)
	submatch := reSig.FindStringSubmatch(auth)
	if submatch == nil || len(submatch) != 2 {
		return []byte{}, errors.New("Failed to match 'Spring-83 Signature' auth")
	}
	sig, err := hex.DecodeString(submatch[1])
	if err != nil {
		return []byte{}, err
	}
	return sig, nil
}

// parse timestamp from HTML time element
// <time datetime="YYYY-MM-DDTHH:MM:SSZ">
func ParseTimestamp(content []byte) (time.Time, error) {

	z := html.NewTokenizer(bytes.NewReader(content))
	for {
		switch tokType := z.Next(); {

		// reached the end
		case tokType == html.ErrorToken && z.Err() == io.EOF:
			return time.Time{}, errors.New("Unable to find a valid time element")

		// unexpected error parsing (boards should be parsable)
		case tokType == html.ErrorToken:
			return time.Time{}, z.Err()

		// time elements are "start tokens"
		case tokType == html.StartTagToken:
			tok := z.Token()

			if tok.Data == "time" && len(tok.Attr) == 1 && tok.Attr[0].Key == "datetime" {
				t, err := time.Parse(TimeFormat8601, tok.Attr[0].Val)
				if err != nil {
					// unparseable time tag (maybe there is another valid one)
					continue
				}
				// got a good timestamp
				return t, nil
			}
		}
	}

	// should not reach here
	return time.Time{}, errors.New("Unable to find a valid time element")
}

// TODO: level of precision for difficulty factor?
// difficultyFactor = ( numBoards / 10_000_000 )**4
func DifficultyFactor(numBoards int) float64 {
	return math.Pow(float64(numBoards)/maxNumBoards, 4)
}

// maxKey = (2**256 - 1)
func maxKey() *big.Int {
	maxKey := big.NewInt(2)
	maxKey.Exp(maxKey, big.NewInt(256), nil)
	maxKey.Sub(maxKey, big.NewInt(1))
	return maxKey
}

// keyThreshold = maxKey * ( 1.0 - difficultyFactor )
func KeyThreshold(difficultyFactor float64) *big.Int {
	threshold := new(big.Float)
	threshold.Sub(big.NewFloat(1), big.NewFloat(difficultyFactor))
	threshold.Mul(threshold, new(big.Float).SetInt(maxKey()))

	res, _ := threshold.Int(nil)
	return res
}
