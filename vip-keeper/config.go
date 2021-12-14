package vip_keeper

import (
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"sort"
	"strings"
)

type Config struct {
	IP        string   `mapstructure:"ip"`
	Mask      int      `mapstructure:"netmask"`
	Iface     string   `mapstructure:"interface"`
	Endpoints []string `mapstructure:"endpoints"`
	User      string   `mapstructure:"user"`
	Password  string   `mapstructure:"password"`
	Interval  int      `mapstructure:"interval"` //milliseconds
}

func NewConfig() (*Config, error) {
	var err error
	defineFlags()
	pflag.Parse()
	// import p_flags into viper
	_ = viper.BindPFlags(pflag.CommandLine)
	setDefaults()
	if err = checkMandatory(); err != nil {
		return nil, err
	}
	conf := &Config{}
	err = viper.Unmarshal(conf)
	if err != nil {
		log.Fatalf("unable to decode viper config into config struct, %v", err)
	}
	printSettings()
	return conf, err
}

func printSettings() {
	s := []string{}

	for k, v := range viper.AllSettings() {
		if v != "" {
			switch k {
			case "password":
				fallthrough
			default:
				s = append(s, fmt.Sprintf("\t%s : %v\n", k, v))
			}
		}
	}

	sort.Strings(s)
	log.Println("This is the config that will be used:")
	for k := range s {
		fmt.Print(s[k])
	}
}

func setDefaults() {
	if !viper.IsSet("interval") {
		viper.SetDefault("interval", 1000)
	}
	if viper.IsSet("endpoints") {
		endpointsString := viper.GetString("endpoints")
		if strings.Contains(endpointsString, ",") {
			viper.Set("endpoints", strings.Split(endpointsString, ","))
		}
	} else {
		log.Println("No endpoints specified, trying to use localhost with standard ports!")
		viper.Set("endpoints", []string{"http://127.0.0.1:2379"})
	}
}

func defineFlags() {
	pflag.Bool("version", false, "Show the version number.")
	pflag.String("ip", "", "Virtual IP address to configure.")
	pflag.String("netmask", "", "The netmask used for the IP address. Defaults to -1 which assigns ipv4 default mask.")
	pflag.String("interface", "", "Network interface to configure on .")
	pflag.String("endpoints", "", "Endpoint(s), separate multiple endpoints using commas. (default \"http://127.0.0.1:2379\" or \"http://127.0.0.1:8500\" depending on dcs-type.)")
	pflag.String("user", "", "Username for etcd DCS endpoints.")
	pflag.String("password", "", "Password for etcd DCS endpoints.")
	pflag.String("interval", "1000", "DCS scan interval in milliseconds.")
	pflag.CommandLine.SortFlags = false
}

func checkMandatory() error {
	mandatory := []string{
		"ip",
		"netmask",
		"interface",
	}
	success := true
	for _, v := range mandatory {
		success = checkSetting(v) && success
	}
	if !success {
		return errors.New("one or more mandatory settings were not set")
	}
	return nil
}

func checkSetting(name string) bool {
	if !viper.IsSet(name) {
		log.Printf("Setting %s is mandatory", name)
		return false
	}
	return true
}

