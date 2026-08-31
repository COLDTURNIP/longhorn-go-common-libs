package net

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/test"
)

func TestIsLoopbackHost(t *testing.T) {
	type testCase struct {
		host     string
		expected bool
	}

	testCases := map[string]testCase{
		"Localhost": {
			host:     "localhost",
			expected: true,
		},
		"127.0.0.1": {
			host:     "127.0.0.1",
			expected: true,
		},
		"0.0.0.0": {
			host:     "0.0.0.0",
			expected: true,
		},
		"::1": {
			host:     "::1",
			expected: true,
		},
		"Empty": {
			host:     "",
			expected: true,
		},
		"8.8.8.8": {
			host:     "8.8.8.8",
			expected: false,
		},
	}

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			result := IsLoopbackHost(testCase.host)
			assert.Equal(t, testCase.expected, result, Commentf(test.ErrResultFmt, testName))
		})
	}
}

func TestGetLocalIPv4fromInterface(t *testing.T) {
	type testCase struct {
		host     string
		expected bool
	}

	testCases := map[string]testCase{
		"Local": {
			host:     "",
			expected: true,
		},
	}

	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			interfaces, err := net.Interfaces()
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))

			for _, iface := range interfaces {
				ip, err := GetLocalIPv4fromInterface(iface.Name)
				if err != nil {
					assert.True(t, strings.Contains(err.Error(), "don't have an IPv4 address"), Commentf(test.ErrResultFmt, testName))
					continue
				}

				assert.True(t, isIPv4(ip), Commentf(test.ErrResultFmt, testName))
			}
		})
	}
}

func TestGetAnyExternalIP(t *testing.T) {
	type testCase struct {
		host     string
		expected bool
	}

	testCases := map[string]testCase{
		"Local": {
			host:     "",
			expected: true,
		},
	}

	for testName := range testCases {
		t.Run(testName, func(t *testing.T) {
			ip, err := GetAnyExternalIP()
			assert.Nil(t, err, Commentf(test.ErrErrorFmt, testName, err))
			assert.True(t, isIPv4(ip), Commentf(test.ErrResultFmt, testName))
		})
	}
}

func isIPv4(ip string) bool {
	return strings.Count(ip, ":") < 2
}

type syntheticAddr struct {
	network string
	address string
}

func (a syntheticAddr) Network() string {
	return a.network
}

func (a syntheticAddr) String() string {
	return a.address
}

func TestParseIPFamily(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected IPFamily
		valid    bool
	}{
		{name: "unspecified", value: "", expected: IPFamilyUnspecified, valid: true},
		{name: "IPv4", value: "ipv4", expected: IPFamilyIPv4, valid: true},
		{name: "IPv6", value: "ipv6", expected: IPFamilyIPv6, valid: true},
		{name: "uppercase IPv4", value: "IPV4", expected: IPFamilyIPv4, valid: true},
		{name: "mixed case IPv6", value: "IPv6", expected: IPFamilyIPv6, valid: true},
		{name: "whitespace padded", value: " ipv4 "},
		{name: "unknown", value: "unknown"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			family, err := ParseIPFamily(testCase.value)
			if testCase.valid {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, family)
				return
			}
			assert.Error(t, err)
			assert.Empty(t, family)
		})
	}
}

func TestParseIPFamilyFromAddress(t *testing.T) {
	testCases := []struct {
		name     string
		address  string
		expected IPFamily
	}{
		{name: "bare IPv4", address: "192.0.2.10", expected: IPFamilyIPv4},
		{name: "IPv4 host-port", address: "192.0.2.10:9500", expected: IPFamilyIPv4},
		{name: "bare IPv6", address: "2001:db8::10", expected: IPFamilyIPv6},
		{name: "IPv6 host-port", address: "[2001:db8::10]:9500", expected: IPFamilyIPv6},
		{name: "hostname", address: "example.com:9500"},
		{name: "missing port", address: "[2001:db8::10]"},
		{name: "empty"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			family, err := ParseIPFamilyFromAddress(testCase.address)
			if testCase.expected == IPFamilyUnspecified {
				assert.Error(t, err)
				assert.Empty(t, family)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, family)
		})
	}
}

