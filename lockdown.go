package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lightsail"
)

type RuntimeConfigStruct struct {
	Region       string
	Instance     string
	AllowedCIDR4 string
	AllowedCIDR6 string
	Force        bool
	Dryrun       bool
}

func LoadConfig() RuntimeConfigStruct {
	var RuntimeConfig RuntimeConfigStruct

	flag.StringVar(&RuntimeConfig.Region, "r", "", "AWS Region (required)")
	flag.StringVar(&RuntimeConfig.Instance, "i", "", "Lightsail instance name (required)")
	flag.StringVar(&RuntimeConfig.AllowedCIDR4, "4", "", "IPv4 CIDR to allow")
	flag.StringVar(&RuntimeConfig.AllowedCIDR6, "6", "", "IPv6 CIDR to allow")
	flag.BoolVar(&RuntimeConfig.Force, "f", false, "Force update")
	flag.BoolVar(&RuntimeConfig.Dryrun, "d", false, "Dry-run: do everything but send the firewall update to the AWS API.")
	flag.Parse()

	if len(RuntimeConfig.Region) == 0 {
		log.Fatal(errors.New("No AWS Region specified."))
	}
	if len(RuntimeConfig.Instance) == 0 {
		log.Fatal(errors.New("No Lightsail instance specified."))
	}
	if len(RuntimeConfig.AllowedCIDR4) == 0 && len(RuntimeConfig.AllowedCIDR6) == 0 {
		log.Fatal(errors.New("No IPv4 or IPv6 CIDR specified."))
	}

	if RuntimeConfig.AllowedCIDR4 == "none" {
		RuntimeConfig.AllowedCIDR4 = ""
	}

	if len(RuntimeConfig.AllowedCIDR4) > 0 {
		_, Net4, err := net.ParseCIDR(RuntimeConfig.AllowedCIDR4)
		if err != nil {
			log.Fatal(err)
		}
		RuntimeConfig.AllowedCIDR4 = Net4.String()
	}

	if RuntimeConfig.AllowedCIDR6 == "none" {
		RuntimeConfig.AllowedCIDR6 = ""
	}

	if len(RuntimeConfig.AllowedCIDR6) > 0 {
		_, Net6, err := net.ParseCIDR(RuntimeConfig.AllowedCIDR6)
		if err != nil {
			log.Fatal(err)
		}
		RuntimeConfig.AllowedCIDR6 = Net6.String()
	}

	return RuntimeConfig
}

func GetAllowedPorts(svc *lightsail.Lightsail, Instance string) *lightsail.GetInstancePortStatesOutput {
	var err error
	var input lightsail.GetInstancePortStatesInput

	input.SetInstanceName(Instance)
	err = input.Validate()
	if err != nil {
		log.Fatal(err)
	}

	output, err := svc.GetInstancePortStates(&input)
	if err != nil {
		log.Fatal(err)
	}

	return output
}

func SetAllowedPorts(svc *lightsail.Lightsail, Instance string, GIPSO *lightsail.GetInstancePortStatesOutput, AllowedCIDR4 string, AllowedCIDR6 string, Dryrun bool) {
	var err error
	var PIPPI lightsail.PutInstancePublicPortsInput
	var PortInfo []*lightsail.PortInfo

	for _, CurrentPortState := range GIPSO.PortStates {

		var PortInfoEntry lightsail.PortInfo

		PortInfoEntry.SetFromPort(*CurrentPortState.FromPort)
		PortInfoEntry.SetToPort(*CurrentPortState.ToPort)
		PortInfoEntry.SetProtocol(*CurrentPortState.Protocol)

		if len(AllowedCIDR4) > 0 {
			var Dummy []*string
			Dummy = append(Dummy, &AllowedCIDR4)
			PortInfoEntry.SetCidrs(Dummy)
		} else {
			if len(CurrentPortState.Ipv6Cidrs) == 0 {
				continue
			}
		}

		if len(AllowedCIDR6) > 0 {
			var Dummy []*string
			Dummy = append(Dummy, &AllowedCIDR6)
			PortInfoEntry.SetIpv6Cidrs(Dummy)
		} else {
			if len(CurrentPortState.Cidrs) == 0 {
				continue
			}
		}

		PortInfo = append(PortInfo, &PortInfoEntry)
	}

	PIPPI.SetInstanceName(Instance)
	PIPPI.SetPortInfos(PortInfo)
	err = PIPPI.Validate()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(&PIPPI)

	if Dryrun == false {
		_, err = svc.PutInstancePublicPorts(&PIPPI)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Dryrun: no update performed.")
	}
}

func CompareBlocks(Active []*string, Allowed string) (Update bool) {
	Open := len(Allowed) > 0

	if Open && len(Active) != 1 {
		return true
	} else if Open && len(Active) == 1 {
		if *Active[0] != Allowed {
			return true
		}
	} else if !Open && len(Active) != 0 {
		return true
	} else if !Open && len(Active) == 0 {
		return false
	} else {
		log.Panic("???")
	}

	return false
}

func main() {
	// var err error
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	Config := LoadConfig()

	//	fmt.Println("Connecting to", Config.Region, "lightsail instance", Config.Instance)

	mySession := session.Must(session.NewSession())
	svc := lightsail.New(mySession)
	svc = lightsail.New(mySession, aws.NewConfig().WithRegion(Config.Region))

	gipsOutput := GetAllowedPorts(svc, Config.Instance)

	fmt.Printf("Current firewall rules for %s:%s:\n", Config.Region, Config.Instance)

	var Update = false
	for _, e := range gipsOutput.PortStates {

		var FromPort int64 = *e.FromPort
		var ToPort int64 = *e.ToPort

		var Cidrs []string
		for _, r := range e.Cidrs {
			Cidrs = append(Cidrs, *r)
		}

		var Cidrs6 []string
		for _, r := range e.Ipv6Cidrs {
			Cidrs6 = append(Cidrs6, *r)
		}

		Update = Update || CompareBlocks(e.Cidrs, Config.AllowedCIDR4)
		Update = Update || CompareBlocks(e.Ipv6Cidrs, Config.AllowedCIDR6)

		fmt.Printf(" %5d-%-5d %v %v\n", FromPort, ToPort, Cidrs, Cidrs6)
	}

	if Update == false && Config.Force == false {
		fmt.Println("No update required.")
		return
	}

	fmt.Println("Updating firewall IPv4 CIDRs to match", Config.AllowedCIDR4)
	fmt.Println("Updating firewall IPv6 CIDRs to match", Config.AllowedCIDR6)

	SetAllowedPorts(svc, Config.Instance, gipsOutput, Config.AllowedCIDR4, Config.AllowedCIDR6, Config.Dryrun)
}
