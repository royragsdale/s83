package s83

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGenCreator(t *testing.T) {
	// fast: not necessarily a valid creator
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

	if testCreator.Valid() {
		t.Fatalf("Test creator should not be valid (expired)")
	}

	if !testCreator.PublicKey.Equal(testPublisher.PublicKey) {
		t.Fatalf(`Error matching creator/publisher that share a key: %s %s`, testCreator, testPublisher)
	}

	_, err = NewCreatorFromKey("a")
	if err == nil {
		t.Errorf("NewCreatorFromKey should error on a short key")
	}

	_, err = NewCreatorFromKey("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	if err == nil {
		t.Errorf("NewCreatorFromKey should error on a a non-hex key")
	}

}

func dateToKey(t time.Time) string {
	stub := strings.Repeat("a", KeyLen-7) // must be hex char
	prefix := "83e"                       // valid prefix
	return fmt.Sprintf("%s%s%02d%s", stub, prefix, int(t.Month()), strconv.Itoa(t.Year())[2:])
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

	cur := time.Now().UTC()

	// valid
	yr1 := dateToKey(cur.AddDate(1, 0, 0))
	yr2 := dateToKey(cur.AddDate(2, 0, 0))

	// invalid
	badPrefix := strings.Repeat("a", KeyLen-4) + yr1[KeyLen-4:]
	prevM := dateToKey(cur.AddDate(0, -1, 0)) // expired
	yr2m1 := dateToKey(cur.AddDate(2, 1, 0))  // not yet valid

	// bad keys (should error on load)
	short := "a"
	nonHex := strings.Repeat("X", 64) // must be hex char

	type keyTest struct {
		name        string
		key         string
		valid       bool
		errExpected bool
	}
	var keyTests = []keyTest{
		{"cur", dateToKey(cur), true, false}, {"yr 1", yr1, true, false}, {"yr 2", yr2, true, false},
		{"bad prefix", badPrefix, false, false}, {"next2", prevM, false, false}, {"next2", yr2m1, false, false},
		{"short", short, false, true}, {"non hex", nonHex, false, true},
	}

	for _, tt := range keyTests {
		p, err := NewPublisherFromKey(tt.key)
		if err != nil && !tt.errExpected {
			t.Errorf(`Unexpected error loading publisher from key: %s: %v`, tt.name, err)
		}
		actual := p.valid()
		if actual != tt.valid {
			t.Errorf("Wrong key validity (%s : %s): expected %t, actual %t", tt.name, tt.key, tt.valid, actual)
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
		t.Errorf(`Error creating board: %v`, err)
	}

	if !board.VerifySignature() {
		t.Errorf("Board failed signature verification")
	}

	if len(board.Signature()) != sigLen {
		t.Errorf("Signature should be %d long", KeyLen)
	}

	// force invalid signature
	board.signature = []byte("xxx")
	if board.VerifySignature() {
		t.Errorf("Board with bad signature should fail")
	}

	now := time.Now().UTC()
	second := now.AddDate(-1, 0, 0)

	multipleTS := fmt.Sprintf(`%s%s`, timeElem(now), timeElem(second))

	board, err = creator.NewBoard([]byte(multipleTS))
	if err != nil {
		t.Errorf("Board with multiple timestamps is valid should take first seen")
	}

	// compare strings so that it strips the fractional seconds
	if board.timestamp.Format(TimeFormat8601) != now.Format(TimeFormat8601) {
		t.Errorf("Board with multiple timestamps is valid should take first seen")
		fmt.Println(board.timestamp, now)
	}

	future := now.AddDate(0, 0, 1)
	board, err = creator.NewBoard([]byte(timeElem(future)))
	if err == nil {
		t.Errorf("Board with future timestamps should fail")
	}

}

func TestStringFormats(t *testing.T) {
	creator, err := genCreator()
	if err != nil || creator.PrivateKey == nil || creator.PublicKey == nil {
		t.Fatalf(`Failure making creator: %v %v`, creator, err)
	}

	keys := []string{creator.String(), creator.ExportPrivateKey(), creator.Publisher.String()}
	for _, key := range keys {
		if len(key) != KeyLen {
			t.Errorf("Creator should be %d long", KeyLen)
		}

		_, err = hex.DecodeString(key)
		if err != nil {
			t.Errorf("Keys should be hex encoded")
		}
	}
}

func TestDifficulty(t *testing.T) {

	tests := []struct {
		factor    float64
		threshold uint64
	}{{1.0, 0}, {0.0, MaxKey}}
	for _, tt := range tests {
		dT, err := DifficultyThreshold(tt.factor)
		if err != nil {
			t.Fatalf(`Error with valid difficulty factor: %f %v`, tt.factor, err)
		}
		if tt.threshold != dT {
			t.Errorf("Factor %f, expected threshold %d, got %d", tt.factor, tt.threshold, dT)
		}
	}

	// test random creators and round trip strength/difficulty factors to ensure they are valid
	for i := 0; i < 10; i++ {
		creator, err := genCreator()
		strength := creator.Strength()
		factor := StrengthFactor(strength)
		dT, err := DifficultyThreshold(factor)
		if err != nil {
			t.Fatalf(`Error with difficulty factor derived from creator: %f %v`, factor, err)
		}
		if strength >= dT {
			t.Errorf("Predicted strength factor %f, exceeded threshold %d for strength=%d, c=%s", factor, dT, strength, creator)
		}
	}

}

// TODO: test strings
// TODO: test NewBoard edge cases
// TODO: NewBoardFromHTTP
// TODO: parseSignatureHeader
// TODO: test ParseTimestamp directly for edge cases (including capitalization)
