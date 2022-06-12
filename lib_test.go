package s83

import (
	"math/big"
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
		t.Errorf(`Error creating board: %v`, err)
	}

	if !board.VerifySignature() {
		t.Errorf("Board failed signature verification")
	}

	// force invalid signature
	board.signature = []byte("xxx")
	if board.VerifySignature() {
		t.Errorf("Board with bad signature should fail")
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
