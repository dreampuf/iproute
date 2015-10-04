package main

import (
	"flag"
	"fmt"
	"github.com/aeden/traceroute"
	"net"
	"github.com/wangtuanjie/ip17mon"
	"os"
)

func printHop(hop traceroute.TracerouteHop) {
	addr := address(hop.Address)
	hostOrAddr := addr
	if hop.Host != "" {
		hostOrAddr = hop.Host
	}
	if hop.Success {
		loc, err := ip17mon.Find(addr)
		if err == nil {
			fmt.Printf("%-3d %v (%v) %s %v\n", hop.TTL, hostOrAddr, addr, meaningfulOutput(loc), hop.ElapsedTime)
		} else {
			fmt.Printf("%-3d %v (%v)   %v\n", hop.TTL, hostOrAddr, addr, hop.ElapsedTime)
		}
	} else {
		fmt.Printf("%-3d *\n", hop.TTL)
	}
}

func meaningfulOutput(loc *ip17mon.LocationInfo) string {
	output := ""
	for _, i := range []string{ loc.Country, loc.City, loc.Isp } {
		if i != "中国" && i != "N/A" {
			output += i
		}
	}
	return output
}

func address(address [4]byte) string {
	return fmt.Sprintf("%v.%v.%v.%v", address[0], address[1], address[2], address[3])
}

func main() {
	var m = flag.Int("m", traceroute.DEFAULT_MAX_HOPS, `Set the max time-to-live (max number of hops) used in outgoing probe packets (default is 64)`)
	var q = flag.Int("q", 1, `Set the number of probes per "ttl" to nqueries (default is one probe).`)
	var d = flag.String("d", "17monipdb.dat", `IP locate database path (default is 17monipdb.dat)`)

	flag.Parse()
	host := flag.Arg(0)

	if err := ip17mon.Init(*d); err != nil {
		fmt.Printf("Unvalied IP locate database (%s) with error: %s\n", d, err.Error())
		fmt.Println("Please provider a vailided IP locate database (download from here: http://s.qdcdn.com/17mon/17monipdb.zip)")
		os.Exit(1)
	}

	options := traceroute.TracerouteOptions{}
	options.SetRetries(*q - 1)
	options.SetMaxHops(*m + 1)

	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return
	}

	fmt.Printf("traceroute to %v (%v), %v hops max, %v byte packets\n", host, ipAddr, options.MaxHops(), options.PacketSize())

	c := make(chan traceroute.TracerouteHop, 0)
	go func() {
		for {
			hop, ok := <-c
			if !ok {
				fmt.Println()
				return
			}
			printHop(hop)
		}
	}()

	_, err = traceroute.Traceroute(host, &options, c)
	if err != nil {
		if err.Error() == "operation not permitted" {
			fmt.Println("Operation not permitted (Maybe with sudo?)")
		} else {
			fmt.Printf("Error: ", err)
		}
	}
}
