package controller

type ContainerStats map[string]*Stats

type Stats struct {
	Mem Memory
}

type Memory struct {
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
