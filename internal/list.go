package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
)

type Module struct {
	Path     string // module path
	Version  string // module version
	Indirect bool   // indirect dependency?
	Error    any    // error on loading module
}

func (m *Module) String() string {
	return fmt.Sprintf("%s,%s,%v", m.Path, m.Version, m.Error)
}

type Modules []*Module

func (ms Modules) ForEach(f func(*Module)) {
	for _, module := range ms {
		f(module)
	}
}

func (ms Modules) Filter(p func(*Module) bool) Modules {
	r := []*Module{}
	for _, module := range ms {
		if p(module) {
			r = append(r, module)
		}
	}
	return r
}

func (ms Modules) RemoveError() Modules {
	return ms.Filter(func(x *Module) bool {
		return x.Error == nil
	})
}

func (ms Modules) RemoveIndirect() Modules {
	return ms.Filter(func(x *Module) bool {
		return !x.Indirect
	})
}

type Loader interface {
	// Load loads and returns the Go modules.
	Load() (Modules, error)
}

// GoListLoader requires go list command.
type GoListLoader struct{}

func (*GoListLoader) Load() (Modules, error) {
	Debugf("Load modules from go list")

	out, err := exec.Command("go", "list", "-m", "-json", "all").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to go list %w", err)
	}

	arrayStr := fmt.Sprintf("[%s]", strings.ReplaceAll(
		strings.NewReplacer(
			"\r", "",
			"\r\n", "",
			"\n", "",
		).Replace(string(out)),
		"}{",
		"},{"))

	var modules []*Module
	if err := json.Unmarshal([]byte(arrayStr), &modules); err != nil {
		return nil, fmt.Errorf("failed to load modules: %w", err)
	}

	Debugf("%d modules loaded (go list %d bytes)", len(modules), len([]byte(arrayStr)))
	return modules, nil
}

// GoModLoader requires go.mod file.
type GoModLoader struct{}

func (*GoModLoader) Load() (Modules, error) {
	Debugf("Load modules from go.mod")

	f, err := os.Open("go.mod")
	if err != nil {
		return nil, fmt.Errorf("failed to open mod file %w", err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read mod file %w", err)
	}
	mf, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mod file %w", err)
	}

	requires := make([]*Module, len(mf.Require))
	for i, require := range mf.Require {
		requires[i] = &Module{
			Path:     require.Mod.Path,
			Version:  require.Mod.Version,
			Indirect: require.Indirect,
		}
	}

	Debugf("%d modules loaded (go.mod %d bytes)", len(requires), len(b))
	return requires, nil
}
