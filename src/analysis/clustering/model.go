package clustering

import (
	"github.com/uncatchable-de/goml/cluster"

	"encoding/json"
	"io/ioutil"
)

type SaveOptions struct {
	NumIterations int
	NumClusters   int
	NumFeatures   int
	Distortion    float64
	ScaleFactors  []float64
	ScaleLog      bool
}

// Load a model from the files created with SaveModel
// filepath must point to the .json file containing basic information
// filepath.data is then the model containing the clusters
func LoadModel(filepath string) (*cluster.KMeans, *SaveOptions, error) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, nil, err
	}
	options := &SaveOptions{}
	err = json.Unmarshal(b, options)
	if err != nil {
		return nil, nil, err
	}

	data := make([][]float64, 1)
	data[0] = make([]float64, options.NumFeatures)
	model := cluster.NewKMeans(options.NumClusters, options.NumIterations, data)
	err = model.RestoreFromFile(filepath + ".data")
	if err != nil {
		return nil, nil, err
	}
	return model, options, nil
}

// Save Model to filepath.
// Creates the filepath file with global options and
// a filepath.data file with the centroids
func SaveModel(model *cluster.KMeans, scaleFactors []float64, scaleLog bool, filepath string) error {
	options := SaveOptions{
		NumClusters:   len(model.Centroids),
		NumFeatures:   len(model.Centroids[0]),
		NumIterations: model.MaxIterations(),
		Distortion:    model.Distortion(),
		ScaleFactors:  scaleFactors,
		ScaleLog:      scaleLog,
	}
	b, err := json.Marshal(options)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath, b, 0644)
	if err != nil {
		return err
	}
	err = model.PersistToFile(filepath + ".data")
	return err
}
