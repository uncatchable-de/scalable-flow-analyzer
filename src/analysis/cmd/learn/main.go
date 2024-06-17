package main

import (
	"scalable-flow-analyzer/utils"
	"scalable-flow-analyzer/clustering"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/uncatchable-de/goml/cluster"
)

var input = flag.String("i", "", "Input Directory with information files (flow info, session info,...).")
var outputDirectory = flag.String("Output", "", "Output `Directory` where the models are stored. If not set, only statistics will be printed.")
var numClusters = flag.Int("NumClusters", 5, "This parameter defines the number of clusters. If an estimator is used, this defines the maximum number of clusters.")
var numIterations = flag.Int("NumIterations", 100, "This defines the number of iterations.")
var numParallel = flag.Int("NumParallel", 64, "This defines the number of parallel go functions.")
var scaleLog = flag.Bool("ScaleLog", false, "If set, the values are log scaled.")
var evaluationDirectory = flag.String("EvaluationDirectory", "", "If set, the evaluation of the optimal cluster size is written to this `Directory`. When set, NumClusters will be overwritten with a fixed list of clusters to evaluate.")

var evaluateNumClusters = []int{1, 2, 3, 5, 10, 20, 50, 100}

func checkFlags() {
	if *input == "" {
		log.Fatalf("Input argument must not be empty")
	}
	if !utils.DirectoryExists(*input) {
		log.Fatalln("Input argument must either be a directory.", *input)
	}

	if *outputDirectory == "" {
		fmt.Println("No ouput directory specified, so I will not store the result.")
	} else {
		if *evaluationDirectory != "" {
			log.Fatalln("If in evaluation mode, the results of the evaluation will be stored to the provided directory. Using OutputDirectory to store cluster model is then not allowed.")
		}

		if utils.DirectoryExists(*outputDirectory) {
			log.Fatalln("Output directory does already exist. (Re)move it before execution.")
		} else {
			utils.CreateDir(*outputDirectory)
		}
	}
	if *evaluationDirectory != "" {
		if utils.DirectoryExists(*evaluationDirectory) {
			log.Fatalln("Evaluation directory does already exist. (Re)move it before execution.")
		} else {
			utils.CreateDir(*evaluationDirectory)
		}
	}
	if *numClusters <= 0 {
		log.Fatalf("Number of clusters muste be at least 1")
	}
	if *numIterations <= 0 {
		log.Fatalf("Number of iterations muste be at least 1")
	}
	if *numParallel <= 0 {
		log.Fatalf("Number of parallel go functions muste be at least 1")
	}
}

func evaluateOptimalClusterSize(data [][]float64, modelName, filenameWithoutExt string) {
	for _, numCluster := range evaluateNumClusters {
		model := cluster.NewKMeans(numCluster, *numIterations, data)
		default_output := model.Output
		model.Output, _ = os.Open(os.DevNull)
		if err := model.LearnParallel(*numParallel); err != nil {
			log.Fatalln("Error while learning", err)
		}
		model.Output = default_output

		// Write default distortion
		// If the file doesn't exist, create it, or append to the file
		filepath := path.Join(*evaluationDirectory, modelName+"_"+filenameWithoutExt+".csv")

		f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}

		model_distortion := model.Distortion()
		total_distortion := strconv.FormatFloat(model_distortion, 'f', -1, 64)
		avg_distortion := strconv.FormatFloat(model_distortion/float64(len(data)), 'f', -1, 64)

		if _, err := f.Write([]byte(strconv.Itoa(numCluster) + "," + total_distortion + "," + avg_distortion + "\n")); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		model = nil
	}
}

