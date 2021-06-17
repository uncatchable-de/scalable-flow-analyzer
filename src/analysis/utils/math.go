package utils

import "math"

const MaxUint16 = ^uint16(0)
const MaxUint16AsUint32 = uint32(MaxUint16)
const MaxUint32 = ^uint32(0)

func GetDistributionStats(values []int) (mean float64, min, max int, stdDev float64) {
	if len(values) == 0 {
		return mean, min, max, stdDev
	}
	sum := 0
	min = math.MaxInt64
	for _, value := range values {
		if value > max {
			max = value
		}
		if value < min {
			min = value
		}
		sum += value
	}
	mean = float64(sum) / float64(len(values))

	var sumStdDev float64
	for _, value := range values {
		sumStdDev += math.Pow(float64(value)-mean, 2)
	}

	variance := sumStdDev / float64(len(values))
	return mean, min, max, math.Sqrt(variance)
}
