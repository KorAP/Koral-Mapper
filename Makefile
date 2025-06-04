MODULE  = github.com/KorAP/KoralPipe-TermMapper
CONFIG  = github.com/KorAP/KoralPipe-TermMapper/config
DEV_DIR      = $(shell pwd)
BUILDDATE    = $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
BUILDVERSION = $(shell git describe --tags --abbrev=0 2>/dev/null)
BUILDCOMMIT  = $(shell git rev-parse --short HEAD)

BUILDOUT =
ifeq ($(ACTION), build)
  BUILDOUT = -o ./termmapper
endif


ifeq ($(strip $(BUILDVERSION)), )
  BUILDVERSION = "EARLY"
endif

build: 
	go build -v \
	         -ldflags "-X $(CONFIG).Buildtime=$(BUILDDATE) \
	                   -X $(CONFIG).Buildhash=$(BUILDCOMMIT) \
			   -X $(CONFIG).Version=$(BUILDVERSION) \
	                   -s \
                       -w" \
					--trimpath \
					$(BUILDOUT) \
					./cmd/termmapper/

update:	## Update all dependencies and clean up the dependency files.
	go get -u all && go mod tidy

test: 
	go test ./...

bench: 	## Run all benchmarks in the code.
	go test -bench=. -benchmem ./... -run=^# -count 5

vet: 	## Run `go vet` on the code.
	go vet ./...

fuzz:
	go test -fuzz=FuzzTransformEndpoint -fuzztime=1m ./cmd/termmapper
