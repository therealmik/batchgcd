package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/therealmik/batchgcd"
	"log"
	"math/big"
	"os"
	"runtime"
)

const (
	MODULI_BASE = 16 // Hex
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetOutput(os.Stderr)

	var algorithm string

	flag.StringVar(&algorithm, "algorithm", "pairwise", "Algorithm: <mulaccum|pairwise|smootherparts>")
	flag.Parse()

	var f func([]big.Int, chan<- batchgcd.Collision)

	switch algorithm {
	case "pairwise":
		f = batchgcd.BasicPairwiseGCD
	case "mulaccum":
		f = batchgcd.MulAccumGCD
	case "smootherparts":
		f = batchgcd.SmootherPartsGCD
	default:
		log.Fatal("Invalid algorithm: ", algorithm)
	}

	if len(flag.Args()) == 0 {
		log.Fatal("No files specified")
	}

	moduli := make([]big.Int, 0)
	for _, filename := range flag.Args() {
		log.Print("Loading moduli from ", filename)
		moduli = loadModuli(moduli, filename)
	}

	ch := make(chan batchgcd.Collision, 256)
	log.Print("Executing...")
	go f(moduli, ch)

	for compromised := range ch {
		if !compromised.Test() {
			log.Fatal("Test failed on ", compromised)
		}
		fmt.Println(compromised)
	}
	log.Print("Finished.")
}

func loadModuli(moduli []big.Int, filename string) []big.Int {
	fp, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		m := big.Int{}
		_, ok := m.SetString(scanner.Text(), MODULI_BASE)
		if !ok {
			log.Fatal("Invalid modulus in filename ", filename, ": ", scanner.Text())
		}
		moduli = append(moduli, m)
	}
	return moduli
}
