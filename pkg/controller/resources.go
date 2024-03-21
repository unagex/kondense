package controller

import (
	"fmt"
)

type Resources struct {
	Memory Pressure
}

func (r *Resources) String() string {
	return fmt.Sprintf("memory: {limit: %d, prevTotal: %d, integral: %d, current: %d}",
		r.Memory.Limit,
		r.Memory.PrevTotal,
		r.Memory.Integral,
		r.Memory.Current)
}

type Pressure struct {
	Limit     int64
	PrevTotal int64
	Integral  int64
	Current   int64

	AVG10  float64
	AVG60  float64
	AVG300 float64

	GraceTicks int
	Interval   int
}
