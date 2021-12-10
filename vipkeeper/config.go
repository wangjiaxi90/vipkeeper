package vipkeeper

import (
	"errors"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"os"
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
	// make viper look for env variables that are prefixed VIP_...
	// e.g.: viper.getString("ip") will return the value of env variable VIP_IP
	viper.SetEnvPrefix("vip")
	viper.AutomaticEnv()

	if err = mapDeprecated(); err != nil {
		return nil, err
	}
	setDefaults()

	if viper.IsSet("endpoints") {
		endpointsString := viper.GetString("endpoints")
		if strings.Contains(endpointsString, ",") {
			viper.Set("dcs-endpoints", strings.Split(endpointsString, ","))
		}
	} else {
		log.Println("No endpoints specified, trying to use localhost with standard ports!")
		viper.Set("dcs-endpoints", []string{"http://127.0.0.1:2379"})
	}

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
}

func mapDeprecated() error {
	deprecated := map[string]string{
		// "deprecated" : "new",
		"ip":        "ip",
		"mask":      "netmask",
		"iface":     "interface",
		"user":      "user",
		"password":  "password",
		"endpoints": "endpoints",
		"interval":  "interval",
	}
	complaints := []string{}
	errors := false
	for k, v := range deprecated {
		if viper.IsSet(k) {
			if _, exists := os.LookupEnv("VIP_" + strings.ToUpper(k)); !exists {
				// using deprecated key in config file (as not exists in ENV)
				complaints = append(complaints, fmt.Sprintf("Parameter \"%s\" has been deprecated, please use \"%s\" instead", k, v))
			} else {
				if k != v {
					// this string is not a direct replacement (e.g. etcd-user replaces etcd-user, i.e. in both cases VIP_ETCD_USER is the valid env key)
					// for example, complain about VIP_IFACE, but not VIP_CONSUL_TOKEN or VIP_ETCD_USER...
					complaints = append(complaints, fmt.Sprintf("Parameter \"%s\" has been deprecated, please use \"%s\" instead", "VIP_"+strings.ToUpper(k), "VIP_"+strings.ReplaceAll(strings.ToUpper(v), "-", "_")))
				} else {
					continue
				}
			}

			if viper.IsSet(v) {
				if viper.IsSet(v) {
					complaints = append(complaints, fmt.Sprintf("Conflicting settings: %s or %s and %s or %s are both specified…", k, "VIP_"+strings.ToUpper(k), v, "VIP_"+strings.ReplaceAll(strings.ToUpper(v), "-", "_")))
					if viper.Get(k) == viper.Get(v) {
						complaints = append(complaints, fmt.Sprintf("… But no conflicting values: %s and %s are equal…ignoring.", viper.GetString(k), viper.GetString(v)))
						continue
					} else {
						complaints = append(complaints, fmt.Sprintf("…conflicting values: %s and %s", viper.GetString(k), viper.GetString(v)))
						errors = true
						continue
					}
				}
			}
			// if this is a valid mapping due to deprecation, set the new key explicitly to the value of the deprecated key.
			viper.Set(v, viper.Get(k))
			// "unset" the deprecated setting so it will not show up in our config later
			viper.Set(k, "")

		}
	}
	for c := range complaints {
		log.Println(complaints[c])
	}
	if errors {
		log.Fatal("Cannot continue due to conflicts.")
	}
	return nil
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
