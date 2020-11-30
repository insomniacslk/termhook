package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/insomniacslk/termhook"
)

// default values for flag args
const (
	DefaultPort  = "/dev/ttyUSB0"
	DefaultSpeed = 115200
)

func main() {
	var (
		err   error
		port  = DefaultPort
		speed = DefaultSpeed
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [[device] speed]\n", os.Args[0])
	}
	flag.Parse()
	if flag.Arg(0) != "" {
		port = flag.Arg(0)
	}
	if flag.Arg(1) != "" {
		speed, err = strconv.Atoi(flag.Arg(1))
		if err != nil {
			panic(err)
		}
	}

	hook, err := termhook.NewHook(port, speed, false, nil)
	if err != nil {
		panic(err)
	}
	defer hook.Close()
	if err := hook.Run(); err != nil {
		panic(err)
	}
}
