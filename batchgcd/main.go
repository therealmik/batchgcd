package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ncw/gmp"
	"github.com/therealmik/batchgcd"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
)

const (
	MODULI_BASE = 16 // Hex
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var algorithmName = flag.String("algorithm", "smoothparts", "mulaccum|pairwise|smoothparts")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetOutput(os.Stderr)
	flag.Parse()

	var f func([]*gmp.Int, chan<- batchgcd.Collision)

	switch *algorithmName {
	case "pairwise":
		f = batchgcd.BasicPairwiseGCD
	case "mulaccum":
		f = batchgcd.MulAccumGCD
	case "smoothparts":
		f = batchgcd.SmoothPartsGCD
	default:
		log.Fatal("Invalid algorithm: ", *algorithmName)
	}

	if len(flag.Args()) == 0 {
		log.Fatal("No files specified")
	}

	moduli := make([]*gmp.Int, 0)
	for _, filename := range flag.Args() {
		log.Print("Loading moduli from ", filename)
		moduli = loadModuli(moduli, filename)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ch := make(chan batchgcd.Collision, 256)
	log.Print("Executing...")
	go f(moduli, ch)

	for compromised := range uniqifyCollisions(ch) {
		if !compromised.Test() {
			log.Fatal("Test failed on ", compromised)
		}
		fmt.Println(compromised.Csv())
	}
	log.Print("Finished.")
}

func loadModuli(moduli []*gmp.Int, filename string) []*gmp.Int {
	fp, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		m := gmp.NewInt(0)

		splitModuli := strings.SplitN(scanner.Text(), ",", 2)
		s := splitModuli[0] // Accept CSV moduli, so long as modulus is first column

		// Dedupe
		if _, ok := seen[s]; ok {
			continue
		} else {
			seen[s] = struct{}{}
		}

		if _, ok := m.SetString(s, MODULI_BASE); !ok {
			log.Fatal("Invalid modulus in filename ", filename, ": ", scanner.Text())
		}
		moduli = append(moduli, m)
	}
	return moduli
}

func uniqifyCollisions(in <-chan batchgcd.Collision) chan batchgcd.Collision {
	out := make(chan batchgcd.Collision)
	go uniqifyProc(in, out)
	return out
}

func uniqifyProc(in <-chan batchgcd.Collision, out chan<- batchgcd.Collision) {
	seen := make(map[string]struct{})
	for c := range in {
		s := c.String()
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out <- c
	}
	close(out)
}
