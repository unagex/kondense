package controller

const (
	DefaultMemInterval uint64 = 10
	// DefaultMemTargetPressure in bytes
	DefaultMemTargetPressure uint64  = 10_000
	DefaultMemMaxInc         float64 = 0.5
	DefaultMemMaxDec         float64 = 0.02
	DefaultMemCoeffInc       float64 = 20
	DefaultMemCoeffDec       float64 = 10

	// DefaultMemMin is 50M
	DefaultMemMin uint64 = 50_000_000
	// DefaultMemMax is 100G
	DefaultMemMax uint64 = 100_000_000_000
)

const (
	DefaultCPUInterval uint64 = 1
	// DefaultCPUTargetPressure in nanoseconds
	DefaultCPUTargetPressure uint64  = 10_000_000
	DefaultCPUMaxInc         float64 = 0.5
	DefaultCPUMaxDec         float64 = 0.02
	DefaultCPUCoeffInc       float64 = 20
	DefaultCPUCoeffDec       float64 = 10

	DefaultCPUMin float64 = 0.1
	DefaultCPUMax float64 = 100
)

type ContainerStats map[string]*Stats

type Stats struct {
	Mem Memory
	Cpu CPU
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

type CPU struct {
	// Limit     int64
	// PrevTotal uint64
	// Integral  uint64
	// Current   int64

	TargetPressure uint64

	MaxInc   float64
	MaxDec   float64
	CoeffInc float64
	CoeffDec float64

	Min float64
	Max float64

	// AVG10  float64
	// AVG60  float64
	// AVG300 float64

	GraceTicks uint64
	Interval   uint64
}
