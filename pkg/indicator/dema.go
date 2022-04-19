package indicator

import (
	"github.com/c9s/bbgo/pkg/types"
)

// Refer: Double Exponential Moving Average
// Refer URL: https://investopedia.com/terms/d/double-exponential-moving-average.asp

//go:generate callbackgen -type DEMA
type DEMA struct {
	types.IntervalWindow
	Values types.Float64Slice
	a1     *EWMA
	a2     *EWMA

	UpdateCallbacks []func(value float64)
}

func (inc *DEMA) Update(value float64) {
	if len(inc.Values) == 0 {
		inc.a1 = &EWMA{IntervalWindow: types.IntervalWindow{inc.Interval, inc.Window}}
		inc.a2 = &EWMA{IntervalWindow: types.IntervalWindow{inc.Interval, inc.Window}}
	}

	inc.a1.Update(value)
	inc.a2.Update(inc.a1.Last())
	inc.Values.Push(2*inc.a1.Last() - inc.a2.Last())
	if len(inc.Values) > MaxNumOfEWMA {
		inc.Values = inc.Values[MaxNumOfEWMATruncateSize-1:]
	}
}

func (inc *DEMA) Last() float64 {
	return inc.Values.Last()
}

func (inc *DEMA) Index(i int) float64 {
	if len(inc.Values)-i-1 >= 0 {
		return inc.Values[len(inc.Values)-1-i]
	}
	return 0
}

func (inc *DEMA) Length() int {
	return len(inc.Values)
}

var _ types.Series = &DEMA{}

func (inc *DEMA) calculateAndUpdate(allKLines []types.KLine) {
	for _, k := range allKLines {
		inc.Update(k.Close.Float64())
		inc.EmitUpdate(inc.Last())
	}
}

func (inc *DEMA) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.calculateAndUpdate(window)
}

func (inc *DEMA) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}
