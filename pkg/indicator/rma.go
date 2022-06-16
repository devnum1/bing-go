package indicator

import (
	"time"

	"github.com/c9s/bbgo/pkg/types"
)

// Running Moving Average
// Refer: https://github.com/twopirllc/pandas-ta/blob/main/pandas_ta/overlap/rma.py#L5
// Refer: https://pandas.pydata.org/docs/reference/api/pandas.DataFrame.ewm.html#pandas-dataframe-ewm
//go:generate callbackgen -type RMA
type RMA struct {
	types.IntervalWindow
	Values          types.Float64Slice
	counter         int
	Adjust          bool
	tmp             float64
	sum             float64
	EndTime         time.Time
	UpdateCallbacks []func(value float64)
}

func (inc *RMA) Update(x float64) {
	lambda := 1 / float64(inc.Window)
	if inc.counter == 0 {
		inc.sum = 1
		inc.tmp = x
	} else {
		if inc.Adjust {
			inc.sum = inc.sum*(1-lambda) + 1
			inc.tmp = inc.tmp + (x-inc.tmp)/inc.sum
		} else {
			inc.tmp = inc.tmp*(1-lambda) + x*lambda
		}
	}
	inc.counter++

	if inc.counter < inc.Window {
		inc.Values.Push(0)
		return
	}

	inc.Values.Push(inc.tmp)
}

func (inc *RMA) Last() float64 {
	return inc.Values.Last()
}

func (inc *RMA) Index(i int) float64 {
	length := len(inc.Values)
	if length == 0 || length-i-1 < 0 {
		return 0
	}
	return inc.Values[length-i-1]
}

func (inc *RMA) Length() int {
	return len(inc.Values)
}

var _ types.Series = &RMA{}

func (inc *RMA) calculateAndUpdate(kLines []types.KLine) {
	for _, k := range kLines {
		if inc.EndTime != zeroTime && !k.EndTime.After(inc.EndTime) {
			continue
		}
		inc.Update(k.Close.Float64())
	}

	inc.EmitUpdate(inc.Last())
	inc.EndTime = kLines[len(kLines)-1].EndTime.Time()
}
func (inc *RMA) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.calculateAndUpdate(window)
}

func (inc *RMA) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}
