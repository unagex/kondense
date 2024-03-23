package controller

const (
	DefaultMemInterval       uint64  = 2
	DefaultMemTargetPressure uint64  = 10_000
	DefaultMemMaxInc         float64 = 1
	DefaultMemMaxDec         float64 = 0.01
	DefaultMemCoeffInc       float64 = 20
	DefaultMemCoeffDec       float64 = 10

	// DefaultMemMin is 10M
	DefaultMemMin uint64 = 10_000_000
	// DefaultMemMax is 100G
	DefaultMemMax uint64 = 100_000_000_000
)

type ContainerStats map[string]*Stats

type Stats struct {
	Mem Memory
}

type Memory struct {
	Limit     int64
	PrevTotal uint64
	Integral  uint64
	Current   int64

	TargetPressure uint64

	MaxInc   float64
	MaxDec   float64
	CoeffInc float64
	CoeffDec float64

	Min uint64
	Max uint64

	AVG10  float64
	AVG60  float64
	AVG300 float64

	GraceTicks uint64
	Interval   uint64
}
