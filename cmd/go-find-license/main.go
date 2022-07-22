package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/berquerant/go-find-license/internal"
)

var (
	debug          = flag.Bool("debug", false, "Enable debug logs.")
	dry            = flag.Bool("dry", false, "Without searching licenses from pkg.go.dev.")
	useList        = flag.Bool("list", false, "Use go list.")
	ignoreIndirect = flag.Bool("direct", false, "Ignore indirect dependencies.")
)

const usage = `Usage of go-find-license:
  go-find-license [flags]

Search pkg.go.dev and display the licenses for the dependencies of the module,
like:

{
  "Content": CONTENT,
  "Type": TYPE,
  "URI": URI,
  "Source": SOURCE,
  "Module": {
    "Path": PATH,
    "Version": VERSION,
    "Indirect": INDIRECT,
    "Error": ERROR
  },
  "Err": ERR
}

CONTENT: string
  license text
TYPE: string
  type of license, e.g. MIT
URI: string
  scraped URI
SOURCE: string
  license file path
PATH: string
  module path
VERSION: string
  module version
INDIRECT: bool
  indirect dependency or not
ERROR: any
  loading module error
ERR: string
  scraping error

Flags:`

func Usage() {
	fmt.Fprintln(os.Stderr, usage)
	flag.PrintDefaults()
}

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func loader() internal.Loader {
	if *useList {
		return &internal.GoListLoader{}
	}
	return &internal.GoModLoader{}
}

func filterModules(ms internal.Modules) internal.Modules {
	if *ignoreIndirect {
		ms = ms.RemoveIndirect()
	}
	ms.ForEach(func(x *internal.Module) {
		if x.Error != nil {
			b, _ := json.Marshal(x.Error)
			internal.Infof("Got error from %s %s %s", x.Path, x.Version, b)
		}
	})
	return ms.RemoveError()
}

func main() {
	flag.Usage = Usage
	flag.Parse()
	if *debug {
		internal.EnableDebug()
	}

	modules, err := loader().Load()
	fail(err)
	modules = filterModules(modules)

	if *dry {
		b, err := json.Marshal(modules)
		fail(err)
		fmt.Printf("%s\n", b)
		return
	}

	for license := range internal.NewFetcher().FetchLicenses(context.Background(), modules) {
		d := map[string]any{
			"Module": license.Module(),
			"URI":    license.URI(),
		}
		if err := license.Err(); err != nil {
			d["Err"] = err.Error()
		} else {
			d["Source"] = license.Source()
			d["Content"] = license.Content()
			d["Type"] = license.Type()
		}
		b, err := json.Marshal(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			continue
		}
		fmt.Printf("%s\n", b)
	}
}
