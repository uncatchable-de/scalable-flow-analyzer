package common

type ExportUnivariateFormat struct {
	Values [][]int // List of [value, counter] tuples
}

type ExportUnivariateClusterFormat struct {
	Clusters map[int]*ExportUnivariateFormat
}

type ExportBivariateFormat struct {
	Variable map[int]*ExportUnivariateFormat
}

type ExportBivariateClusterFormat struct {
	Clusters map[int]*ExportBivariateFormat
}
