package s83

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
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

func NewCreator() (Creator, error) {
	var err error
	cnt := 0
	c := Creator{}
	for !c.Publisher.valid() {
		c, err = genCreator()
		if err != nil {
			return c, err
		}
		cnt += 1
	}
	fmt.Printf("found valid key in %d iterations\n", cnt)
	return c, nil
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

func (c Creator) NewBoard(content []byte) (Board, error) {

	// validate
	if !utf8.Valid(content) {
		return Board{}, errors.New("Invalid Board: not UTF-8")
	}

	// prepend timestamp tag
	timestamp := time.Now().UTC()
	httpTime := timestamp.Format(http.TimeFormat)
	lastModMeta := `<meta http-equiv="last-modified" content="%s">`
	lastMod := []byte(fmt.Sprintf(lastModMeta, httpTime))
	content = append(lastMod, content...)

	if len(content) > maxBoardLen {
		max := maxBoardLen - len(lastMod)
		msg := fmt.Sprintf("Invalid Board: too large (%d). Max: %d (leaves room for timestamp) ", len(content), max)
		return Board{}, errors.New(msg)
	}

	sig := ed25519.Sign(c.PrivateKey, content)
	return Board{c.Publisher, timestamp, sig, content}, nil

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
	return fmt.Sprintf("%s sends:\n%s\nsig verifies: %t", b.Publisher, b.Content, b.VerifySignature())
}

func (b Board) VerifySignature() bool {
	return ed25519.Verify(b.Publisher.PublicKey, b.Content, b.signature)
}

// TODO: function to parse timestamp from HTML meta tag.
//func (b Board) ParseTimeStamp() time.Time {

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