func TestGetLocalIPFromAddrsByFamily(t *testing.T) {
	testCases := []struct {
		name     string
		addrs    []net.Addr
		family   IPFamily
		expected string
	}{
		{
			name: "IPv4 from dual-stack addresses",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("2001:db8::1")},
				&net.IPNet{IP: net.ParseIP("192.0.2.10")},
			},
			family:   IPFamilyIPv4,
			expected: "192.0.2.10",
		},
		{
			name: "IPv6 from dual-stack addresses",
			addrs: []net.Addr{
				&net.IPAddr{IP: net.ParseIP("192.0.2.10")},
				&net.IPAddr{IP: net.ParseIP("2001:db8::1")},
			},
			family:   IPFamilyIPv6,
			expected: "2001:db8::1",
		},
		{
			name: "skips unusable IPv4 addresses",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("127.0.0.1")},
				&net.IPNet{IP: net.ParseIP("169.254.10.20")},
				&net.IPNet{IP: net.ParseIP("0.0.0.0")},
				&net.IPNet{IP: net.ParseIP("192.0.2.10")},
			},
			family:   IPFamilyIPv4,
			expected: "192.0.2.10",
		},
		{
			name: "ignores unsupported and malformed addresses",
			addrs: []net.Addr{
				syntheticAddr{network: "synthetic", address: "192.0.2.11/24"},
				(*net.IPNet)(nil),
				(*net.IPAddr)(nil),
				&net.IPNet{IP: net.IP{1, 2, 3}},
				&net.IPAddr{IP: nil},
			},
			family: IPFamilyIPv4,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip := getLocalIPFromAddrsByFamily(testCase.addrs, testCase.family)
			assert.Equal(t, testCase.expected, ip)
		})
	}
}

type testInterfaceResult struct {
	addrs []net.Addr
	err   error
}

type testIPForPodHooks struct {
	interfaces       map[string]testInterfaceResult
	interfaceName    string
	interfaceNameErr error
	interfaceCalls   []string
	nameCalls        int
}

func (h *testIPForPodHooks) interfaceAddrs(name string) ([]net.Addr, error) {
	h.interfaceCalls = append(h.interfaceCalls, name)
	result, ok := h.interfaces[name]
	if !ok {
		return nil, nil
	}
	return result.addrs, result.err
}

