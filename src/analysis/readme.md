# Installation
1. gopacket requires libpcap on Linux. For Ubuntu use the following command: `sudo apt-get install libpcap-dev`
2. Set GOPATH to the root directory of this repository (e.g. ~/l4-traffic-analyzer).
3. Use `go get ./...` in analysis folder to install all dependencies.

# Running
Use `make -- run -h` to see all available command line options.