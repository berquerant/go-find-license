# go-find-license

```
❯ go-find-license -h
Usage of go-find-license:
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

Flags:
  -debug
        Enable debug logs.
  -direct
        Ignore indirect dependencies.
  -dry
        Without searching licenses from pkg.go.dev.
  -list
        Use go list.
```
