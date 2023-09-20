package cidrcalc

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"net/netip"
	"sort"
)

type IPRange struct {
	start, stop uint64
}

func FromCIDR(cidr string) (*IPRange, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, err
	}
	if !prefix.IsValid() {
		return nil, fmt.Errorf("invalid %v from %s", prefix, cidr)
	}
	if !prefix.Addr().Is4() {
		return nil, fmt.Errorf("prefix %v from %s is not v4", prefix, cidr)
	}

	start := uint64(binary.BigEndian.Uint32(prefix.Masked().Addr().AsSlice()))
	stop := start + 1<<(32-prefix.Bits())
	return &IPRange{
		start: start,
		stop:  stop,
	}, nil
}

func toCIDR(start, step uint64) string {
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, uint32(start))
	addr, ok := netip.AddrFromSlice(b4)
	if !ok {
		panic(fmt.Sprintf("bad start+step for CIDR: %d+%d", start, step))
	}
	return fmt.Sprintf("%s/%d", addr, 32-bits.TrailingZeros64(step))
}

func (a *IPRange) CIDRs() []string {
	start := a.start
	cidrs := []string{}
	for start < a.stop {
		step := start & ^(start - 1)
		if start == 0 {
			step = 1 << 32
		}
		for start+step > a.stop {
			step >>= 1
		}
		cidrs = append(cidrs, toCIDR(start, step))
		start += step
	}
	return cidrs
}

type endpoint struct {
	ip uint64

	// start = 1; stop = -1 "by default" (not inverted)
	ss int
}

type byIP []endpoint

func (a byIP) Len() int {
	return len(a)
}

func (a byIP) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a byIP) Less(i, j int) bool {
	if a[i].ip < a[j].ip {
		return true
	} else if a[i].ip > a[j].ip {
		return false
	} else {
		// process "start" first, or we may get unwanted empty gaps.
		return a[i].ss > a[j].ss
	}
}

type IPRanges struct {
	ranges     []IPRange
	simplified bool
}

func FromCIDRs(cidrs []string) (*IPRanges, error) {
	ranges := []IPRange{}
	for _, cidr := range cidrs {
		ipRange, err := FromCIDR(cidr)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, *ipRange)
	}
	return &IPRanges{
		ranges:     ranges,
		simplified: len(cidrs) < 2,
	}, nil
}

func (a *IPRanges) CIDRs() []string {
	cidrs := []string{}
	for _, ipRange := range a.ranges {
		cidrs = append(cidrs, ipRange.CIDRs()...)
	}
	return cidrs
}

func (a *IPRanges) Add(b *IPRanges) *IPRanges {
	return &IPRanges{
		ranges:     append(a.ranges, b.ranges...),
		simplified: false,
	}
}

func endpointsToRanges(endpoints []endpoint) []IPRange {
	cnt := 0
	var start uint64
	ranges := []IPRange{}
	for _, ep := range endpoints {
		newCnt := cnt + ep.ss
		if cnt == 0 && newCnt == 1 {
			start = ep.ip
		} else if cnt == 1 && newCnt == 0 && start != ep.ip {
			ranges = append(ranges, IPRange{
				start: start,
				stop:  ep.ip,
			})
		}
		cnt = newCnt
	}
	if cnt != 0 {
		panic(fmt.Sprintf("unclosed endpoints: %v", endpoints))
	}
	return ranges
}

func (a *IPRanges) Simplify() *IPRanges {
	if a.simplified {
		return a
	}
	endpoints := a.toEndpoints(false)
	sort.Sort(byIP(endpoints))
	return &IPRanges{
		ranges:     endpointsToRanges(endpoints),
		simplified: true,
	}
}

func (a *IPRanges) Subtract(b *IPRanges) *IPRanges {
	a = a.Simplify()
	aEndpoints := a.toEndpoints(false)
	bEndpoints := b.toEndpoints(true)
	endpoints := append(aEndpoints, bEndpoints...)
	sort.Sort(byIP(endpoints))
	return &IPRanges{
		ranges:     endpointsToRanges(endpoints),
		simplified: true,
	}
}

func (a *IPRanges) toEndpoints(invert bool) []endpoint {
	start, stop := 1, -1
	if invert {
		start, stop = -1, 1
	}
	endpoints := []endpoint{}
	for _, ipRange := range a.ranges {
		endpoints = append(endpoints, endpoint{
			ip: ipRange.start,
			ss: start,
		}, endpoint{
			ip: ipRange.stop,
			ss: stop,
		})
	}
	return endpoints
}
