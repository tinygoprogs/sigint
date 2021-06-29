TOOLS = ./tools/wifi-to-sqlite.go
TOOL_BINS = $(foreach tool,$(TOOLS),$(tool:.go=))
define gobuild
$(1:.go=) : $(1) proto
	go build -o $(1:.go=) $(1)
endef
$(foreach tool,$(TOOLS),$(eval $(call gobuild,$(tool))))
.PHONY: clean test proto tools
proto: $(wildcard ./proto/*.proto)
	go generate
clean:
	go clean
	rm -f *.pb.go local/test.db $(TOOL_BINS)
test: proto
	git lfs checkout # needed for testdata/*.cap files
	go test && cd local && go test
