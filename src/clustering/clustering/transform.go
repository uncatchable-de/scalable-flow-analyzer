package clustering

import (
	"clustering/dataformat"
	"math"
)

// GetDataOfRRP returns the rrp data
func GetDataOfRRP(rrp *dataformat.RRP) []float64 {
	return []float64{
		float64(rrp.RequestSize),
		float64(rrp.ResponseSize),
	}
}

// GetDataOfFlow returns the flow data
func GetDataOfFlow(flow *dataformat.Flow) []float64 {
	return []float64{
		float64(flow.NumRrp),
		//float64(flow.InterReq.Max),
		//float64(flow.InterReq.Min),
		float64(flow.InterReq.Mean),
		//float64(flow.InterReq.StdDev),
	}
}

// GetDataOfSession returns the session data
func GetDataOfSession(session *dataformat.Session) []float64 {
	return []float64{
		float64(session.NumServers),
		float64(session.NumFlows),
		//float64(session.InterFlow.Max),
		//float64(session.InterFlow.Min),
		float64(session.InterFlow.Mean),
		//float64(session.InterFlow.StdDev),
	}
}

// GetDataOfUser returns the user data
func GetDataOfUser(user *dataformat.User) []float64 {
	return []float64{
		float64(user.NumSessions),
		//float64(user.InterSession.Max),
		//float64(user.InterSession.Min),
		float64(user.InterSession.Mean),
		//float64(user.InterSession.StdDev),
	}
}

// ScaleDatas scales the data according to the scalingFactor
//
// If scalingFactors is nil, the data will be linearly scaled to 0...1
// If logScaling is true for index i, the feature at index i will be first log10 scaled
// If the data contains negative values it will scale the data to -1...0...1
func ScaleDatas(data [][]float64, scalingFactors []float64, logScaling bool) (scaleFactors []float64) {
	for i, _ := range data {
		for j, _ := range data[i] {
			if logScaling {
				// do not log scale negative values or values lower 0
				if data[i][j] > 0 {
					data[i][j] = math.Log10(data[i][j])
				} else {
					// Due to float arithmetic, 5 flows had -0.0e-8 ns inter-flow times, when analyzing generated traffic
					// This is impossible and we fix it by setting it to zero
					// Did not happen for analyzing CAIDA traffic
					data[i][j] = 0
				}
			}
		}
	}

	if scalingFactors == nil {
		scalingFactors = make([]float64, len(data[0]))
		for i, _ := range data {
			for j, _ := range data[i] {
				scalingFactors[j] = math.Max(math.Abs(data[i][j]), scalingFactors[j])
			}
		}
	}
	for i, _ := range data {
		for j, _ := range data[i] {
			data[i][j] = data[i][j] / scalingFactors[j]
		}
	}
	return scalingFactors
}

// ScaleData scales the data according to the scalingFactor
func ScaleData(data []float64, options *SaveOptions) {
	for i, _ := range data {
		if data[i] != 0 && options.ScaleLog {
			data[i] = math.Log10(data[i])
		}
		data[i] = data[i] / options.ScaleFactors[i]
	}
}
