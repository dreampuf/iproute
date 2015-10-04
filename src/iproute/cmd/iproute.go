package main

import (
	"flag"
	"fmt"
	"github.com/aeden/traceroute"
	"net"
	"os"
	"github.com/oschwald/geoip2-golang"
	"errors"
)

var (
	db geoip2.Reader
)

type Coord struct {
	TTL int
	Latitude, Longitude float64
	City string
}

func printHop(db *geoip2.Reader, hop traceroute.TracerouteHop) (Coord, error) {
	addr := address(hop.Address)
	hostOrAddr := addr
	if hop.Host != "" {
		hostOrAddr = hop.Host
	}
	if hop.Success {
		loc, err := ip2loc(db, addr)
		if err == nil {
			fmt.Printf("%-3d %v (%v) %s %v\n", hop.TTL, hostOrAddr, addr, meaningfulOutput(loc), hop.ElapsedTime)
		} else {
			fmt.Printf("%-3d %v (%v)   %v\n", hop.TTL, hostOrAddr, addr, hop.ElapsedTime)
		}
		//fmt.Printf("%s %s", loc.Location.Latitude, loc.Location.Longitude)
		return Coord{ hop.TTL, loc.Location.Latitude, loc.Location.Longitude, loc.City.Names["en"]}, nil
	} else {
		fmt.Printf("%-3d *\n", hop.TTL)
		return Coord{}, errors.New("get location failed")
	}
}

func ip2loc(db *geoip2.Reader, addr string) (*geoip2.City, error) {
	ip := net.ParseIP(addr)
	return db.City(ip)
}
func meaningfulOutput(loc *geoip2.City) string {
	output := ""
	for _, i := range []string{ loc.Country.Names["zh-CN"], loc.City.Names["zh-CN"] } {
		output += i
	}
	return output
}

func imageURL(coords []Coord) string {
	tpl := "http://restapi.amap.com/v3/staticmap?zoom=1&size=1024*500&markers=%s&paths=%s&key=ee95e52bf08006f63fd29bcfbcf21df0"
	markers := ""
	paths := "10,,,,:"
	for n, i := range coords {
		markers += fmt.Sprintf("mid,,%d:%.4f,%.4f|", n, i.Longitude, i.Latitude)
		paths += fmt.Sprintf("%.4f,%.4f;", i.Longitude, i.Latitude)
	}
	if markers[len(markers)-1] == '|' {
		markers = markers[:len(markers)-1]
	}
	if paths[len(paths)-1] == ';' {
		paths = paths[:len(paths)-1]
	}
	return fmt.Sprintf(tpl, markers, paths)
}

func address(address [4]byte) string {
	return fmt.Sprintf("%v.%v.%v.%v", address[0], address[1], address[2], address[3])
}

func main() {
	var m = flag.Int("m", traceroute.DEFAULT_MAX_HOPS, `Set the max time-to-live (max number of hops) used in outgoing probe packets (default is 64)`)
	var q = flag.Int("q", 1, `Set the number of probes per "ttl" to nqueries (default is one probe).`)
	var d = flag.String("d", "GeoLite2-City.mmdb", `IP locate database path (default is 17monipdb.dat)`)

	flag.Parse()
	host := flag.Arg(0)
	db, err := geoip2.Open(*d)
	if  err != nil {
		fmt.Printf("Unvalied IP locate database (%s) with error: %s\n", d, err.Error())
		fmt.Println("Please provider a vailided IP locate database (download from here: http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz)")
		os.Exit(1)
	}
	defer db.Close()

	options := traceroute.TracerouteOptions{}

	options.SetRetries(*q - 1)
	options.SetMaxHops(*m + 1)

	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return
	}

	fmt.Printf("traceroute to %v (%v), %v hops max, %v byte packets\n", host, ipAddr, options.MaxHops(), options.PacketSize())

	c := make(chan traceroute.TracerouteHop, 0)
	coords := make([]Coord, 0)
	lastCity := ""
	go func() {
		for {
			hop, ok := <-c
			if !ok {
				fmt.Println()
				return
			}
			coord, err := printHop(db, hop)
			if err == nil && coord.City != lastCity {
				lastCity = coord.City
				coords = append(coords, coord)
			}
		}
	}()

	_, err = traceroute.Traceroute(host, &options, c)
	if err != nil {
		if err.Error() == "operation not permitted" {
			fmt.Println("Operation not permitted (Maybe with sudo?)")
		} else {
			fmt.Printf("Error: ", err)
		}
	} else {
		fmt.Printf("Visualize URL: %s\n", imageURL(coords))
	}
}
