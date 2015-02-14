package batchgcd

import (
	"math/big"
	"runtime"
	"sync"
)

func productTreeLevel(input []big.Int, output []big.Int, wg *sync.WaitGroup, start, step int) {
	for i := start; i < (len(input) / 2); i += step {
		j := i * 2
		output[i].Mul(&input[j], &input[j+1])
	}
	wg.Done()
}

func remainderTreeLevel(tree [][]big.Int, level int, wg *sync.WaitGroup, start, step int) {
	prevLevel := tree[level+1]
	thisLevel := tree[level]
	tmp := &big.Int{}

	for i := start; i < len(thisLevel); i += step {
		x := &thisLevel[i]
		y := &prevLevel[i/2]
		tmp.Mul(x, x)
		x.Rem(y, tmp)
	}
	wg.Done()
}

func remainderTreeFinal(lastLevel, moduli []big.Int, output chan<- Collision, wg *sync.WaitGroup, start, step int) {
	tmp := &big.Int{}

	for i := start; i < len(moduli); i += step {
		modulus := &moduli[i]
		y := &lastLevel[i/2]
		tmp.Mul(modulus, modulus)
		tmp.Rem(y, tmp)
		tmp.Quo(tmp, modulus)
		if tmp.GCD(nil, nil, tmp, modulus).BitLen() != 1 {
			q := &big.Int{}
			q.Quo(modulus, tmp)
			output <- Collision{
				Modulus: modulus,
				P:       tmp,
				Q:       q,
			}
			tmp = &big.Int{}
		}
	}
	wg.Done()
}

func SmootherPartsGCD(moduli []big.Int, output chan<- Collision) {
	defer close(output)
	if len(moduli) < 2 {
		return
	}

	tree := make([][]big.Int, 0)
	for n := (len(moduli) + 1) / 2; ; n = (n + 1) / 2 {
		tree = append(tree, make([]big.Int, n))
		if n == 1 {
			break
		}
	}

	var wg sync.WaitGroup
	nThreads := runtime.NumCPU()

	input := moduli
	for level := 0; level < len(tree); level++ {
		output := tree[level]

		wg.Add(nThreads)
		for i := 0; i < nThreads; i++ {
			go productTreeLevel(input, output, &wg, i, nThreads)
		}

		if (len(input) & 1) == 1 {
			output[len(output)-1] = input[len(input)-1]
		}
		wg.Wait()

		input = output
	}

	for level := len(tree) - 2; level >= 0; level-- {
		wg.Add(nThreads)
		for i := 0; i < nThreads; i++ {
			go remainderTreeLevel(tree, level, &wg, i, nThreads)
		}
		wg.Wait()
	}

	wg.Add(nThreads)
	for i := 0; i < nThreads; i++ {
		go remainderTreeFinal(tree[0], moduli, output, &wg, i, nThreads)
	}
	wg.Wait()
}
