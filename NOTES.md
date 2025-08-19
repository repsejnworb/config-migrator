config-migrator/
├── go.mod
├── go.sum
├── cmd/
│ └── migrator/
│ └── main.go # CLI entrypoint
├── pkg/
│ └── migrator/
│ ├── engine.go # Migration engine
│ ├── resolver.go # Path resolution (slash notation + wildcards)
│ └── types.go # Types & DSL structs
├── migrations/
│ ├── v1_to_v2.json
│ └── v2_to_v1.json
└── schemas/
├── v1.json
└── v2.json



Non‑Reversible / Lossy Steps

Mark any step as non‑reversible to skip reverse generation:

{ "op": "set", "path": "a/b", "value": 42, "reversible": false }

If omitted, set and delete are treated as non‑reversible by default (engine won’t auto‑emit reverse for them). You can still hand‑author reverse migrations if you want.


