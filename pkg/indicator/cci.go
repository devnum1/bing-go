package indicator

import (
	"math"

	"github.com/c9s/bbgo/pkg/fixedpoint"
	"github.com/c9s/bbgo/pkg/types"
)

// Refer: Commodity Channel Index
// Refer URL: http://www.andrewshamlet.net/2017/07/08/python-tutorial-cci
// with modification of ddof=0 to let standard deviation to be divided by N instead of N-1
//go:generate callbackgen -type CCI
type CCI struct {
	types.SeriesBase
	types.IntervalWindow
	Input        types.Float64Slice
	TypicalPrice types.Float64Slice
	MA           types.Float64Slice
	Values       types.Float64Slice

	UpdateCallbacks []func(value float64)
}

func (inc *CCI) Update(value float64) {
	if len(inc.TypicalPrice) == 0 {
		inc.SeriesBase.Series = inc
		inc.TypicalPrice.Push(value)
		inc.Input.Push(value)
		return
	} else if len(inc.TypicalPrice) > MaxNumOfEWMA {
		inc.TypicalPrice = inc.TypicalPrice[MaxNumOfEWMATruncateSize-1:]
		inc.Input = inc.Input[MaxNumOfEWMATruncateSize-1:]
	}

	inc.Input.Push(value)
	tp := inc.TypicalPrice.Last() - inc.Input.Index(inc.Window) + value
	inc.TypicalPrice.Push(tp)
	if len(inc.Input) < inc.Window {
		return
	}
	ma := tp / float64(inc.Window)
	inc.MA.Push(ma)
	if len(inc.MA) > MaxNumOfEWMA {
		inc.MA = inc.MA[MaxNumOfEWMATruncateSize-1:]
	}
	md := 0.
	for i := 0; i < inc.Window; i++ {
		diff := inc.Input.Index(i) - ma
		md += diff * diff
	}
	md = math.Sqrt(md / float64(inc.Window))

	cci := (value - ma) / (0.015 * md)

	inc.Values.Push(cci)
	if len(inc.Values) > MaxNumOfEWMA {
		inc.Values = inc.Values[MaxNumOfEWMATruncateSize-1:]
	}
}

func (inc *CCI) Last() float64 {
	if len(inc.Values) == 0 {
		return 0
	}
	return inc.Values[len(inc.Values)-1]
}

func (inc *CCI) Index(i int) float64 {
	if i >= len(inc.Values) {
		return 0
	}
	return inc.Values[len(inc.Values)-1-i]
}

func (inc *CCI) Length() int {
	return len(inc.Values)
}

var _ types.SeriesExtend = &CCI{}

var three = fixedpoint.NewFromInt(3)

func (inc *CCI) calculateAndUpdate(allKLines []types.KLine) {
	if inc.TypicalPrice.Length() == 0 {
		for _, k := range allKLines {
			inc.Update(k.High.Add(k.Low).Add(k.Close).Div(three).Float64())
			inc.EmitUpdate(inc.Last())
		}
	} else {
		k := allKLines[len(allKLines)-1]
		inc.Update(k.High.Add(k.Low).Add(k.Close).Div(three).Float64())
		inc.EmitUpdate(inc.Last())
	}
}

func (inc *CCI) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.calculateAndUpdate(window)
}

func (inc *CCI) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}
