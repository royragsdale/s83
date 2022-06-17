package s83

import (
	"encoding/hex"
	"fmt"
	"math/big"
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
	stub := strings.Repeat("a", keyLen-7) // must be hex char
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
	badPrefix := strings.Repeat("a", keyLen-4) + yr1[keyLen-4:]
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
		t.Errorf("Signature should be %d long", keyLen)
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

func TestDifficultyFactor(t *testing.T) {
	type difficultyTest struct {
		num       int
		factor    float64
		threshold *big.Int
	}

	// setup <an inscrutable gigantic number> (matches example in spec)
	specExBoards := 8_500_000
	specExFactor := 0.5220062499999999
	gS := "55347894954879420465823292524996605235896494567000172791001875836472156749824"
	gF := new(big.Float)
	gF, _, err := gF.Parse(gS, 10)
	if err != nil {
		t.Fatalf("Error creating fixed threshold test value (parsing)")
	}

	specExThresh, acc := gF.Int(nil)
	if acc != big.Exact {
		t.Fatalf("Error creating fixed threshold test value (converting)")
	}

	var difficultyTests = []difficultyTest{
		// TODO: check off by one, intuitively it is ok that it is 1 more
		// because keys have to be "less than" the threshold, but why?
		{0, 0, maxKey().Add(maxKey(), big.NewInt(1))},
		{specExBoards, specExFactor, specExThresh},
		{maxNumBoards, 1, big.NewInt(0)},
	}

	for _, tt := range difficultyTests {
		df := DifficultyFactor(tt.num)
		threshold := KeyThreshold(df)
		if df != tt.factor {
			t.Errorf("Wrong difficulty factor: expected %f, actual %f", tt.factor, df)
		}

		if threshold.Cmp(tt.threshold) != 0 {
			t.Errorf("Wrong key threshold: expected %d, actual %d", tt.threshold, threshold)
		}
	}
}

func TestStringFormats(t *testing.T) {
	creator, err := genCreator()
	if err != nil || creator.PrivateKey == nil || creator.PublicKey == nil {
		t.Fatalf(`Failure making creator: %v %v`, creator, err)
	}

	keys := []string{creator.String(), creator.ExportPrivateKey(), creator.Publisher.String()}
	for _, key := range keys {
		if len(key) != keyLen {
			t.Errorf("Creator should be %d long", keyLen)
		}

		_, err = hex.DecodeString(key)
		if err != nil {
			t.Errorf("Keys should be hex encoded")
		}
	}
}

// TODO: test strings
// TODO: test NewBoard edge cases
// TODO: NewBoardFromHTTP
// TODO: parseAuthorizationHeader
// TODO: test ParseTimestamp directly for edge cases (including capitalization)
