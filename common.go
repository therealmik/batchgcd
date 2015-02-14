package batchgcd

import (
	"fmt"
	"math/big"
)

type Collision struct {
	Modulus *big.Int
	P       *big.Int
	Q       *big.Int
}

func (x Collision) HavePrivate() bool {
	return x.P != nil || x.Q != nil
}

func (x Collision) String() string {
	if x.HavePrivate() {
		if x.P.Cmp(x.Q) < 0 {
			return fmt.Sprintf("COLLISION: N=%x P=%x Q=%x", x.Modulus, x.P, x.Q)
		} else {
			return fmt.Sprintf("COLLISION: N=%x P=%x Q=%x", x.Modulus, x.Q, x.P)
		}
	} else {
		return fmt.Sprintf("DUPLICATE: %x", x.Modulus)
	}
}

func (x Collision) Test() bool {
	if !x.HavePrivate() {
		return true
	}
	n := big.NewInt(0)
	n.Mul(x.P, x.Q)
	return n.Cmp(x.Modulus) == 0
}
