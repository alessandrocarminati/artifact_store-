MAJOR=$(shell ./verscripts/maj.sh)
MINOR=$(shell ./verscripts/min.sh)
CHASH=$(shell git log --pretty=oneline| head -n1 |cut -d" " -f1)
DIRTY=$(shell ./verscripts/dirty.sh)
all:
	go build -ldflags "-w -X 'main.Version=$(MAJOR)' -X 'main.Build=$(MINOR)' -X 'main.Hash=$(CHASH)' -X 'main.Dirty=$(DIRTY)'"
