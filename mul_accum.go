package batchgcd

import (
	"github.com/ncw/gmp"
	"runtime"
	"sync"
)

type gcdTask struct {
	accum *gmp.Int
	i     int
}

// This performs the GCD of the product of all previous moduli with the current one.
// This uses around double the memory (minus quite a lot of overhead), and identifies
// problematic input in O(n) time, but has to do another O(n) scan for each collision
// to figure get the private key back.
// If there are no collisions, this algorithm isn't parallel at all.
// If we get a GCD that is the same as the modulus, we do a manual scan for either colliding Q or identical moduli
// If we get a GCD lower than the modulus, we have one private key, then do a manual scan for others.
func MulAccumGCD(moduli []*gmp.Int, collisions chan<- Collision) {
	accum := gmp.NewInt(1)
	var wg sync.WaitGroup
	nThreads := runtime.NumCPU()

	gcdChan := make(chan gcdTask, nThreads*2)
	wg.Add(nThreads)
	for i := 0; i < nThreads; i++ {
		go gcdProc(gcdChan, moduli, collisions, &wg)
	}

	for i := 0; i < len(moduli); i++ {
		gcdChan <- gcdTask{accum, i}
		accum = gmp.NewInt(0).Mul(accum, moduli[i])
	}
	close(gcdChan)
	wg.Wait()
	close(collisions)
}

func gcdProc(gcdChan <-chan gcdTask, moduli []*gmp.Int, collisions chan<- Collision, wg *sync.WaitGroup) {
	gcd := gmp.NewInt(0)

	for task := range gcdChan {
		modulus := moduli[task.i]
		gcd.GCD(nil, nil, task.accum, modulus)
		if gcd.BitLen() == 1 {
			continue
		}
		wg.Add(1)
		if gcd.Cmp(modulus) == 0 {
			go findGCD(wg, moduli, task.i, collisions)
		} else {
			go findDivisors(wg, moduli, task.i, gcd, collisions)
			gcd = gmp.NewInt(0)
		}
	}
	wg.Done()
}

// Tests the candidate (i) against all other moduli
func findDivisors(wg *sync.WaitGroup, moduli []*gmp.Int, i int, gcd *gmp.Int, collisions chan<- Collision) {
	m := moduli[i]
	q := gmp.NewInt(0)
	r := gmp.NewInt(0)

	q.Quo(m, gcd)
	collisions <- Collision{
		Modulus: m,
		P:       gcd,
		Q:       q,
	}
	q = gmp.NewInt(0)

	for j := 0; j < i; j++ {
		n := moduli[j]
		q.QuoRem(n, gcd, r)
		if r.BitLen() == 0 {
			collisions <- Collision{
				Modulus: n,
				P:       gcd,
				Q:       q,
			}
		}
		q = gmp.NewInt(0)
	}
	wg.Done()
}

func findGCD(wg *sync.WaitGroup, moduli []*gmp.Int, i int, collisions chan<- Collision) {
	m := moduli[i]
	q := gmp.NewInt(0)
	gcd := gmp.NewInt(0)

	for j := 0; j < i; j++ {
		n := moduli[j]

		if gcd.GCD(nil, nil, m, n).BitLen() != 1 {
			q.Quo(m, gcd)
			collisions <- Collision{
				Modulus: m,
				P:       gcd,
				Q:       q,
			}
			q = gmp.NewInt(0)

			q.Quo(n, gcd)
			collisions <- Collision{
				Modulus: n,
				P:       gcd,
				Q:       q,
			}
			q = gmp.NewInt(0)

			gcd = gmp.NewInt(0)
		}
	}
	wg.Done()
}
