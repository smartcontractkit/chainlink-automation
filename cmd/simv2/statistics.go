package main

import "math"

func findMedianAndSplitData(values []int) (median float64, a []int, b []int) {
	if len(values)%2 == 0 {
		// median is the average of the two middle integers
		idx := len(values) / 2
		median = float64(values[idx]+values[idx+1]) / 2

		a = values[:idx]
		b = values[idx:]
	} else {
		// median is the middle value
		idx := int(math.Floor(float64(len(values))/2)) + 1
		median = float64(values[idx])

		a = values[:idx]
		b = values[idx+1:]
	}

	return
}

func findLowestAndOutliers(lowerFence float64, set []int) (lowest int, outliers int) {
	lowest = math.MaxInt
	for i := 0; i < len(set); i++ {
		if set[i] < int(lowerFence) {
			outliers++
			if set[i] < lowest {
				lowest = set[i]
			}
		}
	}
	if lowest == math.MaxInt {
		lowest = -1
	}
	return
}

func findHighestAndOutliers(upperFence float64, set []int) (highest int, outliers int) {
	highest = -1
	for i := 0; i < len(set); i++ {
		if set[i] > int(upperFence) {
			outliers++
			if set[i] > highest {
				highest = set[i]
			}
		}
	}

	return
}
