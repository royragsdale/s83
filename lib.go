package s83

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/net/html"
)

// SpringVersion for use in http headers
const SpringVersion = "83"

const KeyLen = 64
const sigLen = 128

const maxNumBoards = 10_000_000

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

type CreatorResult struct {
	Creator Creator
	Count   int
	Err     error
}

func cancelableNewCreator(out chan *CreatorResult, ctx context.Context) {
	var err error
	cnt := 0
	c := Creator{}
	for !c.Publisher.valid() {
		select {
		case <-ctx.Done():
			out <- &CreatorResult{c, cnt, errors.New("canceled")}
			return
		default:
			c, err = genCreator()
			if err != nil {
				break
			}
			cnt += 1
		}
	}
	out <- &CreatorResult{c, cnt, err}
}

func NewCreator(j int) CreatorResult {
	out := make(chan *CreatorResult)
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < j; i++ {
		go cancelableNewCreator(out, ctx)
	}
	// block and wait for a winner
	creatorResult := <-out
	cancel()
	for i := 0; i < j-1; i++ {
		lostResult := <-out
		creatorResult.Count += lostResult.Count
	}

	return *creatorResult
}

func NewCreatorFromKey(privateKeyHex string) (Creator, error) {
	if len(privateKeyHex) != KeyLen {
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
	if len(publicKeyHex) != KeyLen {
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
	yearStr := p.String()[KeyLen-2:]
	keyYear, err := strconv.Atoi(yearStr)
	if err != nil {
		return false
	}

	monthStr := p.String()[KeyLen-4 : KeyLen-2]
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

func parseSignatureHeader(auth string) (Signature, error) {
	//Spring-Signature: <signature>
	reSig := regexp.MustCompile(`^[0-9A-Fa-f]{128}$`)
	match := reSig.FindString(auth)
	if match == "" {
		return []byte{}, errors.New("Invalid format for 'Spring-Signature'")
	}
	sig, err := hex.DecodeString(match)
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

	// will not reach here
}
