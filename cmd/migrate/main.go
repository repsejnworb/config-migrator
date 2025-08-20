package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "strings"

    "github.com/repsejnworb/config-migrator/pkg/migrate"
)

func main() {
    migrationsDir := flag.String("migrations", "./migrations", "directory containing forward migration JSON files")
    from := flag.String("from", "", "source version")
    to := flag.String("to", "", "target version")
    in := flag.String("in", "", "input config JSON file")
    out := flag.String("out", "-", "output file ('-' for stdout)")
    pretty := flag.Bool("pretty", true, "pretty-print JSON")
    flag.Parse()

    if *from == "" || *to == "" || *in == "" {
        fmt.Println("Usage: migrator --migrations ./migrations --from 1.0 --to 2.0 --in ./examples/v1_config.json [--out -] [--pretty]")
        os.Exit(1)
    }

    eng := migrator.NewEngine()
    if err := eng.LoadAll(*migrationsDir); err != nil { panic(err) }

    raw, err := os.ReadFile(*in)
    if err != nil { panic(err) }
    var cfg map[string]interface{}
    if err := json.Unmarshal(raw, &cfg); err != nil { panic(err) }

    outCfg, err := eng.Apply(cfg, *from, *to)
    if err != nil { panic(err) }

    var enc []byte
    if *pretty { enc, _ = json.MarshalIndent(outCfg, "", "  ") } else { enc, _ = json.Marshal(outCfg) }

    if *out == "-" {
        os.Stdout.Write(enc)
        if *pretty { os.Stdout.WriteString("
") }
        return
    }
    if err := os.WriteFile(*out, enc, 0o644); err != nil { panic(err) }
    fmt.Fprintln(os.Stderr, "wrote", strings.TrimSpace(*out))
}