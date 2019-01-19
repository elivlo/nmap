// Package nmap provides idiomatic `nmap` bindings for go developers.
package nmap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/pkg/errors"
)

// Scanner represents an Nmap scanner.
type Scanner struct {
	args       []string
	binaryPath string
	ctx        context.Context
}

// Run runs nmap synchronously and returns the result of the scan.
func (s *Scanner) Run() (*Run, error) {
	var stdout, stderr bytes.Buffer

	// Enable XML output
	s.args = append(s.args, "-oX")

	// Get XML output in stdout instead of writing it in a file
	s.args = append(s.args, "-")

	cmd := exec.Command(s.binaryPath, s.args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	// Make a goroutine to notify the select when the scan is done.
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-s.ctx.Done():
		// Context was done before the scan was finished. The process is killed and a timeout
		// error is returned.
		cmd.Process.Kill()
		return nil, ErrTimeout
	case err := <-done:
		// Scan finished before timeout.
		if err != nil {
			return nil, err
		}

		if stderr.Len() > 0 {
			return nil, errors.New(stderr.String())
		}

		return Parse(stdout.Bytes())
	}
}

func (s Scanner) String() string {
	return fmt.Sprint(s.binaryPath, s.args)
}

// New creates a new Scanner, and can take options to apply to the scanner.
func New(options ...func(*Scanner)) (*Scanner, error) {
	scanner := &Scanner{}

	for _, option := range options {
		option(scanner)
	}

	if scanner.binaryPath == "" {
		var err error
		scanner.binaryPath, err = exec.LookPath("nmap")
		if err != nil {
			return nil, ErrNmapNotInstalled
		}
	}

	return scanner, nil
}

// WithContext adds a context to a scanner, to make it cancellable and able to timeout.
func WithContext(ctx context.Context) func(*Scanner) {
	return func(s *Scanner) {
		s.ctx = ctx
	}
}

// WithBinaryPath sets the nmap binary path for a scanner.
func WithBinaryPath(binaryPath string) func(*Scanner) {
	return func(s *Scanner) {
		s.binaryPath = binaryPath
	}
}

/*** Target specification ***/

// WithTarget sets the target of a scanner.
func WithTarget(target string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, target)
	}
}

// WithTargetExclusion sets the excluded targets of a scanner.
func WithTargetExclusion(target string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--exclude")
		s.args = append(s.args, target)
	}
}

// WithTargetInput sets the input file name to set the targets.
func WithTargetInput(inputFileName string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-iL")
		s.args = append(s.args, inputFileName)
	}
}

// WithTargetExclusionInput sets the input file name to set the target exclusions.
func WithTargetExclusionInput(inputFileName string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--excludefile")
		s.args = append(s.args, inputFileName)
	}
}

// WithRandomTargets sets the amount of targets to randomly choose from the targets.
func WithRandomTargets(randomTargets int) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-iR")
		s.args = append(s.args, fmt.Sprint(randomTargets))
	}
}

/*** Host discovery ***/

// WithListScan sets the discovery mode to simply list the targets to scan and not scan them.
func WithListScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-sL")
	}
}

// WithPingScan sets the discovery mode to simply ping the targets to scan and not scan them.
func WithPingScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-sn")
	}
}

// WithSkipHostDiscovery diables host discovery and considers all hosts as online.
func WithSkipHostDiscovery() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-Pn")
	}
}

// WithSYNDiscovery sets the discovery mode to use SYN packets.
// If the portList argument is empty, this will enable SYN discovery
// for all ports. Otherwise, it will be only for the specified ports.
func WithSYNDiscovery(portList string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, fmt.Sprintf("-PS%s", portList))
	}
}

// WithACKDiscovery sets the discovery mode to use ACK packets.
// If the portList argument is empty, this will enable ACK discovery
// for all ports. Otherwise, it will be only for the specified ports.
func WithACKDiscovery(portList string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, fmt.Sprintf("-PA%s", portList))
	}
}

// WithUDPDiscovery sets the discovery mode to use UDP packets.
// If the portList argument is empty, this will enable UDP discovery
// for all ports. Otherwise, it will be only for the specified ports.
func WithUDPDiscovery(portList string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, fmt.Sprintf("-PU%s", portList))
	}
}

