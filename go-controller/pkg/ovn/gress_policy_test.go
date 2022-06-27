package ovn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	knet "k8s.io/api/networking/v1"
)

func TestGetMatchFromIPBlock(t *testing.T) {
	testcases := []struct {
		desc       string
		ipBlocks   []*knet.IPBlock
		lportMatch string
		l4Match    string
		expected   []string
	}{
		{
			desc: "IPv4 only no except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR: "0.0.0.0/0",
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected:   []string{"ip4.src == 0.0.0.0/0 && input && fake"},
		},
		{
			desc: "multiple IPv4 only no except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR: "0.0.0.0/0",
				},
				{
					CIDR: "10.1.0.0/16",
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected: []string{"ip4.src == 0.0.0.0/0 && input && fake",
				"ip4.src == 10.1.0.0/16 && input && fake"},
		},
		{
			desc: "IPv6 only no except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR: "fd00:10:244:3::49/32",
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected:   []string{"ip6.src == fd00:10:244:3::49/32 && input && fake"},
		},
		{
			desc: "mixed IPv4 and IPv6  no except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR: "::/0",
				},
				{
					CIDR: "0.0.0.0/0",
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected: []string{"ip6.src == ::/0 && input && fake",
				"ip4.src == 0.0.0.0/0 && input && fake"},
		},
		{
			desc: "IPv4 only with except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR:   "0.0.0.0/0",
					Except: []string{"10.1.0.0/16"},
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected:   []string{"ip4.src == 0.0.0.0/0 && ip4.src != {10.1.0.0/16} && input && fake"},
		},
		{
			desc: "multiple IPv4 with except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR:   "0.0.0.0/0",
					Except: []string{"10.1.0.0/16"},
				},
				{
					CIDR: "10.1.0.0/16",
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected: []string{"ip4.src == 0.0.0.0/0 && ip4.src != {10.1.0.0/16} && input && fake",
				"ip4.src == 10.1.0.0/16 && input && fake"},
		},
		{
			desc: "IPv4 with IPv4 except",
			ipBlocks: []*knet.IPBlock{
				{
					CIDR:   "0.0.0.0/0",
					Except: []string{"10.1.0.0/16"},
				},
			},
			lportMatch: "fake",
			l4Match:    "input",
			expected:   []string{"ip4.src == 0.0.0.0/0 && ip4.src != {10.1.0.0/16} && input && fake"},
		},
	}

	for _, tc := range testcases {
		gressPolicy := newGressPolicy(knet.PolicyTypeIngress, 5, "testing", "test")
		for _, ipBlock := range tc.ipBlocks {
			gressPolicy.addIPBlock(ipBlock)
		}
		output := gressPolicy.getMatchFromIPBlock(tc.lportMatch, tc.l4Match)
		assert.Equal(t, tc.expected, output)
	}
}

func TestGetL4Match(t *testing.T) {
	testcases := []struct {
		desc     string
		protocol string
		port     int32
		endPort  int32
		expected string
	}{
		{
			desc:     "unsupported protocol",
			protocol: "kube",
			port:     0,
			endPort:  0,
			expected: "",
		},
		{
			desc:     "valid protocol with no endport specified",
			protocol: "TCP",
			port:     300,
			endPort:  0,
			expected: "tcp && tcp.dst==300",
		},
		{
			desc:     "valid protocol with endport specified",
			protocol: "TCP",
			port:     300,
			endPort:  310,
			expected: "tcp && 300<=tcp.dst<=310",
		},
		{
			desc:     "valid protocol with no ports specified",
			protocol: "TCP",
			port:     0,
			endPort:  0,
			expected: "tcp",
		},
	}

	for _, tc := range testcases {
		pp := &portPolicy{
			protocol: tc.protocol,
			port:     tc.port,
			endPort:  tc.endPort,
		}
		l4Match, _ := pp.getL4Match()
		assert.Equal(t, tc.expected, l4Match)
	}
}
