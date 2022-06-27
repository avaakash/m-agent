package network

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type NetworkChaosParams struct {
	ExperimentName              string
	NetworkLatency              int
	NetworkPacketLossPercentage int
	DestinationHosts            string
	DestinationIPs              string
	NetworkInterface            string
}

// NetworkChaosPreRequisiteCheck checks for vm status and other pre-requisites
func NetworkChaosPreRequisiteCheck(payload []byte) error {

	var networkChaosParams NetworkChaosParams

	if err := json.Unmarshal(payload, &networkChaosParams); err != nil {
		return err
	}

	if networkChaosParams.NetworkInterface == "" {
		return errors.Errorf("no network interface provide")
	}

	isTCInstalled, err := getTCStatus()
	if err != nil {
		return errors.Errorf("unable to verify if tc is installed, err: %v", err)
	}

	if !isTCInstalled {
		return errors.Errorf("tc not found")
	}

	isNetworkInterfacePresent, err := verifyNetworkInterface(networkChaosParams.NetworkInterface)
	if err != nil {
		return errors.Errorf("unable to verify if network interface is present, err: %v", err)
	}

	if !isNetworkInterfacePresent {
		return errors.Errorf("network interface %v not found.", networkChaosParams.NetworkInterface)
	}
	return nil
}

// getTCStatus returns true if tc is installed
func getTCStatus() (bool, error) {
	command := `command -v "tc" &> /dev/null && echo true || echo false`
	stdout, stderr, err := Shellout(command)

	if err != nil {
		return false, err
	} else if stderr != "" {
		return false, errors.Errorf("%s", stderr)
	}

	// a newline character gets appendeed to the end of the string in stdout
	return strconv.ParseBool(strings.TrimSuffix(stdout, "\n"))
}

// verifyNetworkInterface checks if the given network interface is present
func verifyNetworkInterface(networkInterface string) (bool, error) {
	command := fmt.Sprintf(`ip link show | grep %s > /dev/null && echo True || echo False`, networkInterface)
	stdout, stderr, err := Shellout(command)
	if err != nil {
		return false, err
	} else if stderr != "" {
		return false, errors.Errorf("%s", stderr)
	}
	if stdout == "" {
		return false, nil
	}

	return strconv.ParseBool(strings.TrimSuffix(stdout, "\n"))
}

// GetTargetIps return the comma separated target ips
// It fetch the ips from the target ips (if defined by users)
// it append the ips from the host, if target host is provided
func GetTargetIps(targetIPs, targetHosts string) ([]string, error) {

	var uniqueIps []string

	ipsFromHost, err := getIpsForTargetHosts(targetHosts)
	if err != nil {
		return uniqueIps, err
	}
	if targetIPs == "" {
		targetIPs = ipsFromHost
	} else if ipsFromHost != "" {
		targetIPs = targetIPs + "," + ipsFromHost
	}

	if targetIPs != "" {
		ips := strings.Split(targetIPs, ",")

		// removing duplicates ips from the list, if any
		for i := range ips {
			isPresent := false
			for j := range uniqueIps {
				if ips[i] == uniqueIps[j] {
					isPresent = true
				}
			}
			if !isPresent {
				// checking the validity of the ip
				err := verifyIP(ips[i])
				if err != nil {
					uniqueIps = append(uniqueIps, ips[i])
				}
			}
		}
	}
	return uniqueIps, nil
}

// getIpsForTargetHosts resolves IP addresses for comma-separated list of target hosts and returns comma-separated ips
func getIpsForTargetHosts(targetHosts string) (string, error) {

	if targetHosts == "" {
		return "", nil
	}

	hosts := strings.Split(targetHosts, ",")
	fmt.Printf("Resolving IPs for target hosts(array): %v", hosts)
	finalHosts := ""
	var commaSeparatedIPs []string
	for i := range hosts {
		command := fmt.Sprintf(`host %s | grep -oP "(?<=address).*"`, hosts[i])
		stdout, _, err := Shellout(command)
		if err != nil {
			HostConvertedIPs := strings.Split(stdout, "\n")
			fmt.Printf("Resolved IPs for target hosts %v: %v", hosts[i], HostConvertedIPs)
			for _, ip := range HostConvertedIPs {
				if ip != "" {
					commaSeparatedIPs = append(commaSeparatedIPs, ip)
				}
			}
			if finalHosts == "" {
				finalHosts = hosts[i]
			} else {
				finalHosts = finalHosts + "," + hosts[i]
			}
		}
	}
	if len(commaSeparatedIPs) == 0 {
		return "", errors.Errorf("provided hosts: {%v} are invalid, unable to resolve", targetHosts)
	}
	return strings.Join(commaSeparatedIPs, ","), nil
}

// checkIPType returns the ip type based on the ip (ip4\ip6)
func CheckIPType(ip string) string {
	if strings.Contains(ip, ":") {
		return "ip6"
	}
	return "ip"
}

// verifyIP checks the correctness of the ip
func verifyIP(ip string) error {
	if strings.Contains(ip, ":") {
		if net.ParseIP(ip) == nil {
			return errors.Errorf("invalid ipv6 address: %s", ip)
		}
	} else {
		if net.ParseIP(ip) == nil {
			return errors.Errorf("invalid ipv4 address: %s", ip)
		}
	}
	return nil
}
