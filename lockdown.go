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

	if len(RuntimeConfig.AllowedCIDR4) > 0 && RuntimeConfig.AllowedCIDR4 != "none" {
		_, Net4, err := net.ParseCIDR(RuntimeConfig.AllowedCIDR4)
		if err != nil {
			log.Fatal(err)
		}
		RuntimeConfig.AllowedCIDR4 = Net4.String()
	}

	if len(RuntimeConfig.AllowedCIDR6) > 0 && RuntimeConfig.AllowedCIDR6 != "none" {
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

func SetAllowedPorts(svc *lightsail.Lightsail, Instance string, Existing *lightsail.GetInstancePortStatesOutput, AllowedCIDR4 string, AllowedCIDR6 string, Dryrun bool) {
	var err error
	var Input lightsail.PutInstancePublicPortsInput
	var InputPorts []*lightsail.PortInfo

	for _, e := range Existing.PortStates {

		var PortInfo lightsail.PortInfo

		PortInfo.SetFromPort(*e.FromPort)
		PortInfo.SetToPort(*e.ToPort)
		PortInfo.SetProtocol(*e.Protocol)

		if AllowedCIDR4 != "none" {
			if len(AllowedCIDR4) > 0 {
				var Dummy []*string
				Dummy = append(Dummy, &AllowedCIDR4)
				PortInfo.SetCidrs(Dummy)
			} else {
				PortInfo.SetCidrs(e.Cidrs)
			}
		}

		if AllowedCIDR6 != "none" {
			if len(AllowedCIDR6) > 0 {
				var Dummy []*string
				Dummy = append(Dummy, &AllowedCIDR6)
				PortInfo.SetIpv6Cidrs(Dummy)
			} else {
				PortInfo.SetIpv6Cidrs(e.Ipv6Cidrs)
			}

			InputPorts = append(InputPorts, &PortInfo)
		}
	}

	Input.SetInstanceName(Instance)
	Input.SetPortInfos(InputPorts)
	err = Input.Validate()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(&Input)

	if Dryrun == false {
		_, err = svc.PutInstancePublicPorts(&Input)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("Dryrun: no update performed.")
	}
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

		var Cidrs4 []string
		for _, r := range e.Cidrs {
			Cidrs4 = append(Cidrs4, *r)
			if *r != Config.AllowedCIDR4 && len(Config.AllowedCIDR4) > 0 {
				Update = true
			}
		}
		if len(Config.AllowedCIDR4) > 0 && len(e.Cidrs) == 0 {
			Update = true
		}

		var Cidrs6 []string
		for _, r := range e.Ipv6Cidrs {
			Cidrs6 = append(Cidrs6, *r)
			if *r != Config.AllowedCIDR6 && len(Config.AllowedCIDR6) > 0 {
				Update = true
			}
		}
		if len(Config.AllowedCIDR6) > 0 && len(e.Ipv6Cidrs) == 0 {
			Update = true
		}

		fmt.Printf(" %5d-%-5d %v %v\n", FromPort, ToPort, Cidrs4, Cidrs6)
	}

	if Update == false && Config.Force == false {
		fmt.Println("No update required.")
		return
	}

	fmt.Println("Updating firewall CIDRs to match", Config.AllowedCIDR4)

	SetAllowedPorts(svc, Config.Instance, gipsOutput, Config.AllowedCIDR4, Config.AllowedCIDR6, Config.Dryrun)
}
