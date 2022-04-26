package indicator

import (
	"github.com/c9s/bbgo/pkg/types"
)

// Refer: Zero Lag Exponential Moving Average
// Refer URL: https://en.wikipedia.org/wiki/Zero_lag_exponential_moving_average

//go:generate callbackgen -type ZLEMA
type ZLEMA struct {
	types.IntervalWindow

	data  types.Float64Slice
	zlema *EWMA
	lag   int

	UpdateCallbacks []func(value float64)
}

func (inc *ZLEMA) Index(i int) float64 {
	if inc.zlema == nil {
		return 0
	}
	return inc.zlema.Index(i)
}

func (inc *ZLEMA) Last() float64 {
	if inc.zlema == nil {
		return 0
	}
	return inc.zlema.Last()
}

func (inc *ZLEMA) Length() int {
	if inc.zlema == nil {
		return 0
	}
	return inc.zlema.Length()
}

func (inc *ZLEMA) Update(value float64) {
	if inc.lag == 0 || inc.zlema == nil {
		inc.zlema = &EWMA{IntervalWindow: types.IntervalWindow{inc.Interval, inc.Window}}
		inc.lag = int((float64(inc.Window)-1.)/2. + 0.5)
	}
	inc.data.Push(value)
	if len(inc.data) > MaxNumOfEWMA {
		inc.data = inc.data[MaxNumOfEWMATruncateSize-1:]
	}
	if inc.lag >= inc.data.Length() {
		return
	}
	emaData := 2.*value - inc.data[len(inc.data)-1-inc.lag]
	inc.zlema.Update(emaData)
}

var _ types.Series = &ZLEMA{}

func (inc *ZLEMA) calculateAndUpdate(allKLines []types.KLine) {
	if inc.zlema == nil {
		for _, k := range allKLines {
			inc.Update(k.Close.Float64())
			inc.EmitUpdate(inc.Last())
		}
	} else {
		inc.Update(allKLines[len(allKLines)-1].Close.Float64())
		inc.EmitUpdate(inc.Last())
	}
}

func (inc *ZLEMA) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.calculateAndUpdate(window)
}

func (inc *ZLEMA) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}
