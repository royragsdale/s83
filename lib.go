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

const TestPublic = "fad415fbaa0339c4fd372d8287e50f67905321ccfd9c43fa4c20ac40afed1983"
const TestPrivate = "a7e4d1c8be858d683ab9cb15574bd0bc3a87e6c846cdaf848da498909cb574f7"

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

func (c Creator) NewBoard(content []byte) (Board, error) {

	// check board doesn't already have a timestamp
	ts, err := ParseTimestamp(content)
	if err != nil {
		// TODO: consider other error cases (e.g. unparsable/multiple)
		// no good timestamp, so helpfully prepend one
		httpTime := time.Now().UTC().Format(http.TimeFormat)
		lastModMeta := `<meta http-equiv="last-modified" content="%s">`
		lastMod := []byte(fmt.Sprintf(lastModMeta, httpTime))
		content = append(lastMod, content...)
	} else if ts.After(time.Now().UTC()) {
		// check the timestamp provided is not in the future
		return Board{}, errors.New("last-modified timestamp is in the future")
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
	// ends in ed20XX where X are digits and "must fall in the range 2022 .. 2099"
	reValidKey := regexp.MustCompile(`ed20[2-9][0-9]$`)
	if !reValidKey.MatchString(p.String()) {
		return false
	}

	// "Keys are only valid in two calendar years:
	//		the year specified in their final four digits,
	//		and the year previous."
	keyYear, err := strconv.Atoi(p.String()[len(p.String())-4:])
	if err != nil {
		return false
	}
	curYear := time.Now().Year()

	return (curYear-1 <= keyYear) && (keyYear <= curYear)
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

// parse timestamp from HTML meta tag
// <meta http-equiv="last-modified" content="<date and time in HTTP format>">
func ParseTimestamp(content []byte) (time.Time, error) {
	ts := time.Time{}

	z := html.NewTokenizer(bytes.NewReader(content))
tokLoop:
	for {
		switch tokType := z.Next(); {

		// reached the end
		case tokType == html.ErrorToken && z.Err() == io.EOF:
			break tokLoop

		// unexpected error parsing (boards should be parsable)
		case tokType == html.ErrorToken:
			return time.Time{}, z.Err()

		// meta tags are "start tokens"
		case tokType == html.StartTagToken:
			tok := z.Token()
			// TODO: case insensitivity?
			if tok.Data == "meta" && len(tok.Attr) == 2 {
				a := tok.Attr[0]
				b := tok.Attr[1]
				tStr := ""
				if a.Key == "http-equiv" && a.Val == "last-modified" && b.Key == "content" {
					tStr = b.Val
				} else if b.Key == "http-equiv" && b.Val == "last-modified" && a.Key == "content" {
					tStr = a.Val
				} else {
					// some other meta tag
					break
				}

				// check if we have already found a good last-modified meta tag
				if !ts.IsZero() {
					return time.Time{}, errors.New("Multiple last-modified meta tags")
				}

				t, err := http.ParseTime(tStr)
				if err != nil {
					return time.Time{}, errors.New("Unparsable last-modified meta tag")
				}
				// got a good timestamp
				ts = t
			}
		}
	}

	// got to the end of the board without finding a last-modified meta tag
	if ts.IsZero() {
		return ts, errors.New("Unable to find a last-modified meta tag")
	} else {
		return ts, nil
	}
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
