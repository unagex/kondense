package controller

import "time"

const (
	DefaultMemMin            uint64  = 50_000_000
	DefaultMemMax            uint64  = 100_000_000_000
	DefaultMemMaxInc         float64 = 0.5
	DefaultMemMaxDec         float64 = 0.02
	DefaultMemTargetPressure uint64  = 10_000
	DefaultMemInterval       uint64  = 10
	DefaultMemCoeffInc       float64 = 20
	DefaultMemCoeffDec       float64 = 10
)

const (
	DefaultCPUMin       uint64  = 10
	DefaultCPUMax       uint64  = 100_000
	DefaultCPUMaxInc    float64 = 0.5
	DefaultCPUMaxDec    float64 = 0.1
	DefaultCPUTargetAvg float64 = 0.8
	DefaultCPUInterval  uint64  = 6
	DefaultCPUCoeff     uint64  = 6
)

type ContainerStats map[string]*Stats

type Stats struct {
	Mem Memory
	Cpu CPU

	LastUpdate time.Time
}

type Memory struct {
	// Limit is the memory limit in bytes of the container.
	Limit int64
	// Min is the minimum memory limit in bytes allowed on the container.
	Min uint64
	// Max is the maximum memory limit in bytes allowed on the container.
	Max uint64
	// PrevTotal is the previous total of memory used in bytes on the container.
	PrevTotal uint64
	// Integral is the sum of memory used every second.
	// It is put back to 0 when Interval of time passed or when the memory is patched.
	Integral uint64
	// Target presssure is the target memory pressure in microseconds of the container.
	TargetPressure uint64
	// MaxInc is the max memory increase in percent allowed. For example 0.5 means kondense can increase the memory limit up to 50%.
	MaxInc float64
	// MaxDec is the max memory decrease in percent allowed. For example 0.5 means kondense can decrease the memory limit up to 50%.
	MaxDec float64
	// CoeffInc defines how sensitive we are to fluctuations around the target pressure when pressure is higher than target pressure.
	// e.g. when CoeffInc is 10, the curve reaches MaxInc when pressure is 10 times the target pressure.
	CoeffInc float64
	// CoeffDec defines how sensitive we are to fluctuations around the target pressure when pressure is lower than target pressure.
	// e.g. when CoeffDec is 10, the curve reaches MaxDec when pressure is 1/10 times the target pressure.
	CoeffDec float64
	// Interval is the number of seconds to calculate the target memory pressure.
	// e.g. when Interval is 7, it means that ideally the target memory pressure should be obtained after 7 seconds.
	Interval uint64
	// GraceTicks is the number of seconds passed since Interval went to 0 for the last time.
	GraceTicks uint64
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
