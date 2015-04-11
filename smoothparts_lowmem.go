package batchgcd

// NOTE: This code was written with fastgcd available at https://factorable.net/
// as a reference, which was written by Nadia Heninger and J. Alex Halderman.
// I have put a substantial amount of my own design into this, and they do not
// claim it as a derivative work.
// I thank them for their original code and paper.

import (
	"encoding/gob"
	"fmt"
	"github.com/ncw/gmp"
	"io"
	"log"
	"os"
	"time"
)

func tmpfileReadWriter(inChan chan *gmp.Int, outChan chan *gmp.Int, prefix string, typ string, level int) {
	filename := fmt.Sprintf("%s-%s-%d", typ, prefix, level)
	tmpFile, err := os.Create(filename)
	if err != nil {
		log.Panic(err)
	}

	var writeCount uint64
	enc := gob.NewEncoder(tmpFile)
	for inData := range inChan {
		writeCount += 1
		if e := enc.Encode(inData); err != nil {
			log.Panic(e)
		}
	}

	if newOffset, e := tmpFile.Seek(0, 0); e != nil || newOffset != 0 {
		log.Panic(e)
	}

	var readCount uint64
	m := gmp.NewInt(0)
	dec := gob.NewDecoder(tmpFile)
	var e error
	for e = dec.Decode(m); e == nil; e = dec.Decode(m) {
		readCount += 1
		outChan <- m
		m = gmp.NewInt(0)
	}

	if e != io.EOF {
		log.Panic(e)
	}
	if writeCount != readCount {
		log.Panicf("Didn't write as many as we read: write=%v read=%v", writeCount, readCount)
	}
	close(outChan)
	// tmpFile.Truncate(0);
}

// Multiply sets of two adjacent inputs, placing into a single output
func lowmemProductTreeLevel(prefix string, level int, input chan *gmp.Int, channels []chan *gmp.Int, finalOutput chan<- Collision) {
	resultChan := make(chan *gmp.Int, 0)
	defer close(resultChan)

	hold := <-input
	m, ok := <-input
	if !ok {
		go lowmemRemainderTreeLevel(resultChan, channels, finalOutput)
		resultChan <- hold
		return
	}

	fileWriteChan := make(chan *gmp.Int, 0)
	fileReadChan := make(chan *gmp.Int, 0)
	go tmpfileReadWriter(fileWriteChan, fileReadChan, prefix, "product", level)
	fileWriteChan <- hold
	fileWriteChan <- m

	channels = append(channels, fileReadChan)
	go lowmemProductTreeLevel(prefix, level+1, resultChan, channels, finalOutput)
	resultChan <- gmp.NewInt(0).Mul(hold, m)
	hold = nil

	for m = range input {
		fileWriteChan <- m
		if hold != nil {
			resultChan <- gmp.NewInt(0).Mul(hold, m)
			hold = nil
		} else {
			hold = m
		}
	}

	close(fileWriteChan)

	if hold != nil {
		resultChan <- hold
	}
}

// For each productTree node 'x', and remainderTree parent 'y', compute y%(x*x)
func lowmemRemainderTreeLevel(input chan *gmp.Int, productTree []chan *gmp.Int, finalOutput chan<- Collision) {
	tmp := gmp.NewInt(0)

	products := productTree[len(productTree)-1]
	productTree = productTree[:len(productTree)-1]
	output := make(chan *gmp.Int, 0)
	defer close(output)

	if len(productTree) == 0 {
		lowmemRemainderTreeFinal(input, products, finalOutput)
		return
	} else {
		go lowmemRemainderTreeLevel(output, productTree, finalOutput)
	}

	for y := range input {
		x, ok := <-products
		if !ok {
			log.Panicf("Expecting more products")
		}
		tmp.Mul(x, x)
		x.Rem(y, tmp)
		output <- x

		x, ok = <-products
		if ok {
			tmp.Mul(x, x)
			x.Rem(y, tmp)
			output <- x
		}
	}
}

// For each input modulus 'x' and remainderTree parent 'y', compute z = (y%(x*x))/x; gcd(z, x)
func lowmemRemainderTreeFinal(input, moduli chan *gmp.Int, output chan<- Collision) {
	defer close(output)
	tmp := new(gmp.Int)

	for y := range input {
		for i := 0; i < 2; i++ {
			modulus, ok := <-moduli
			if !ok {
				log.Print("Odd number of moduli? (should only see this once)")
				continue
			}
			tmp.Mul(modulus, modulus)
			tmp.Rem(y, tmp)
			tmp.Quo(tmp, modulus)
			if tmp.GCD(nil, nil, tmp, modulus).BitLen() != 1 {
				q := gmp.NewInt(0).Quo(modulus, tmp)
				output <- Collision{
					Modulus: modulus,
					P:       tmp,
					Q:       q,
				}
				tmp = gmp.NewInt(0)
			}
		}
	}
}

// Implementation of D.J. Bernstein's "How to find smooth parts of integers"
// http://cr.yp.to/papers.html#smoothparts
func LowMemSmoothPartsGCD(moduli chan *gmp.Int, output chan<- Collision) {
	prefix := time.Now().Format(time.RFC3339Nano)
	go lowmemProductTreeLevel(prefix, 1, moduli, nil, output)
}
