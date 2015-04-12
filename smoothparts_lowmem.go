package batchgcd

// NOTE: This code was written with fastgcd available at https://factorable.net/
// as a reference, which was written by Nadia Heninger and J. Alex Halderman.
// I have put a substantial amount of my own design into this, and they do not
// claim it as a derivative work.
// I thank them for their original code and paper.

import (
	"github.com/ncw/gmp"
	"io"
	"io/ioutil"
	"log"
	"runtime"
	"syscall"
)

type fileChannels struct {
	writeChan chan *gmp.Int
	readChan  chan *gmp.Int
	producing bool
}

func (fch *fileChannels) StartProducing() {
	if fch.producing {
		return
	}
	close(fch.writeChan)
	fch.producing = true
}

func encodeLength(buf []byte, length int) {
	buf[0] = byte(length >> 56)
	buf[1] = byte(length >> 48)
	buf[2] = byte(length >> 40)
	buf[3] = byte(length >> 32)
	buf[4] = byte(length >> 24)
	buf[5] = byte(length >> 16)
	buf[6] = byte(length >> 8)
	buf[7] = byte(length)
}

func decodeLength(buf []byte) int {
	var ret int

	ret |= int(buf[0]) << 56
	ret |= int(buf[1]) << 48
	ret |= int(buf[2]) << 40
	ret |= int(buf[3]) << 32
	ret |= int(buf[4]) << 24
	ret |= int(buf[5]) << 16
	ret |= int(buf[6]) << 8
	ret |= int(buf[7])

	return ret
}

func tmpfileReadWriter(ch fileChannels) {
	tmpFile, err := ioutil.TempFile(".", "product")
	if err != nil {
		log.Panic(err)
	}

	length := make([]byte, 8)

	var writeCount uint64
	for inData := range ch.writeChan {
		buf := inData.Bytes()
		encodeLength(length, len(buf))
		if _, err := tmpFile.Write(length); err != nil {
			log.Panic(err)
		}
		if _, err := tmpFile.Write(buf); err != nil {
			log.Panic(err)
		}
		writeCount += 1
	}

	if newOffset, e := tmpFile.Seek(0, 0); e != nil || newOffset != 0 {
		log.Panic(e)
	}

	var readCount uint64
	m := new(gmp.Int)
	for {
		if _, e := io.ReadFull(tmpFile, length); e != nil {
			if e == io.EOF {
				break
			}
			log.Panic(e)
		}
		buf := make([]byte, decodeLength(length))
		if _, e := io.ReadFull(tmpFile, buf); e != nil {
			log.Panic(e)
		}
		readCount += 1
		ch.readChan <- m.SetBytes(buf)
		m = new(gmp.Int)
	}

	if writeCount != readCount {
		log.Panicf("Didn't write as many as we read: write=%v read=%v", writeCount, readCount)
	}
	close(ch.readChan)
	syscall.Unlink(tmpFile.Name())
	// tmpFile.Truncate(0);
}

// Multiply sets of two adjacent inputs, placing into a single output
func lowmemProductTreeLevel(input chan *gmp.Int, channels []fileChannels, finalOutput chan<- Collision) {
	resultChan := make(chan *gmp.Int, 2)
	defer close(resultChan)

	hold := <-input
	m, ok := <-input
	if !ok {
		go lowmemRemainderTreeLevel(resultChan, channels, finalOutput)
		resultChan <- hold
		return
	}

	fileChans := fileChannels{
		writeChan: make(chan *gmp.Int, 2),
		readChan:  make(chan *gmp.Int, 1),
	}
	go tmpfileReadWriter(fileChans)
	fileChans.writeChan <- hold
	fileChans.writeChan <- m

	channels = append(channels, fileChans)
	go lowmemProductTreeLevel(resultChan, channels, finalOutput)
	resultChan <- new(gmp.Int).Mul(hold, m)
	hold = nil

	for m = range input {
		fileChans.writeChan <- m
		if hold != nil {
			resultChan <- new(gmp.Int).Mul(hold, m)
			hold = nil
		} else {
			hold = m
		}
	}

	if hold != nil {
		resultChan <- hold
	}
}

// For each productTree node 'x', and remainderTree parent 'y', compute y%(x*x)
func lowmemRemainderTreeLevel(input chan *gmp.Int, productTree []fileChannels, finalOutput chan<- Collision) {
	tmp := new(gmp.Int)
	runtime.GC()
	defer runtime.GC()

	ch := productTree[len(productTree)-1]
	productTree = productTree[:len(productTree)-1]

	// We close the fileWriteChannel here so it kicks off reading now, instead of starting too early
	products := ch.readChan

	output := make(chan *gmp.Int, 2)
	defer close(output)

	if len(productTree) == 0 {
		ch.StartProducing()
		lowmemRemainderTreeFinal(input, products, finalOutput)
		return
	} else {
		go lowmemRemainderTreeLevel(output, productTree, finalOutput)
	}

	for y := range input {
		ch.StartProducing()
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
				q := new(gmp.Int).Quo(modulus, tmp)
				output <- Collision{
					Modulus: modulus,
					P:       tmp,
					Q:       q,
				}
				tmp = new(gmp.Int)
			}
		}
	}
}

// Implementation of D.J. Bernstein's "How to find smooth parts of integers"
// http://cr.yp.to/papers.html#smoothparts
func LowMemSmoothPartsGCD(moduli chan *gmp.Int, output chan<- Collision) {
	go lowmemProductTreeLevel(moduli, nil, output)
}