func TestGetInterfaceAddrsWithHooks(t *testing.T) {
	up := net.Interface{Name: StorageNetworkInterface, Flags: net.FlagUp}
	down := net.Interface{Name: StorageNetworkInterface}
	expectedAddrs := []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.20")}}

	t.Run("interface list error", func(t *testing.T) {
		addrs, err := getInterfaceAddrsWithHooks(StorageNetworkInterface,
			func() ([]net.Interface, error) {
				return nil, assert.AnError
			},
			func(net.Interface) ([]net.Addr, error) {
				t.Fatal("listAddrs should not be called")
				return nil, nil
			})
		assert.Nil(t, addrs)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("interface absent", func(t *testing.T) {
		addrs, err := getInterfaceAddrsWithHooks(StorageNetworkInterface,
			func() ([]net.Interface, error) {
				return []net.Interface{{Name: "eth0", Flags: net.FlagUp}}, nil
			},
			func(net.Interface) ([]net.Addr, error) {
				t.Fatal("listAddrs should not be called")
				return nil, nil
			})
		assert.Nil(t, addrs)
		assert.NoError(t, err)
	})

	t.Run("interface down", func(t *testing.T) {
		addrs, err := getInterfaceAddrsWithHooks(StorageNetworkInterface,
			func() ([]net.Interface, error) {
				return []net.Interface{down}, nil
			},
			func(net.Interface) ([]net.Addr, error) {
				t.Fatal("listAddrs should not be called")
				return nil, nil
			})
		assert.Nil(t, addrs)
		assert.ErrorContains(t, err, "interface lhnet1 is down")
	})

	t.Run("address list error", func(t *testing.T) {
		addrs, err := getInterfaceAddrsWithHooks(StorageNetworkInterface,
			func() ([]net.Interface, error) {
				return []net.Interface{up}, nil
			},
			func(net.Interface) ([]net.Addr, error) {
				return nil, assert.AnError
			})
		assert.Nil(t, addrs)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("interface without addresses", func(t *testing.T) {
		addrs, err := getInterfaceAddrsWithHooks(StorageNetworkInterface,
			func() ([]net.Interface, error) {
				return []net.Interface{up}, nil
			},
			func(net.Interface) ([]net.Addr, error) {
				return nil, nil
			})
		assert.Nil(t, addrs)
		assert.ErrorContains(t, err, "interface lhnet1 doesn't have address")
	})

	t.Run("addresses returned", func(t *testing.T) {
		addrs, err := getInterfaceAddrsWithHooks(StorageNetworkInterface,
			func() ([]net.Interface, error) {
				return []net.Interface{up}, nil
			},
			func(net.Interface) ([]net.Addr, error) {
				return expectedAddrs, nil
			})
		assert.Equal(t, expectedAddrs, addrs)
		assert.NoError(t, err)
	})
}
func (h *testIPForPodHooks) interfaceNameByIP(net.IP) (string, error) {
	h.nameCalls++
	return h.interfaceName, h.interfaceNameErr
}

func TestUnifiedPodIPSelection(t *testing.T) {
	dualStack := []net.Addr{
		&net.IPNet{IP: net.ParseIP("2001:db8::2")},
		&net.IPNet{IP: net.ParseIP("192.0.2.20")},
	}
	testCases := []struct {
		name            string
		family          IPFamily
		podIP           string
		hooks           *testIPForPodHooks
		expected        string
		expectedErr     string
		expectedErrIs   error
		expectedNameUse int
		expectedCalls   []string
	}{
		{
			name:   "unspecified family selects first storage address",
			family: IPFamilyUnspecified,
			podIP:  "2001:db8::20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {addrs: dualStack},
				},
			},
			expected:        "2001:db8::2",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "unspecified family selects IPv6-only storage",
			family: IPFamilyUnspecified,
			podIP:  "2001:db8::21",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("2001:db8::2")}},
					},
				},
			},
			expected:        "2001:db8::2",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "unspecified family propagates storage read failure",
			family: IPFamilyUnspecified,
			podIP:  "192.0.2.22",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {err: assert.AnError},
				},
			},
			expectedErrIs:   assert.AnError,
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "unspecified family without an address returns legacy error",
			family: IPFamilyUnspecified,
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
				},
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "absent storage keeps primary PodIP first",
			family: IPFamilyUnspecified,
			podIP:  "192.0.2.30",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
					"eth0":                  {addrs: dualStack},
				},
				interfaceName: "eth0",
			},
			expected:        "192.0.2.30",
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:   "IPv4 storage address is authoritative",
			family: IPFamilyIPv4,
			podIP:  "198.51.100.20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {addrs: dualStack},
				},
			},
			expected:        "192.0.2.20",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "IPv6 storage address is authoritative",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {addrs: dualStack},
				},
			},
			expected:        "2001:db8::2",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "IPv6 storage request rejects IPv4-only storage",
			family: IPFamilyIPv6,
			podIP:  "2001:db8::20",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.21")}},
					},
				},
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "absent storage uses alternate family on PodIP interface",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.30",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
					"eth0":                  {addrs: dualStack},
				},
				interfaceName: "eth0",
			},
			expected:        "2001:db8::2",
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:   "absent storage rejects opposite-family PodIP fallback",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.31",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
					"eth0": {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.31")}},
					},
				},
				interfaceName: "eth0",
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:            "invalid typed family returns an error",
			family:          IPFamily("invalid"),
			podIP:           "2001:db8::35",
			hooks:           &testIPForPodHooks{},
			expectedErr:     `invalid IP family "invalid"`,
			expectedNameUse: 0,
		},
		{
			name:   "storage address read failure propagates",
			family: IPFamilyIPv4,
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {
						err: assert.AnError,
					},
				},
			},
			expectedErrIs:   assert.AnError,
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:   "cluster address read failure propagates",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.36",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
					"eth0": {
						err: assert.AnError,
					},
				},
				interfaceName: "eth0",
			},
			expectedErrIs:   assert.AnError,
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "eth0"},
		},
		{
			name:   "no usable address returns legacy generic error",
			family: IPFamilyIPv4,
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
				},
			},
			expectedErr:     "can't get a ip from either the specified interface or the environment variable",
			expectedNameUse: 0,
			expectedCalls:   []string{StorageNetworkInterface},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := getIPForPod(
				testCase.family,
				testCase.podIP,
				testCase.hooks.interfaceAddrs,
				testCase.hooks.interfaceNameByIP,
			)

			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
				assert.Empty(t, ip)
			} else if testCase.expectedErrIs != nil {
				assert.ErrorIs(t, err, testCase.expectedErrIs)
				assert.Empty(t, ip)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, ip)
			}
			assert.Equal(t, testCase.expectedNameUse, testCase.hooks.nameCalls)
			assert.Equal(t, testCase.expectedCalls, testCase.hooks.interfaceCalls)
		})
	}
}
func TestGetLocalIPFromAddrsByFamilyIPv6(t *testing.T) {
	testCases := []struct {
		name     string
		addrs    []net.Addr
		expected string
	}{
		{
			name: "skips link-local before global-unicast",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("fe80::1")},
				&net.IPNet{IP: net.ParseIP("2001:db8::1")},
			},
			expected: "2001:db8::1",
		},
		{
			name: "rejects link-local only",
			addrs: []net.Addr{
				&net.IPNet{IP: net.ParseIP("fe80::1")},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, getLocalIPFromAddrsByFamily(testCase.addrs, IPFamilyIPv6))
		})
	}
}

