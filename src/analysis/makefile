 # Go parameters
GOCMD=/usr/local/go/bin/go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_NAME=analysis

# Soure: https://stackoverflow.com/questions/2214575/passing-arguments-to-make-run
# If the first argument is "run"...
ifeq (run,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "run"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif


all: test build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v
test: 
	echo "Tests are currently disabled"
	#$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
run: build
	./$(BINARY_NAME) $(RUN_ARGS)
deps:
	$(GOGET) github.com/google/gopacket

.PHONY: run deps clean test build