// WithSCTPDiscovery sets the discovery mode to use SCTP packets
// containing a minimal INIT chunk.
// If the portList argument is empty, this will enable SCTP discovery
// for all ports. Otherwise, it will be only for the specified ports.
// Warning: on Unix, only the privileged user root is generally
// able to send and receive raw SCTP packets.
func WithSCTPDiscovery(portList string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, fmt.Sprintf("-PY%s", portList))
	}
}

// WithICMPEchoDiscovery sets the discovery mode to use an ICMP type 8
// packet (an echo request), like the standard packets sent by the ping
// command.
// Many hosts and firewalls block these packets, so this is usually not
// the best for exploring networks.
func WithICMPEchoDiscovery() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-PE")
	}
}

// WithICMPTimestampDiscovery sets the discovery mode to use an ICMP type 13
// packet (a timestamp request).
// This query can be valuable when administrators specifically block echo
// request packets while forgetting that other ICMP queries can be used
// for the same purpose.
func WithICMPTimestampDiscovery() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-PP")
	}
}

// WithICMPNetMaskDiscovery sets the discovery mode to use an ICMP type 17
// packet (an address mask request).
// This query can be valuable when administrators specifically block echo
// request packets while forgetting that other ICMP queries can be used
// for the same purpose.
func WithICMPNetMaskDiscovery() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-PM")
	}
}

// WithIPProtocolPingDiscovery sets the discovery mode to use the IP
// protocol ping.
// If no protocols are specified, the default is to send multiple IP
// packets for ICMP (protocol 1), IGMP (protocol 2), and IP-in-IP
// (protocol 4).
func WithIPProtocolPingDiscovery(protocolList string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, fmt.Sprintf("-PO%s", protocolList))
	}
}

// WithDisabledDNSResolution disables DNS resolution in the discovery
// step of the nmap scan.
func WithDisabledDNSResolution() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-n")
	}
}

// WithForcedDNSResolution enforces DNS resolution in the discovery
// step of the nmap scan.
func WithForcedDNSResolution() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-R")
	}
}

// WithCustomDNSServers sets custom DNS servers for the scan.
// List format: dns1[,dns2],...
func WithCustomDNSServers(dnsList string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--dns-servers")
		s.args = append(s.args, dnsList)
	}
}

// WithSystemDNS sets the scanner's DNS to the system's DNS.
func WithSystemDNS() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--system-dns")
	}
}

// WithTraceRoute enables the tracing of the hop path to each host.
func WithTraceRoute() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--traceroute")
	}
}

/*** Scan techniques ***/

// WithSYNScan sets the scan technique to use SYN packets over TCP.
// This is the default method, as it is fast, stealthy and not
// hampered by restrictive firewalls.
func WithSYNScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--sS")
	}
}

// WithConnectScan sets the scan technique to use TCP connections.
// This is the default method used when a user does not have raw
// packet privileges. Target machines are likely to log these
// connections.
func WithConnectScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--sT")
	}
}

// WithACKScan sets the scan technique to use ACK packets over TCP.
// This scan is unable to determine if a port is open.
// When scanning unfiltered systems, open and closed ports will both
// return a RST packet.
// Nmap then labels them as unfiltered, meaning that they are reachable
// by the ACK packet, but whether they are open or closed is undetermined.
func WithACKScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--sA")
	}
}

// WithWindowScan sets the scan technique to use ACK packets over TCP and
// examining the TCP window field of the RST packets returned.
// Window scan is exactly the same as ACK scan except that it exploits
// an implementation detail of certain systems to differentiate open ports
// from closed ones, rather than always printing unfiltered when a RST
// is returned.
func WithWindowScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--sW")
	}
}

// WithMaimonScan sends the same packets as NULL, FIN, and Xmas scans,
// except that the probe is FIN/ACK. Many BSD-derived systems will drop
// these packets if the port is open.
func WithMaimonScan() func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--sM")
	}
}

/*** Port specification and scan order ***/

// WithPorts sets the ports which the scanner should scan on each host.
func WithPorts(ports string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "-p")
		s.args = append(s.args, ports)
	}
}

// WithPortExclusions sets the ports that the scanner should not scan on each host.
func WithPortExclusions(ports string) func(*Scanner) {
	return func(s *Scanner) {
		s.args = append(s.args, "--exclude-ports")
		s.args = append(s.args, ports)
	}
}
