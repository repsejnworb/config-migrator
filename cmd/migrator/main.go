package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"config-migrator/pkg/migrator"
)

func main() {
	from := flag.String("from", "", "source version")
	to := flag.String("to", "", "target version")
	input := flag.String("in", "", "input config file")
	migFile := flag.String("mig", "", "migration file")
	flag.Parse()

	if *from == "" || *to == "" || *input == "" || *migFile == "" {
		fmt.Println("Usage: migrator --from v1 --to v2 --in config.json --mig migrations/v1_to_v2.json")
		os.Exit(1)
	}

	engine := migrator.NewEngine()
	if err := engine.LoadMigration(*migFile); err != nil {
		panic(err)
	}

	raw, err := os.ReadFile(*input)
	if err != nil {
		panic(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		panic(err)
	}

	newCfg, err := engine.Apply(cfg, *from, *to)
	if err != nil {
		panic(err)
	}

	out, _ := json.MarshalIndent(newCfg, "", " ")
	fmt.Println(string(out))
}
