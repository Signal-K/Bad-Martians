package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

func main() {
	engines := []Factory{
		{"Ours (PBD lattice)", newOurs},
		{"Ours, 4x iterations", newOursTuned},
		{"Box2D (rigid + welds)", newBox2D},
		{"Chipmunk (impulse lattice)", newChipmunk},
	}
	type section struct {
		Name string
		Rows map[string]map[string]string // metric -> engine -> value
		Keys []string
	}
	var sections []*section
	add := func(name string, run func(Factory) Metrics) {
		sec := &section{Name: name, Rows: map[string]map[string]string{}}
		for _, e := range engines {
			for k, v := range run(e) {
				if sec.Rows[k] == nil {
					sec.Rows[k] = map[string]string{}
					sec.Keys = append(sec.Keys, k)
				}
				sec.Rows[k][e.Name] = v
			}
		}
		sort.Strings(sec.Keys)
		sections = append(sections, sec)
	}
	add("Settling on flat rock (12 s, no throttle)", settle)
	add("Soil slope climb (full throttle, 25 s)", slope)
	add("Drop from 30 m, launched at 14 m/s", func(f Factory) Metrics { return highspeed(f, 30) })
	add("Drop from 150 m, launched at 14 m/s", func(f Factory) Metrics { return highspeed(f, 150) })
	add("Drop from 1000 m (about 86 m/s on impact)", func(f Factory) Metrics { return highspeed(f, 1000) })
	add("Shipped levels (par is the 3-star time)", levels)
	add("Long beam rigidity (10 m, 12 s at rest)", rigidity)
	add("Cost per step (this machine)", perf)
	add("Determinism (12° soil ramp, 20 s, 1200 steps)", func(f Factory) Metrics { m, _ := determinism(f); return m })

	var md strings.Builder
	fmt.Fprintf(&md, "platform: %s/%s, %s\n\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	for _, s := range sections {
		fmt.Fprintf(&md, "### %s\n\n| metric |", s.Name)
		for _, e := range engines {
			fmt.Fprintf(&md, " %s |", e.Name)
		}
		md.WriteString("\n|---|")
		for range engines {
			md.WriteString("---|")
		}
		md.WriteString("\n")
		for _, k := range s.Keys {
			fmt.Fprintf(&md, "| %s |", k)
			for _, e := range engines {
				fmt.Fprintf(&md, " %s |", s.Rows[k][e.Name])
			}
			md.WriteString("\n")
		}
		md.WriteString("\n")
	}
	fmt.Print(md.String())
	if len(os.Args) > 1 {
		b, _ := json.MarshalIndent(sections, "", " ")
		os.WriteFile(os.Args[1], b, 0o644)
	}
}
