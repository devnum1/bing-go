package indicator

import (
	"math"

	"github.com/c9s/bbgo/pkg/types"
)

// Refer: Variable Index Dynamic Average
// Refer URL: https://metatrader5.com/en/terminal/help/indicators/trend_indicators/vida
//go:generate callbackgen -type VIDYA
type VIDYA struct {
	types.IntervalWindow
	Values types.Float64Slice
	input  types.Float64Slice

	UpdateCallbacks []func(value float64)
}

func (inc *VIDYA) Update(value float64) {
	if inc.Values.Length() == 0 {
		inc.Values.Push(value)
		inc.input.Push(value)
		return
	}
	inc.input.Push(value)
	if len(inc.input) > MaxNumOfEWMA {
		inc.input = inc.input[MaxNumOfEWMATruncateSize-1:]
	}
	/*upsum := 0.
	downsum := 0.
	for i := 0; i < inc.Window; i++ {
		if len(inc.input) <= i+1 {
			break
		}
		diff := inc.input.Index(i) - inc.input.Index(i+1)
		if diff > 0 {
			upsum += diff
		} else {
			downsum += -diff
		}

	}
	if upsum == 0 && downsum == 0 {
		return
	}
	CMO := math.Abs((upsum - downsum) / (upsum + downsum))*/
	change := types.Change(&inc.input)
	CMO := math.Abs(types.Sum(change, inc.Window) / types.Sum(types.Abs(change), inc.Window))
	alpha := 2. / float64(inc.Window+1)
	inc.Values.Push(value*alpha*CMO + inc.Values.Last()*(1.-alpha*CMO))
	if inc.Values.Length() > MaxNumOfEWMA {
		inc.Values = inc.Values[MaxNumOfEWMATruncateSize-1:]
	}
}

func (inc *VIDYA) Last() float64 {
	return inc.Values.Last()
}

func (inc *VIDYA) Index(i int) float64 {
	return inc.Values.Index(i)
}

func (inc *VIDYA) Length() int {
	return inc.Values.Length()
}

var _ types.Series = &VIDYA{}

func (inc *VIDYA) calculateAndUpdate(allKLines []types.KLine) {
	if inc.input.Length() == 0 {
		for _, k := range allKLines {
			inc.Update(k.Close.Float64())
			inc.EmitUpdate(inc.Last())
		}
	} else {
		inc.Update(allKLines[len(allKLines)-1].Close.Float64())
		inc.EmitUpdate(inc.Last())
	}
}

func (inc *VIDYA) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.calculateAndUpdate(window)
}

func (inc *VIDYA) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}
