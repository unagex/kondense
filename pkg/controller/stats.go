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

	// DefaultCPUMin in milliCPU, 10 is 0.01 cpu.
	DefaultCPUMin uint64 = 10

	// DefaultCPUMax in milliCPU, 100_000 is 100 cpus.
	DefaultCPUMax uint64 = 100_000
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

	TargetPressure uint64

	MaxInc   float64
	MaxDec   float64
	CoeffInc float64
	CoeffDec float64

	Min uint64
	Max uint64

	GraceTicks uint64
	Interval   uint64
}

type CPU struct {
	Limit     int64
	PrevTotal uint64
	Integral  uint64

	TargetPressure uint64

	MaxInc   float64
	MaxDec   float64
	CoeffInc float64
	CoeffDec float64

	Min uint64
	Max uint64

	GraceTicks uint64
	Interval   uint64
}
