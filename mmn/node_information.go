package mmn

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/cihub/seelog"

	"github.com/mysteriumnetwork/node/metadata"
)

type NodeInformation struct {
	MACAddress  string `json:"mac_address"`
	LocalIP     string `json:"local_ip"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	NodeVersion string `json:"node_version"`
}

func GetNodeInformation() (*NodeInformation, error) {
	var mac, ip string
	ip, err := getLocalNetworkIP()

	if err == nil {
		mac, err = getMACAddress(ip)
		if err != nil {
			mac = ""
		}
	}

	info := &NodeInformation{
		MACAddress:  mac,
		LocalIP:     ip,
		Arch:        runtime.GOOS + "/" + runtime.GOARCH,
		OS:          getOS(),
		NodeVersion: metadata.VersionAsString(),
	}

	j, _ := json.Marshal(info)
	fmt.Println(string(j))

	return info, nil
}

func getOS() string {
	if output := getOSByCommand("darwin", "sw_vers", "-productVersion"); len(output) > 0 {
		return "MAC OS X - " + strings.TrimSpace(string(output))
	}

	if output := getOSByCommand("linux", "lsb_release", "-d"); len(output) > 0 {
		return strings.TrimSpace(strings.Replace(string(output), "Description:", "", 1))
	}

	return ""
}

func getOSByCommand(os string, command string, args ...string) string {
	if runtime.GOOS == os {
		output, err := exec.Command(command, args...).Output()
		if err != nil {
			fmt.Println(err.Error())
			return ""
		}
		return string(output)
	}

	return ""
}

func getLocalNetworkIP() (ip string, err error) {
	addresses, err := net.InterfaceAddrs()

	if err != nil {
		log.Error("Failed to get network interface addresses", err)
		return "", err
	}

	for _, address := range addresses {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}

	return
}

func getMACAddress(ip string) (string, error) {
	var currentNetworkHardwareName string
	interfaces, _ := net.Interfaces()
	for _, i := range interfaces {

		if addresses, err := i.Addrs(); err == nil {
			for _, addr := range addresses {
				// only interested in the name with current IP address
				if strings.Contains(addr.String(), ip) {
					currentNetworkHardwareName = i.Name
				}
			}
		}
	}

	netInterface, err := net.InterfaceByName(currentNetworkHardwareName)

	if err != nil {
		log.Error("Failed to get MAC address", err)

		return "", err
	}

	macAddress, err := net.ParseMAC(netInterface.HardwareAddr.String())

	if err != nil {
		log.Error("Failed to validate MAC address", err)
		return "", err
	}

	return macAddress.String(), nil
}
