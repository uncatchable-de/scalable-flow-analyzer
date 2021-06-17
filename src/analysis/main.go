package main

import (
	"analysis/flows"
	flowMetrics "analysis/metrics/flows"
	standardMetrics "analysis/metrics/standard"
	"analysis/parser"
	"analysis/pool"
	"analysis/reader"
	"analysis/utils"
	"github.com/dustin/go-humanize"
	"github.com/google/gopacket/pcap"

	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"
)

const million = 1000000

const sortingRingBufferSize = 32 * million
const numParser = 16

// Number of parser channels
// Must be maximal numParser, but better if lower to balance load between parsers (e.g. half of numParser)
// If it is too low, the synchronization overhead maybe increases
const numParserChannel = 8

// Flush every x seconds (relative to packet timestamps, not processing time)
const flushRate = int64(40 * time.Second)
const packetStop = 10000 * million

// The two different kinds of metrics one can choose by using the 'flow' flag
var standardMetric *standardMetrics.Metric
var flowMetric *flowMetrics.Metric

var defaultTCPTimeout, _ = time.ParseDuration("5m0s")
var defaultTCPFinTimeout, _ = time.ParseDuration("2s")
var defaultTCPRstTimeout, _ = time.ParseDuration("1s")
var defaultUDPTimeout, _ = time.ParseDuration("5m0s")
var defaultSessionTimeout, _ = time.ParseDuration("10m")

var input = flag.String("i", "", "Path to .pcapng or .pcapng.gz files or to directory with these files (not in combination with --interface)")
var interfaceName = flag.String("interface", "", "Interface name to capture packets from (not in combination with -i)")
var exportDirectory = flag.String("export", "", "Export directory to store the metrics files (Default: metrics)")
var computeFlowMetrics = flag.Bool("flow", true, "Compute flow metrics instead of default metrics (Default: true)")
var tcpFilter = flag.String("tcpFilter", "0-65535", "Filter TCP ports e.g. 0-1023,8080,8443")
var tcpDropIncomplete = flag.Bool("tcpDropIncomplete", false, "If set, the analyzer drops all tcp flows without a SYN packet.")
var dropUnidirectional = flag.Bool("dropUnidirectional", false, "If set, the analyzer will drop all unidirectional traffic. Note, that the reconstruction of TCP flows happens first (if tcpReconstructResponse argument is set).")
var tcpReconstructResponse = flag.Bool("tcpReconstructResponse", false, "If set, the analyzer will try to reconstruct all unidirectional TCP flows, for which only the the packets from the client to the server were captured.")
var udpFilter = flag.String("udpFilter", "0-65535", "Filter UDP ports e.g. 0-1023,8080,8443")
var tcpTimeout = flag.Duration("tcpTimeout", defaultTCPTimeout, "TCP timeout after idle time period")
var tcpFinTimeout = flag.Duration("tcpFinTimeout", defaultTCPFinTimeout, "TCP timeout after a FIN is received")
var tcpRstTimeout = flag.Duration("tcpRstTimeout", defaultTCPRstTimeout, "TCP timeout after a RST is received")
var udpTimeout = flag.Duration("udpTimeout", defaultUDPTimeout, "UDP timeout after idle time period")
var sessionTimeout = flag.Duration("sessionTimeout", defaultSessionTimeout, "Session timeout after idle time period")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var blockprofile = flag.String("blockprofile", "", "write block profile to `file`")
var samplingrate = flag.Float64("sampling", 100, "Sampling rate in percent")
var samplingrateFlows = flag.Int64("samplingFlows", 0, "Sampling rate for flow rate metric in ms. (Default: 0 (average over entire flow))")
var infoDirectory = flag.String("infoDirectory", "", "If a path is specified, the analyzer will output two files for each protocol containing basic rrp, flow, session and user information")
var clusterModelDirectory = flag.String("clusterModelDirectory", "", "If a path is specified, the analyzer will load the clustering models from this path. The models will be used for clustering.")
var statisticTCPReconstruction = flag.Bool("statisticTCPReconstruction", false, "If set, the analyzer will include statistics about the reconstruction in the metric file. This includes sizes of the reconstructed packets as well as speed.")
var computeFlowRRPs = flag.Bool("flowRRPs", false, "If set, the analyzer will compute the size of rrps during the flow based analysis.")
var exportBufferSize = flag.Uint("exportBufferSize", 1000000, "Specified how many serialized flow metrics can be buffered before being written to the flow metrics json file.")