func trainModel(data [][]float64, modelName, filenameWithoutExt string) {
	scaleFactors := clustering.ScaleDatas(data, nil, *scaleLog)

	if *evaluationDirectory != "" {
		evaluateOptimalClusterSize(data, modelName, filenameWithoutExt)
		return
	}
	// Train
	model := cluster.NewKMeans(*numClusters, *numIterations, data)
	default_output := model.Output
	model.Output, _ = os.Open(os.DevNull)
	if err := model.LearnParallel(*numParallel); err != nil {
		log.Fatalln("Error while learning", err)
	}
	model.Output = default_output

	numFlows := make([]int, *numClusters)
	for _, cluster := range model.Guesses() {
		numFlows[cluster] += 1
	}
	fmt.Println("Number of flows per class:", numFlows)
	fmt.Println("Distortion: ", model.Distortion())

	// Save Model
	if *outputDirectory != "" {
		fileSplit := strings.Split(filenameWithoutExt, "_")

		if !utils.DirectoryExists(path.Join(*outputDirectory, modelName)) {
			utils.CreateDir(path.Join(*outputDirectory, modelName))
		}

		outputFile := path.Join(*outputDirectory, modelName, fileSplit[0]+"_"+fileSplit[1]+".json")
		err := clustering.SaveModel(model, scaleFactors, *scaleLog, outputFile)
		if err != nil {
			log.Fatalln("Error while saving model", err)
		}
	}
	fmt.Println("--------------------------")
}

func getFilesFromDir(input string) []string {
	var inputFiles = make([]string, 0)
	files, err := ioutil.ReadDir(input)
	if err != nil {
		log.Fatalln("Error while reading path", input, err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		inputFiles = append(inputFiles, path.Join(input, file.Name()))
	}
	return inputFiles
}

func main() {
	flag.Parse()
	checkFlags()

	inputFiles := getFilesFromDir(path.Join(*input, "rrp"))
	for _, filename := range inputFiles {
		// Load Training data
		filenameWithoutExt := strings.Split(path.Base(filename), ".")[0]
		fmt.Println("--------------------------")
		rrps := clustering.LoadRRPData(filename)
		fmt.Println(*numClusters, "Clusters of", filenameWithoutExt)
		fmt.Println("Use", len(rrps.Rrps), "RRPs in ", *numIterations, "iterations")

		// Transform training data
		var data = make([][]float64, len(rrps.Rrps))
		for i, rrp := range rrps.Rrps {
			data[i] = clustering.GetDataOfRRP(rrp)
		}
		rrps = nil

		trainModel(data, "rrp", filenameWithoutExt)
	}

	inputFiles = getFilesFromDir(path.Join(*input, "flow"))
	for _, filename := range inputFiles {
		// Load Training data
		filenameWithoutExt := strings.Split(path.Base(filename), ".")[0]
		fmt.Println("--------------------------")
		flows := clustering.LoadFlowData(filename)
		fmt.Println(*numClusters, "Clusters of", filenameWithoutExt)
		fmt.Println("Use", len(flows.Flows), "Flows in ", *numIterations, "iterations")

		// Transform training data
		var data = make([][]float64, len(flows.Flows))
		for i, flow := range flows.Flows {
			data[i] = clustering.GetDataOfFlow(flow)
		}
		flows = nil

		trainModel(data, "flow", filenameWithoutExt)
	}

	inputFiles = getFilesFromDir(path.Join(*input, "session"))
	for _, filename := range inputFiles {
		// Load Training data
		filenameWithoutExt := strings.Split(path.Base(filename), ".")[0]
		fmt.Println("--------------------------")
		sessions := clustering.LoadSessionData(filename)
		fmt.Println(*numClusters, "Clusters of", filenameWithoutExt)
		fmt.Println("Use", len(sessions.Sessions), "Sessions in ", *numIterations, "iterations")

		// Transform training data
		var data = make([][]float64, len(sessions.Sessions))
		for i, session := range sessions.Sessions {
			data[i] = clustering.GetDataOfSession(session)
		}
		sessions = nil

		trainModel(data, "session", filenameWithoutExt)
	}

	inputFiles = getFilesFromDir(path.Join(*input, "user"))
	for _, filename := range inputFiles {
		// Load Training data
		filenameWithoutExt := strings.Split(path.Base(filename), ".")[0]
		fmt.Println("--------------------------")
		users := clustering.LoadUserData(filename)
		fmt.Println(*numClusters, "Clusters of", filenameWithoutExt)
		fmt.Println("Use", len(users.Users), "Users in ", *numIterations, "iterations")

		// Transform training data
		var data = make([][]float64, len(users.Users))
		for i, user := range users.Users {
			data[i] = clustering.GetDataOfUser(user)
		}
		users = nil

		trainModel(data, "user", filenameWithoutExt)
	}
}
