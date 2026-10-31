package main

import (
	"net/netip"
	"os"
	"testing"
)

func pfx(ss ...string) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range ss {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}

func covers(nets []netip.Prefix, ip string) bool {
	a := netip.MustParseAddr(ip)
	for _, n := range nets {
		if n.Contains(a) {
			return true
		}
	}
	return false
}

// Сеть исключённой организации между двумя сетями маршрутизируемой не должна попасть в агрегат —
// так же, как RU-сеть; без исключения агрегат её захватывает.
func TestExcludedNotCaptured(t *testing.T) {
	org := pfx("1.0.0.0/24", "1.0.2.0/24")
	excluded := pfx("1.0.1.0/24")
	ru := pfx("1.0.64.0/24")

	free := optimizeNets(append([]netip.Prefix{}, org...), append([]netip.Prefix{}, ru...))
	if !covers(free, "1.0.1.10") {
		t.Fatalf("без исключения 1.0.1.0/24 должна захватываться агрегатом, получилось %v", free)
	}

	protected := append(append([]netip.Prefix{}, ru...), excluded...)
	sortNets(protected)
	got := optimizeNets(append([]netip.Prefix{}, org...), protected)
	for _, ip := range []string{"1.0.0.1", "1.0.2.1"} {
		if !covers(got, ip) {
			t.Errorf("сеть организации %s не покрыта: %v", ip, got)
		}
	}
	for _, ip := range []string{"1.0.1.10", "1.0.64.10"} {
		if covers(got, ip) {
			t.Errorf("%s (исключённая организация / RU) захвачен: %v", ip, got)
		}
	}
	sortNets(got)
	orgSorted := append([]netip.Prefix{}, org...)
	sortNets(orgSorted)
	if c := newlyCaptured(excluded, got, addrs(got), orgSorted, addrs(orgSorted)); len(c) != 0 {
		t.Errorf("validate нашла бы захват исключённых: %v", c)
	}
}

// Сеть исключённой организации, вложенная в сеть маршрутизируемой (общий хостинг), остаётся как была:
// исключение запрещает только НОВЫЙ захват агрегацией, исходные сети организаций не урезает (как и с RU).
func TestExcludedInsideOrgKept(t *testing.T) {
	org := pfx("2.0.0.0/22")
	excluded := pfx("2.0.1.0/24")
	got := optimizeNets(append([]netip.Prefix{}, org...), excluded)
	if !covers(got, "2.0.1.10") {
		t.Fatalf("исходная сеть организации урезана: %v", got)
	}
}

func TestOrgListComments(t *testing.T) {
	p := t.TempDir() + "/list.txt"
	if err := writeFile(p, "ByteDance.com  # kimi.ai\n\n# пусто\nbyteplus.com\n"); err != nil {
		t.Fatal(err)
	}
	got, err := loadOrgList(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got["bytedance.com"] || !got["byteplus.com"] {
		t.Fatalf("разбор списка: %v", got)
	}
}

func writeFile(p, s string) error { return os.WriteFile(p, []byte(s), 0o644) }
