package batchgcd

import (
	"math/big"
	"runtime"
	"sync"
)

func BasicPairwiseGCD(moduli []big.Int, collisions chan<- Collision) {
	var wg sync.WaitGroup
	nThreads := runtime.NumCPU()

	wg.Add(nThreads)
	for i := 0; i < nThreads; i++ {
		go pairwiseThread(i, nThreads, &wg, moduli, collisions)
	}
	wg.Wait()
	close(collisions)
}

func pairwiseThread(start, step int, wg *sync.WaitGroup, moduli []big.Int, collisions chan<- Collision) {
	gcd := new(big.Int)

	for i := start; i < len(moduli); i += step {
		for j := i + 1; j < len(moduli); j++ {
			m1 := &moduli[i]
			m2 := &moduli[j]
			if m1.Cmp(m2) == 0 {
				collisions <- Collision{Modulus: m1}
			} else if gcd.GCD(nil, nil, m1, m2).BitLen() != 1 { // There's only one number with a BitLen of 1
				q := new(big.Int)
				q.Quo(m1, gcd)
				collisions <- Collision{
					Modulus: m1,
					P:       gcd,
					Q:       q,
				}

				q = new(big.Int)
				q.Quo(m2, gcd)
				collisions <- Collision{
					Modulus: m2,
					P:       gcd,
					Q:       q,
				}

				gcd = new(big.Int) // Old gcd var can't be overwritten
			}
		}
	}
	wg.Done()
}
