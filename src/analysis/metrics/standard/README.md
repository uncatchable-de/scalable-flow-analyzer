# Metric Directory

This directory contains all metrics. They are all managed by the central `metric.go` file. This struct, registers the metrics to the hooks, as well as to the lists for automated inclusion in the json exports. The export is then done by the `export.go` file.

The three files `IntMetric.go`, `IntMetricUnivariate` and `IntMetricBivariate` contains threadsafe implementation of a simple counter (e.g to count the number of packets), a univariate distribution (e.g. to count the different sizes a packet can have) and a bivariate distribution (e.g. to count the size of a packet, depending on the packet number within a flow).

The file `utils.go` contains some helpful utility functions to handle application protocol identification.

The `ReqResIdentifier.go` and `SessionIdentifier.go` contains the logic to identify Request/response pairs within a flow, as well as sessions. The first file, also contains the logic to reconstruct flows, based on ACK Nr analysis, in case they contain only unidirectional traffic.

The `ClusterController.go` is used to write out flow information (e.g. average size) to a protobuf file. This file can then be used by the clustering project to train models to identify clusters. The `ClusterController.go` is also responsible to load the learned models and then to identify the cluster of the model.