func TestIsUsableIPForFamily(t *testing.T) {
	testCases := []struct {
		name   string
		ip     net.IP
		family IPFamily
		match  bool
	}{
		{name: "nil IPv4", family: IPFamilyIPv4},
		{name: "global IPv4", ip: net.ParseIP("192.0.2.1"), family: IPFamilyIPv4, match: true},
		{name: "loopback IPv4", ip: net.ParseIP("127.0.0.1"), family: IPFamilyIPv4},
		{name: "link-local IPv4", ip: net.ParseIP("169.254.1.1"), family: IPFamilyIPv4},
		{name: "unspecified IPv4", ip: net.ParseIP("0.0.0.0"), family: IPFamilyIPv4},
		{name: "global IPv4 unspecified family", ip: net.ParseIP("192.0.2.1"), family: IPFamilyUnspecified, match: true},
		{name: "global IPv6 unspecified family", ip: net.ParseIP("2001:db8::1"), family: IPFamilyUnspecified, match: true},
		{name: "link-local unspecified family", ip: net.ParseIP("fe80::1"), family: IPFamilyUnspecified},
		{name: "global IPv6", ip: net.ParseIP("2001:db8::1"), family: IPFamilyIPv6, match: true},
		{name: "link-local IPv6", ip: net.ParseIP("fe80::1"), family: IPFamilyIPv6},
		{name: "multicast IPv6", ip: net.ParseIP("ff02::1"), family: IPFamilyIPv6},
		{name: "IPv4 against IPv6", ip: net.ParseIP("192.0.2.1"), family: IPFamilyIPv6},
		{name: "IPv6 against IPv4", ip: net.ParseIP("2001:db8::1"), family: IPFamilyIPv4},
		{name: "invalid family", ip: net.ParseIP("192.0.2.1"), family: IPFamily("invalid")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.match, IsUsableIPForFamily(testCase.ip, testCase.family))
		})
	}
}

