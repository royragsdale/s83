package s83

import (
	"math"
	"math/big"
)

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