func createMemoryProfile(suffix string) {
	utils.PrintMemUsage()
	if *memprofile != "" {
		f, err := os.Create(suffix + *memprofile)
		if err != nil {
			log.Println("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Println("could not write memory profile: ", err)
		}
	}
}

// CheckFlags will check if the specified flags are valid
func checkFlags() {
	if *input == "" && *interfaceName == "" {
		log.Fatalln("Abort program. Please specify input directoy via -i or an interface.")
	}

	if *input != "" && *interfaceName != "" {
		log.Fatalln("Abort program. Please specify either input directoy via -i or an interface, not both!")
	}

	if *interfaceName != "" && *exportDirectory == "" {
		log.Fatalln("Abort program. Please specify a export Directory if you specify an interface to capture traffic from.")
	} else if *exportDirectory == "" {
		if utils.FileExists(*input) {
			*exportDirectory = path.Join(path.Dir(*input), "metrics")
		} else {
			*exportDirectory = path.Join(*input, "metrics")
		}
	}

	// Create export Directory if it does not exist
	if !utils.DirectoryExists(*exportDirectory) {
		utils.CreateDir(*exportDirectory)
	}

	if *infoDirectory != "" {
		if !utils.DirectoryExists(*infoDirectory) {
			utils.CreateDir(*infoDirectory)
		}
	}

	if *statisticTCPReconstruction && !*tcpReconstructResponse {
		log.Println("statisticTCPReconstruction can only be set in combination with the tcpReconstructResponse flag")
	}
}

func main() {
	flag.Parse()

	checkFlags()

	if *cpuprofile != "" {
		log.Println("Create CPU Profile")
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Println("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Println("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	if *blockprofile != "" {
		log.Println("Create Block Profile")
		f, err := os.Create(*blockprofile)
		if err != nil {
			log.Println("could not create CPU profile: ", err)
		}
		defer f.Close()
		runtime.SetBlockProfileRate(1)
		p := pprof.Lookup("block")
		defer func() {
			err := p.WriteTo(f, 0)
			if err != nil {
				log.Printf("Error writing block profile: %v\n", err)
			}
		}()
	}

	startTime := time.Now()

	// Initialize Pool
	flows.TCPTimeout = tcpTimeout.Nanoseconds()
	flows.TCPRstTimeout = tcpRstTimeout.Nanoseconds()
	flows.TCPFinTimeout = tcpFinTimeout.Nanoseconds()
	flows.UDPTimeout = udpTimeout.Nanoseconds()
	pools := pool.NewPools(utils.ExpandIntegerList(*tcpFilter), utils.ExpandIntegerList(*udpFilter), *tcpDropIncomplete)

	// Initialize Parser
	packetParser := parser.NewParser(pools, sortingRingBufferSize, numParser, *samplingrate, numParserChannel)

	// Initialize Metrics
	if *computeFlowMetrics {
		flowMetric = flowMetrics.NewMetric(*samplingrateFlows, *computeFlowRRPs, *exportBufferSize)
		pools.RegisterMetric(flowMetric)
		go flowMetric.ExportRoutine(*exportDirectory)
	} else {
		standardMetric = standardMetrics.NewMetric(
			sessionTimeout.Nanoseconds(), *infoDirectory,
			*clusterModelDirectory, *dropUnidirectional,
			*tcpReconstructResponse, *statisticTCPReconstruction,
		)
		pools.RegisterMetric(standardMetric)
	}

	// Initialize Reader
	var packetReader = reader.NewPacketReader(pools, packetParser)

	if *input != "" {
		for _, pcapFile := range utils.GetPcapFiles(*input) {
			fmt.Println("Read file: ", pcapFile)
			fmt.Println("Already read", humanize.Comma(packetReader.PacketIdx), "packets")

			packetDataSource, ioHandle, deleteFile, fileName := reader.ReadPcapFile(pcapFile)
			packetStopReached := packetReader.Read(packetStop, flushRate, packetDataSource)

			// Delete uncompressed filed
			if deleteFile {
				_ = os.Remove(fileName)
			}
			_ = ioHandle.Close()
			if packetStopReached {
				break
			}
		}
	} else {
		handle, err := pcap.OpenLive(*interfaceName, 152200, true, pcap.BlockForever)
		if err != nil {
			panic(err)
		}
		packetReader.Read(packetStop, flushRate, handle)
		handle.Close()
	}

	createMemoryProfile("PacketStop")
	secondsAnalyzed := float64(packetReader.LastPacketTimestamp-packetReader.FirstPacketTimestamp) / float64(time.Second)
	fmt.Println("Analyzed\t\t\t", humanize.CommafWithDigits(secondsAnalyzed, 2), "seconds of traffic")

	packetParser.Close()
	fmt.Println("Decoded\t\t\t\t", humanize.Comma(packetReader.PacketIdx), "packets")
	fmt.Println("Time until Parsing Completed:\t", time.Since(startTime))
	pools.PrintStatistics()

	pools.Close()
	fmt.Println("Time until Pool Closed:\t\t", time.Since(startTime))

	if !*computeFlowMetrics {
		standardMetric.ForceFlush()
		createMemoryProfile("MetricFlush")
		fmt.Println("Time until Metric flushed:\t", time.Since(startTime))

		standardMetric.MetricSize.PrintStatistic(false)
		standardMetric.MetricNumRRPairs.PrintStatistic(false)
		standardMetric.MetricInterRequest.PrintStatistic(false)
		standardMetric.MetricNumSessions.PrintStatistic(false)
		standardMetric.MetricInterSessions.PrintStatistic(false)
		standardMetric.MetricNumFlows.PrintStatistic(false)
		standardMetric.MetricInterFlowTimes.PrintStatistic(false)
		standardMetric.ReqResIdentifier.PrintStatistic(false)
	}

	if *computeFlowMetrics {
		flowMetric.Flush()
		flowMetric.Wait()
	} else {
		fmt.Println("Time until export start:", time.Since(startTime))
		standardMetric.Export(*exportDirectory)
		fmt.Println("Time until export finished:", time.Since(startTime))
	}
}
