package main

/* stdlib includes */
import (
	"flag"
	"fmt"
	"os"
)

/* external includes */
import "github.com/op/go-logging"

/* definitions */

/* meat */

/* dependancy injection is for another day */
var log = logging.MustGetLogger(os.Args[0])

func main() {
	var configPath string
	var config Configuration

	flag.StringVar(&configPath, "config", "etc/hfm.conf", "Configuration file path")
	flag.Parse()

	e := config.LoadConfiguration(configPath)
	if e != nil {
		log.Error(fmt.Sprintf("Could not load configuration file %v: %+v", configPath, e))
		panic(e)
	}

	fmt.Println("Rule defaults")
	for _, rule := range config.ruleDefaults {
		fmt.Printf("%+v\n", rule)
	}

	fmt.Println("")

	fmt.Println("Rules")
	for _, rule := range config.rules {
		fmt.Printf("%+v\n", rule)
	}
}
