package controller

const (
	DefaultMemInterval       uint64  = 2
	DefaultMemTargetPressure uint64  = 10_000
	DefaultMemMaxProbe       float64 = 0.01
	DefaultMemMaxBackoff     float64 = 1
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

	MaxProbe   float64
	MaxBackOff float64

	AVG10  float64
	AVG60  float64
	AVG300 float64

	GraceTicks uint64
	Interval   uint64
}
