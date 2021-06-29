package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

var (
	inst   = os.ExpandEnv("$PROTOBUF")
	protoc = inst + "/bin/protoc"
	plugin = inst + "/bin/protoc-gen-go"
	lib    = inst + "/lib"

	inc = flag.String("i", ".", "protobuf include path")
	out = flag.String("o", ".", "protobuf output path")
)

func init() {
	flag.Parse()
}

func sanitize() {
	for _, f := range []string{protoc, plugin, lib, *inc, *out} {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			log.Fatalf("%v", err)
		}
	}
}

func main() {
	sanitize()

	ldlp := os.Getenv("LD_LIBRARY_PATH")
	os.Setenv("LD_LIBRARY_PATH", fmt.Sprintf("%s:%s", ldlp, lib))

	src := flag.Args()
	args := append([]string{"-I", *inc, "--plugin=" + plugin, "--go_out=plugins=grpc:" + *out}, src...)
	cmd := exec.Command(protoc, args...)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
