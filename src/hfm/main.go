package main

/* stdlib includes */
import (
	"flag"
	"fmt"
	"os"
)

/* external includes */
import "github.com/op/go-logging"
import "github.com/mitchellh/go-libucl"

/* dependancy injection is for another day */
var log = logging.MustGetLogger(os.Args[0])

func load_configuration(config_path string) (*libucl.Object, error) {
	p := libucl.NewParser(0)
	defer p.Close()

	e := p.AddFile(config_path)
	if e != nil {
		log.Error(fmt.Sprintf("Could not load configuration file %v: %+v", config_path, e))
		return nil, e
	}

	config := p.Object()
	return config, nil
}

func main() {
	var config_path string

	flag.StringVar(&config_path, "config", "etc/hfm.conf", "Configuration file path")
	flag.Parse()

	config, e := load_configuration(config_path)
	if e != nil {
		log.Error(fmt.Sprintf("Could not load configuration file %v: %+v", config_path, e))
		panic(e)
	}

	fmt.Println(config.Emit(libucl.EmitConfig))

	fmt.Println("Iterating over config")

	group_iter := config.Iterate(true)
	defer group_iter.Close()

	//	var test string

	/* for groups */
	for group := group_iter.Next(); group != nil; group = group_iter.Next() {
		defer group.Close()
		fmt.Println(group.Key())

		rule_iter := group.Iterate(true)
		defer rule_iter.Close()

		for rule := rule_iter.Next(); rule != nil; rule = rule_iter.Next() {
			defer group.Close()
			fmt.Println(rule.Key())
		}
	}

	fmt.Println("end...")
}

