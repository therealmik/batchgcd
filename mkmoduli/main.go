package main

import (
	cryptorand "crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"runtime"
	"sync"
)

var dupeprob = flag.Int("prob", 1000, "1/n integers will reuse a modulus")
var nummoduli = flag.Int("num", 100000, "How many moduli to generate")
var bits = flag.Int("bits", 2048, "Bits per RSA modulus")

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.SetOutput(os.Stderr)
	flag.Parse()

	numModuli := *nummoduli
	numThreads := runtime.NumCPU()
	perThread := (numModuli + numThreads - 1) / numThreads
	var wg sync.WaitGroup
	ch := make(chan *big.Int, numThreads)

	for numModuli > 0 {
		if perThread > numModuli {
			perThread = numModuli
		}
		wg.Add(1)
		go genModuli(perThread, ch, &wg)
		numModuli -= perThread
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	for modulus := range ch {
		fmt.Printf("%x\n", modulus)
	}
}

func genModuli(numModuli int, output chan<- *big.Int, wg *sync.WaitGroup) {
	dupChan := make(chan *big.Int, 1)
	var prime1, prime2 *big.Int
	var err error

	for i := 0; i < numModuli; i++ {
		prime1, err = cryptorand.Prime(cryptorand.Reader, (*bits+1)/2)
		if err != nil {
			log.Fatal("Unable to generate random prime")
		}
		if (i % (*dupeprob)) == 1 {
			select {
			case prime2 = <-dupChan:
				output <- new(big.Int).Mul(prime1, prime2)
				continue
			default:
				dupChan <- prime1
			}
		}
		prime2, err = cryptorand.Prime(cryptorand.Reader, (*bits)/2)
		if err != nil {
			log.Fatal("Unable to generate random prime")
		}
		output <- new(big.Int).Mul(prime1, prime2)
	}
	wg.Done()
}
