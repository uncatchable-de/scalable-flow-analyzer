# ScalableFlow Analysis in Go 

A scalable network flow analyzer written in Go as presented in our paper:

Simon Bauer, Benedikt Jaeger, Fabian Helfert, Philippe Barias, and Georg Carle. "On the Evolution of Internet Flow Characteristics," in The Applied Networking Research Workshop 2021 (ANRW ’21), Jul. 2021

## Setup the Analyzer

1. Install Go (tested with Go 1.12.4)
2. `apt-get install libpcap-dev`
2. Clone this repo 
3. export GOROOT to your Go directory `export GOROOT=/usr/local/go`
4. `export GOPATH=/$path/scalable-flow-analyzer/`
5. `cd /$path/scalable-flow-analyzer/src/analysis`
6. `go get`
7. `make`

## Usage 

* `./analysis --help`
* `./analysis -i $path-to-PCAP --flow -tcpDropIncomplete -export $path-to-results`


