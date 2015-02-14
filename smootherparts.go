package batchgcd

import (
	"math/big"
	"runtime"
	"sync"
)

// Multiply sets of two adjacent inputs, placing into a single output
func productTreeLevel(input []big.Int, output []big.Int, wg *sync.WaitGroup, start, step int) {
	for i := start; i < (len(input) / 2); i += step {
		j := i * 2
		output[i].Mul(&input[j], &input[j+1])
	}
	wg.Done()
}

// For each productTree node 'x', and remainderTree parent 'y', compute y%(x*x)
func remainderTreeLevel(tree [][]big.Int, level int, wg *sync.WaitGroup, start, step int) {
	prevLevel := tree[level+1]
	thisLevel := tree[level]
	tmp := new(big.Int)

	for i := start; i < len(thisLevel); i += step {
		x := &thisLevel[i]
		y := &prevLevel[i/2]
		tmp.Mul(x, x)
		x.Rem(y, tmp)
	}
	wg.Done()
}

// For each input modulus 'x' and remainderTree parent 'y', compute z = (y%(x*x))/x; gcd(z, x)
func remainderTreeFinal(lastLevel, moduli []big.Int, output chan<- Collision, wg *sync.WaitGroup, start, step int) {
	tmp := new(big.Int)

	for i := start; i < len(moduli); i += step {
		modulus := &moduli[i]
		y := &lastLevel[i/2]
		tmp.Mul(modulus, modulus)
		tmp.Rem(y, tmp)
		tmp.Quo(tmp, modulus)
		if tmp.GCD(nil, nil, tmp, modulus).BitLen() != 1 {
			q := new(big.Int)
			q.Quo(modulus, tmp)
			output <- Collision{
				Modulus: modulus,
				P:       tmp,
				Q:       q,
			}
			tmp = new(big.Int)
		}
	}
	wg.Done()
}

// Implementation of D.J. Bernstein's "How to find smooth parts of integers"
// http://cr.yp.to/papers.html#smoothparts
func SmootherPartsGCD(moduli []big.Int, output chan<- Collision) {
	defer close(output)
	if len(moduli) < 2 {
		return
	}

	// Create a tree, each level being ceil(n/2) in size, where
	// n is the size of the level below.  The top level is size 1,
	// the bottom level is the input moduli
	tree := make([][]big.Int, 0)
	for n := (len(moduli) + 1) / 2; ; n = (n + 1) / 2 {
		tree = append(tree, make([]big.Int, n))
		if n == 1 {
			break
		}
	}

	var wg sync.WaitGroup
	nThreads := runtime.NumCPU()

	// Create a product tree
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

	// Create a remainder tree
	for level := len(tree) - 2; level >= 0; level-- {
		wg.Add(nThreads)
		for i := 0; i < nThreads; i++ {
			go remainderTreeLevel(tree, level, &wg, i, nThreads)
		}
		wg.Wait()
	}

	// The final round is a little special, partially because of how my code
	// handles it's input, but also because there are extra steps.
	wg.Add(nThreads)
	for i := 0; i < nThreads; i++ {
		go remainderTreeFinal(tree[0], moduli, output, &wg, i, nThreads)
	}
	wg.Wait()
}