func TestGetIPForPodByNetworkAndFamilyFallbacks(t *testing.T) {
	const expectedError = "can't get a ip from either the specified interface or the environment variable"
	testCases := []struct {
		name            string
		family          IPFamily
		podIP           string
		hooks           *testIPForPodHooks
		expected        string
		expectedError   string
		expectedNameUse int
		expectedCalls   []string
	}{
		{
			name:   "explicit family rejects malformed PodIP",
			family: IPFamilyIPv4,
			podIP:  "not-an-ip",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
				},
			},
			expectedError: expectedError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:   "explicit family rejects opposite-family fallback",
			family: IPFamilyIPv6,
			podIP:  "192.0.2.44",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
					"synthetic0": {
						addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.44")}},
					},
				},
				interfaceName: "synthetic0",
			},
			expectedError:   expectedError,
			expectedNameUse: 1,
			expectedCalls:   []string{StorageNetworkInterface, "synthetic0"},
		},
		{
			name:   "unspecified rejects malformed PodIP",
			family: IPFamilyUnspecified,
			podIP:  "not-an-ip",
			hooks: &testIPForPodHooks{
				interfaces: map[string]testInterfaceResult{
					StorageNetworkInterface: {},
				},
			},
			expectedError: expectedError,
			expectedCalls: []string{StorageNetworkInterface},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := getIPForPod(
				testCase.family,
				testCase.podIP,
				testCase.hooks.interfaceAddrs,
				testCase.hooks.interfaceNameByIP,
			)

			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
				assert.Empty(t, ip)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, ip)
			}
			assert.Equal(t, testCase.expectedNameUse, testCase.hooks.nameCalls)
			assert.Equal(t, testCase.expectedCalls, testCase.hooks.interfaceCalls)
		})
	}
}

func TestSelectIPByNetworkPreference(t *testing.T) {
	testCases := []struct {
		name                  string
		storageNetworkPresent bool
		storageIPs            []string
		podIPs                []string
		expected              string
		expectError           bool
	}{
		{
			name:                  "preserves storage address order",
			storageNetworkPresent: true,
			storageIPs:            []string{"fe80::1", "2001:db8::20", "192.0.2.20"},
			podIPs:                []string{"192.0.2.10"},
			expected:              "2001:db8::20",
		},
		{
			name:                  "storage-present fail-closed",
			storageNetworkPresent: true,
			storageIPs:            []string{"127.0.0.1", "fe80::1"},
			podIPs:                []string{"192.0.2.10"},
			expectError:           true,
		},
		{
			name:     "dual-stack preserves Pod IP order",
			podIPs:   []string{"2001:db8::10", "192.0.2.10"},
			expected: "2001:db8::10",
		},
		{
			name:     "invalid-first valid-second",
			podIPs:   []string{"not-an-ip", "192.0.2.10"},
			expected: "192.0.2.10",
		},
		{
			name:        "empty pod candidates",
			podIPs:      []string{},
			expectError: true,
		},
		{
			name:     "absent storage uses ordered Pod IPs",
			podIPs:   []string{"2001:db8:0:0::10"},
			expected: "2001:db8::10",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := SelectIPByNetworkPreference(
				testCase.storageNetworkPresent, testCase.storageIPs, testCase.podIPs)
			if testCase.expectError {
				assert.Error(t, err)
				assert.Empty(t, ip)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, ip)
		})
	}
}

