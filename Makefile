MODULE  = github.com/KorAP/Koral-Mapper
CONFIG  = github.com/KorAP/Koral-Mapper/config
DEV_DIR      = $(shell pwd)
BUILDDATE    = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILDVERSION = $(shell git describe --tags --abbrev=0 2>/dev/null)
BUILDCOMMIT  = $(shell git rev-parse --short HEAD)
GO_LDFLAGS   = -X $(CONFIG).Buildtime=$(BUILDDATE) \
               -X $(CONFIG).Buildhash=$(BUILDCOMMIT) \
               -X $(CONFIG).Version=$(BUILDVERSION) \
               -s \
               -w

BUILDOUT =
ifeq ($(ACTION), build)
  BUILDOUT = -o ./koralmapper
endif


ifeq ($(strip $(BUILDVERSION)), )
  BUILDVERSION = EARLY
endif

build: 
	go build -v \
	         -ldflags "$(GO_LDFLAGS)" \
					--trimpath \
					$(BUILDOUT) \
					./cmd/koralmapper/

update:	## Update all dependencies and clean up the dependency files.
	go get -u all && go mod tidy

test: 
	go test ./...

bench: 	## Run all benchmarks in the code.
	go test -bench=. -benchmem ./... -run=^# -count 5

vet: 	## Run `go vet` on the code.
	go vet ./...

fuzz:
	# go test -fuzz=FuzzTransformEndpoint -fuzztime=1m ./cmd/koralmapper
	go test -fuzz=FuzzParseCfgParam -fuzztime=1m ./cmd/koralmapper

docker:
	docker build -f Dockerfile -t korap/koral-mapper:latest \
		--build-arg BUILDDATE="$(BUILDDATE)" \
		--build-arg BUILDCOMMIT="$(BUILDCOMMIT)" \
		--build-arg BUILDVERSION="$(BUILDVERSION)" \
		.
