package repl

import (
	"os"
	"sort"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/ndx"
)

func cmdSETINDEX(ctx *context.Context, list string) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	ctx.CloseIndexes()
	a.IndexOrder = nil
	a.IndexPos = 0
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	for i, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		path, err := ndxPath(ctx, name)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		if !fileExists(path) {
			_ = ctx.CloseArea(ctx.ActiveArea)
			return fail(ctx, "index not found: %s", path)
		}
		if i < len(a.NDX) {
			a.NDX[i] = path
		}
	}
	return rebuildIndexOrder(ctx)
}

func rebuildIndexOrder(ctx *context.Context) error {
	a := ctx.Active()
	path := a.NDX[0]
	if path == "" {
		a.IndexOrder = nil
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	defer f.Close()
	idx, err := ndx.Open(f)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	tree, errs := expr.Parse(idx.Header().Expr)
	if len(errs) > 0 || tree == nil {
		return fail(ctx, "invalid index expression")
	}
	type pair struct {
		key string
		rec int
	}
	var pairs []pair
	saved := a.Cursor
	env := newEnv(ctx)
	for n := 1; n <= a.RecCount(); n++ {
		a.Cursor = n
		if err := a.LoadCurrent(); err != nil {
			return fail(ctx, "%s", err.Error())
		}
		v, err := expr.Eval(tree, env)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		k, _, err := valueToKey(v)
		if err != nil {
			return fail(ctx, "%s", err.Error())
		}
		pairs = append(pairs, pair{key: string(k), rec: n})
	}
	a.Cursor = saved
	_ = a.LoadCurrent()
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})
	a.IndexOrder = make([]int, len(pairs))
	for i, p := range pairs {
		a.IndexOrder[i] = p.rec
	}
	a.IndexPos = 0
	for i, r := range a.IndexOrder {
		if r == a.Cursor {
			a.IndexPos = i
			break
		}
	}
	return nil
}

func parseUseIndexes(args string) (name, alias string, indexes []string, err error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", nil, nil
	}
	tokens := strings.Fields(args)
	name = tokens[0]
	up := strings.ToUpper(args)
	if j := strings.Index(up, " ALIAS "); j >= 0 {
		rest := strings.TrimSpace(args[j+7:])
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return "", "", nil, errUseAliasMissing
		}
		alias = fields[0]
	}
	if j := strings.Index(up, " INDEX "); j >= 0 {
		rest := strings.TrimSpace(args[j+7:])
		if k := strings.Index(strings.ToUpper(rest), " ALIAS "); k >= 0 {
			rest = rest[:k]
		}
		for _, p := range strings.Split(rest, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				indexes = append(indexes, p)
			}
		}
	}
	return name, alias, indexes, nil
}