func TestGetIPForPodByNetwork(t *testing.T) {
	const expectedStorageError = "can't get a ip from either the specified interface or the environment variable"
	const expectedPodIPError = "can't get a ip from either the specified interface or the environment variable"

	testCases := []struct {
		name            string
		podIP           string
		hooks           *testIPForPodHooks
		expected        string
		expectedError   string
		expectedErrorIs error
		expectedCalls   []string
	}{
		{
			name:          "absent storage uses IPv4 PodIP",
			podIP:         "192.0.2.10",
			hooks:         &testIPForPodHooks{interfaces: map[string]testInterfaceResult{StorageNetworkInterface: {}}},
			expected:      "192.0.2.10",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:          "absent storage uses canonical IPv6 PodIP",
			podIP:         "2001:db8:0:0::10",
			hooks:         &testIPForPodHooks{interfaces: map[string]testInterfaceResult{StorageNetworkInterface: {}}},
			expected:      "2001:db8::10",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "IPv4-only storage address",
			podIP: "2001:db8::10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("192.0.2.20")}},
				},
			}},
			expected:      "192.0.2.20",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "IPv6-only storage address",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("2001:db8::20")}},
				},
			}},
			expected:      "2001:db8::20",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "dual-stack storage preserves IPv4-first order",
			podIP: "2001:db8::10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{
						&net.IPNet{IP: net.ParseIP("192.0.2.20")},
						&net.IPNet{IP: net.ParseIP("2001:db8::20")},
					},
				},
			}},
			expected:      "192.0.2.20",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "dual-stack storage preserves IPv6-first order",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{
						&net.IPNet{IP: net.ParseIP("2001:db8::20")},
						&net.IPNet{IP: net.ParseIP("192.0.2.20")},
					},
				},
			}},
			expected:      "2001:db8::20",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "skips link-local before global-unicast storage address",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{
						&net.IPNet{IP: net.ParseIP("fe80::1")},
						&net.IPNet{IP: net.ParseIP("2001:db8::20")},
					},
				},
			}},
			expected:      "2001:db8::20",
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "link-local-only storage is unusable",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{&net.IPNet{IP: net.ParseIP("fe80::1")}},
				},
			}},
			expectedError: expectedStorageError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "malformed storage address is unusable",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{syntheticAddr{network: "ip", address: "not-an-ip"}},
				},
			}},
			expectedError: expectedStorageError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "loopback and multicast storage addresses are unusable",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {
					addrs: []net.Addr{
						&net.IPNet{IP: net.ParseIP("127.0.0.1")},
						&net.IPNet{IP: net.ParseIP("224.0.0.1")},
					},
				},
			}},
			expectedError: expectedStorageError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "storage read error does not fall back to PodIP",
			podIP: "192.0.2.10",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {err: assert.AnError},
			}},
			expectedErrorIs: assert.AnError,
			expectedCalls:   []string{StorageNetworkInterface},
		},
		{
			name:  "malformed PodIP is rejected when storage is absent",
			podIP: "not-an-ip",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {},
			}},
			expectedError: expectedPodIPError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "link-local PodIP is rejected when storage is absent",
			podIP: "fe80::1",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {},
			}},
			expectedError: expectedPodIPError,
			expectedCalls: []string{StorageNetworkInterface},
		},
		{
			name:  "unspecified PodIP is rejected when storage is absent",
			podIP: "0.0.0.0",
			hooks: &testIPForPodHooks{interfaces: map[string]testInterfaceResult{
				StorageNetworkInterface: {},
			}},
			expectedError: expectedPodIPError,
			expectedCalls: []string{StorageNetworkInterface},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := getIPForPod(
				IPFamilyUnspecified,
				testCase.podIP,
				testCase.hooks.interfaceAddrs,
				testCase.hooks.interfaceNameByIP,
			)

			switch {
			case testCase.expectedError != "":
				assert.EqualError(t, err, testCase.expectedError)
				assert.Empty(t, ip)
			case testCase.expectedErrorIs != nil:
				assert.ErrorIs(t, err, testCase.expectedErrorIs)
				assert.Empty(t, ip)
			default:
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, ip)
			}
			assert.Equal(t, testCase.expectedCalls, testCase.hooks.interfaceCalls)
		})
	}
}

