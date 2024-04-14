package controller

import "time"

const (
	DefaultMemInterval uint64 = 10
	// DefaultMemTargetPressure in microseconds
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
	DefaultCPUMin       uint64  = 10
	DefaultCPUMax       uint64  = 100_000
	DefaultCPUInterval  uint64  = 6
	DefaultCPUTargetAvg float64 = 0.8
	DefaultCPUCoeff     uint64  = 6
	DefaultCPUMaxInc    float64 = 0.5
	DefaultCPUMaxDec    float64 = 0.1
)

type ContainerStats map[string]*Stats

type Stats struct {
	Mem Memory
	Cpu CPU

	LastUpdate time.Time
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
	// Limit is the cpu limit of the container in millicpus.
	Limit int64
	// Min is the minimum cpu limit allowed on the container in millicpus.
	Min uint64
	// Max is the maximum cpu limit allowed on the container in millicpus.
	Max uint64
	// TargetAvg is the target cpu usage of the container. It is from 0 to 1. e.g. 0.8 means a target cpu usage of 80%.
	TargetAvg float64
	// MaxInc is the max cpu increase in percent allowed. For example 0.5 means kondense can increase the cpu limit up to 50%.
	MaxInc float64
	// MaxDec is the max cpu decrease in percent allowed. For example 0.5 means kondense can decrease the cpu limit up to 50%.
	MaxDec float64
	// Coeff is used to calculate the new cpu limit when a cpu increase is needed. The higher the coeff, the higher the new cpu limit.
	Coeff uint64
	// Interval is the interval in seconds used to calculate the cpu average usage.
	Interval uint64
	// Usage is a queue to store total cpu usage at a specific time.
	Usage []CPUProbe
	// Avg is the cpu average usage in millicpus.
	Avg uint64
}

type CPUProbe struct {
	Usage uint64
	T     time.Time
}
