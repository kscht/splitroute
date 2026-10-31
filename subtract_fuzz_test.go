package main

import (
	"math/rand"
	"net/netip"
	"testing"
)

// subtractNets на случайных данных: адрес покрыт результатом ⇔ покрыт исходными сетями и не покрыт исключениями.
func TestSubtractRandom(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	rnd := func(base uint32, bits int) netip.Prefix {
		a := base | uint32(r.Intn(1<<16))
		p, _ := netip.AddrFrom4([4]byte{byte(a >> 24), byte(a >> 16), byte(a >> 8), byte(a)}).Prefix(bits)
		return p.Masked()
	}
	for round := 0; round < 300; round++ {
		var org, ex []netip.Prefix
		for i := 0; i < 1+r.Intn(4); i++ {
			org = append(org, rnd(0x0b000000, 16+r.Intn(9)))
		}
		for i := 0; i < r.Intn(6); i++ {
			ex = append(ex, rnd(0x0b000000, 18+r.Intn(15)))
		}
		got := subtractNets(org, ex)
		for k := 0; k < 3000; k++ {
			a := 0x0b000000 | uint32(r.Intn(1<<16))
			ip := netip.AddrFrom4([4]byte{byte(a >> 24), byte(a >> 16), byte(a >> 8), byte(a)})
			in := func(ns []netip.Prefix) bool {
				for _, n := range ns {
					if n.Contains(ip) {
						return true
					}
				}
				return false
			}
			if in(got) != (in(org) && !in(ex)) {
				t.Fatalf("раунд %d, %s: результат %v, org %v, ex %v", round, ip, in(got), in(org), in(ex))
			}
		}
	}
}
