package statistics

import (
	"github.com/c9s/bbgo/pkg/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

/*
python

import quantstats as qx
import pandas as pd

print(qx.stats.sharpe(pd.Series([0.01, 0.1, 0.001]), 0, 0, False, False))
print(qx.stats.sharpe(pd.Series([0.01, 0.1, 0.001]), 0, 252, False, False))
print(qx.stats.sharpe(pd.Series([0.01, 0.1, 0.001]), 0, 252, True, False))
*/
func TestSharpe(t *testing.T) {
	var a types.Series = &types.Float64Slice{0.01, 0.1, 0.001}
	output := Sharpe(a, 0, false, false)
	assert.InDelta(t, output, 0.67586, 0.0001)
	output = Sharpe(a, 252, false, false)
	assert.InDelta(t, output, 0.67586, 0.0001)
	output = Sharpe(a, 252, true, false)
	assert.InDelta(t, output, 10.7289, 0.0001)
}
