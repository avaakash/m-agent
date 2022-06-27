package network

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

// InjectNetworkChaos adds tc qdisc to limit network connectivity
func InjectNetworkChaos(payload []byte) error {

	var (
		networkCommand     string
		chaosType          string
		chasoValue         string
		networkChaosParams NetworkChaosParams
	)

	if err := json.Unmarshal(payload, &networkChaosParams); err != nil {
		return errors.Errorf("failed to unmarshal payload: %v, \nerr: %v", string(payload), err)
	}

	fmt.Printf("networkChaosParams: %v", networkChaosParams)

	switch networkChaosParams.ExperimentName {
	case "os-network-latency":
		chaosType = "latency"
		chasoValue = strconv.Itoa(networkChaosParams.NetworkLatency) + "ms"
	case "os-network-loss":
		chaosType = "loss"
		chasoValue = strconv.Itoa(networkChaosParams.NetworkPacketLossPercentage) + "%"
	}

	destinationIPs, err := GetTargetIps(
		networkChaosParams.DestinationIPs, networkChaosParams.DestinationHosts)
	if err != nil {
		return err
	}

	if len(destinationIPs) > 0 {
		networkCommand = fmt.Sprintf(
			"tc qdisc replace dev %s root handle 1: prio && tc qdisc replace dev %s parent 1:3 netem %s %s",
			networkChaosParams.NetworkInterface, networkChaosParams.NetworkInterface, chaosType, chasoValue)

		for _, ip := range destinationIPs {
			networkCommand = networkCommand + fmt.Sprintf(
				" && tc filter add dev %s protocol ip parent 1:0 prio 3 u32 match %s dst %s flowid 1:3",
				networkChaosParams.NetworkInterface, CheckIPType(ip), ip)
		}
	} else {
		networkCommand = fmt.Sprintf("tc qdisc replace dev %s root netem %s %s", networkChaosParams.NetworkInterface, chaosType, chasoValue)
	}

	_, stderr, err := Shellout(networkCommand)

	if err != nil {
		return errors.Errorf("%s, stderr: %s", err, stderr)
	}

	return nil
}

// RevertNetworkChaos removes tc qdisc to revert network connectivity
func RevertNetworkChaos(payload []byte) error {

	var (
		networkCommand     string
		networkChaosParams NetworkChaosParams
	)

	if err := json.Unmarshal(payload, &networkChaosParams); err != nil {
		return err
	}

	networkCommand = fmt.Sprintf("tc qdisc delete dev %s root", networkChaosParams.NetworkInterface)

	_, stderr, err := Shellout(networkCommand)

	if err != nil {
		return errors.Errorf("%s, stderr: %s", err, stderr)
	}

	return nil
}