func TestPublicPodIPSelectors(t *testing.T) {
	t.Setenv(EnvPodIP, "192.0.2.10")
	testCases := []struct {
		name          string
		resolve       func() (string, error)
		expectedError string
	}{
		{
			name:    "network selector",
			resolve: GetIPForPodByNetwork,
		},
		{
			name: "invalid typed family",
			resolve: func() (string, error) {
				return GetIPForPodByNetworkAndFamily(IPFamily("invalid"))
			},
			expectedError: `invalid IP family "invalid"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ip, err := testCase.resolve()
			if testCase.expectedError != "" {
				assert.EqualError(t, err, testCase.expectedError)
				assert.Empty(t, ip)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, ip)
		})
	}
}

func TestGetInterfaceNameByIPWithHooks(t *testing.T) {
	listError := errors.New("synthetic interface list failure")
	addressError := errors.New("synthetic address lookup failure")
	testCases := []struct {
		name                   string
		ip                     net.IP
		interfaces             []net.Interface
		interfaceError         error
		addrs                  map[string][]net.Addr
		addressErrors          map[string]error
		expected               string
		expectedError          error
		expectedErrorMessage   string
		expectedInterfaceCalls int
		expectedAddressCalls   int
	}{
		{
			name: "nil IP",
		},
		{
			name:                   "interface list error",
			ip:                     net.ParseIP("2001:db8::2"),
			interfaceError:         listError,
			expectedError:          listError,
			expectedInterfaceCalls: 1,
		},
		{
			name:       "address error",
			ip:         net.ParseIP("2001:db8::2"),
			interfaces: []net.Interface{{Name: "broken0"}},
			addressErrors: map[string]error{
				"broken0": addressError,
			},
			expectedError:          addressError,
			expectedErrorMessage:   "interface broken0 doesn't have address: synthetic address lookup failure",
			expectedInterfaceCalls: 1,
			expectedAddressCalls:   1,
		},
		{
			name:       "match",
			ip:         net.ParseIP("2001:db8::2"),
			interfaces: []net.Interface{{Name: "match0"}},
			addrs: map[string][]net.Addr{
				"match0": {&net.IPNet{IP: net.ParseIP("2001:db8::2")}},
			},
			expected:               "match0",
			expectedInterfaceCalls: 1,
			expectedAddressCalls:   1,
		},
		{
			name:       "no match",
			ip:         net.ParseIP("2001:db8::2"),
			interfaces: []net.Interface{{Name: "ipv40"}, {Name: "other0"}},
			addrs: map[string][]net.Addr{
				"ipv40":  {&net.IPNet{IP: net.ParseIP("192.0.2.1")}},
				"other0": {&net.IPNet{IP: net.ParseIP("2001:db8::3")}},
			},
			expectedInterfaceCalls: 1,
			expectedAddressCalls:   2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			interfaceCalls := 0
			addressCalls := 0
			name, err := getInterfaceNameByIPWithHooks(
				testCase.ip,
				func() ([]net.Interface, error) {
					interfaceCalls++
					return testCase.interfaces, testCase.interfaceError
				},
				func(iface net.Interface) ([]net.Addr, error) {
					addressCalls++
					return testCase.addrs[iface.Name], testCase.addressErrors[iface.Name]
				},
			)

			if testCase.expectedError != nil {
				assert.ErrorIs(t, err, testCase.expectedError)
				if testCase.expectedErrorMessage != "" {
					assert.EqualError(t, err, testCase.expectedErrorMessage)
				}
				assert.Empty(t, name)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, name)
			}
			assert.Equal(t, testCase.expectedInterfaceCalls, interfaceCalls)
			assert.Equal(t, testCase.expectedAddressCalls, addressCalls)
		})
	}
}
