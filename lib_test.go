package s83

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGenCreator(t *testing.T) {
	creator, err := genCreator()
	if err != nil || creator.PrivateKey == nil || creator.PublicKey == nil {
		t.Fatalf(`Failure making creator: %v %v`, creator, err)
	}
}

func TestLoadingFromTestKeys(t *testing.T) {
	testPublisher, err := NewPublisherFromKey(TestPublic)
	if err != nil {
		t.Fatalf(`Error loading publisher from key: %v`, err)
	}

	testCreator, err := NewCreatorFromKey(TestPrivate)
	if err != nil {
		t.Fatalf(`Error loading creator from key: %v`, err)
	}

	if !testCreator.PublicKey.Equal(testPublisher.PublicKey) {
		t.Fatalf(`Error matching creator/publisher that share a key: %s %s`, testCreator, testPublisher)
	}
}

func TestKeyValidity(t *testing.T) {

	// Publisher
	testPublisher, err := NewPublisherFromKey(TestPublic)
	if err != nil {
		t.Fatalf(`Error loading publisher from key: %v`, err)
	}

	if testPublisher.valid() {
		t.Fatalf(`Test key should not be valid (1983): %v`, testPublisher)
	}

	// supporting strings
	stub58 := strings.Repeat("a", 58) // must be hex char
	prefix := stub58 + "ed"           // valid prefix
	year := time.Now().Year()

	// valid
	curYear := prefix + strconv.Itoa(year)
	prevYear := prefix + strconv.Itoa(year-1)

	// invalid
	badPrefix := stub58 + "bb" + strconv.Itoa(year) // prefix must be ed
	nextYear := prefix + strconv.Itoa(year+1)       // future years are invalid

	// bad keys (should error on load)
	short := "a"
	nonHex := strings.Repeat("X", 64) // must be hex char

	type keyTest struct {
		key         string
		valid       bool
		errExpected bool
	}
	var keyTests = []keyTest{
		{curYear, true, false}, {prevYear, true, false},
		{badPrefix, false, false}, {nextYear, false, false},
		{short, false, true}, {nonHex, false, true},
	}

	for _, tt := range keyTests {
		p, err := NewPublisherFromKey(tt.key)
		if err != nil && !tt.errExpected {
			t.Errorf(`Unexpected error loading publisher from key: %v`, err)
		}
		actual := p.valid()
		if actual != tt.valid {
			t.Errorf("Wrong key validity (%s): expected %t, actual %t", tt.key, tt.valid, actual)
		}
	}

}

func TestBoardCreation(t *testing.T) {

	creator, err := NewCreatorFromKey(TestPrivate)
	if err != nil {
		t.Fatalf(`Error loading creator from key: %v`, err)
	}

	board, err := creator.NewBoard([]byte("foo"))
	if err != nil {
		t.Fatalf(`Error creating board: %v`, err)
	}

	if !board.VerifySignature() {
		t.Fatalf("Board failed signature verification")
	}

	// force invalid signature
	board.signature = []byte("xxx")
	if board.VerifySignature() {
		t.Fatalf("Board with bad signature should fail")
	}

}
