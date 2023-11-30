package repl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
	"github.com/carlosrabelo/gobi/gobi/pkg/help"
)

func init() {
	help.Register("CLEAR", "CLEAR / CLEAR FIELDS — clear screen or field list.")
}

func setRelation(ctx *context.Context, rest string) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		a.RelExpr = ""
		a.RelAlias = ""
		return nil
	}
	up := strings.ToUpper(rest)
	i := strings.Index(up, " INTO ")
	if i < 0 {
		return fail(ctx, "SET RELATION requires INTO")
	}
	exprSrc := strings.TrimSpace(rest[:i])
	alias := strings.TrimSpace(rest[i+6:])
	if exprSrc == "" || alias == "" {
		return fail(ctx, "SET RELATION requires INTO")
	}
	other := ctx.AreaByAlias(alias)
	if other == nil || !other.HasTable() {
		return fail(ctx, "alias not open: %s", alias)
	}
	if controllingIndex(other) == "" {
		return fail(ctx, "related area has no controlling index")
	}
	a.RelExpr = exprSrc
	a.RelAlias = strings.ToUpper(alias)
	return applyRelation(ctx)
}

func applyRelation(ctx *context.Context) error {
	a := ctx.Active()
	if a.RelExpr == "" || a.RelAlias == "" {
		return nil
	}
	other := ctx.AreaByAlias(a.RelAlias)
	if other == nil || !other.HasTable() {
		return nil
	}
	tree, errs := expr.Parse(a.RelExpr)
	if len(errs) > 0 || tree == nil {
		return fail(ctx, "invalid relation expression")
	}
	v, err := expr.Eval(tree, newEnv(ctx))
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	saved := ctx.ActiveArea
	// seek in other area
	for i := 1; i <= context.AreaCount; i++ {
		if ctx.Areas[i] == other {
			ctx.ActiveArea = i
			break
		}
	}
	path := controllingIndex(other)
	key, _, err := valueToKey(v)
	if err != nil {
		ctx.ActiveArea = saved
		return fail(ctx, "%s", err.Error())
	}
	err = seekKey(ctx, path, key, false)
	ctx.ActiveArea = saved
	if err != nil {
		// leave at EOF
		other.Cursor = other.RecCount() + 1
		other.Found = false
		_ = other.LoadCurrent()
	}
	return nil
}

func setFilter(ctx *context.Context, exprSrc string) error {
	a := ctx.Active()
	if strings.EqualFold(strings.TrimSpace(exprSrc), "FILE") ||
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(exprSrc)), "FILE ") {
		return fail(ctx, "SET FILTER TO FILE is not available in Gobi")
	}
	a.FilterExpr = strings.TrimSpace(exprSrc)
	return nil
}

func recordPassesFilter(ctx *context.Context, a *context.Area) (bool, error) {
	if a.FilterExpr == "" {
		return true, nil
	}
	tree, errs := expr.Parse(a.FilterExpr)
	if len(errs) > 0 || tree == nil {
		return false, fmt.Errorf("invalid filter")
	}
	ok, err := evalLogic(tree, newEnv(ctx))
	return ok, err
}

func setFieldsList(ctx *context.Context, rest string) error {
	a := ctx.Active()
	rest = strings.TrimSpace(rest)
	if rest == "" {
		a.FieldList = nil
		return nil
	}
	if strings.EqualFold(rest, "ALL") {
		a.FieldList = nil
		return nil
	}
	parts := strings.Split(rest, ",")
	var list []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			list = append(list, strings.ToUpper(p))
		}
	}
	a.FieldList = list
	return nil
}

func setFieldsOnOff(ctx *context.Context, on bool) error {
	ctx.Active().FieldsOn = on
	ctx.Config.Fields = on
	return nil
}

func clearFields(ctx *context.Context) error {
	for i := 1; i <= context.AreaCount; i++ {
		ctx.Areas[i].FieldList = nil
		ctx.Areas[i].FieldsOn = false
	}
	ctx.Config.Fields = false
	return nil
}

func setOrder(ctx *context.Context, n int) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	if n < 0 || n > 7 {
		return fail(ctx, "SET ORDER out of range")
	}
	if n == 0 {
		a.IndexOrder = nil
		return nil
	}
	path := a.NDX[n-1]
	if path == "" {
		return fail(ctx, "no index in slot %d", n)
	}
	// move controlling index to slot 0 by rotating
	if n != 1 {
		a.NDX[0], a.NDX[n-1] = a.NDX[n-1], a.NDX[0]
	}
	return rebuildIndexOrder(ctx)
}

func setUnique(ctx *context.Context, on bool) error {
	ctx.Config.Unique = on
	return nil
}

func parseOrderN(to string, args []string) (int, error) {
	s := strings.TrimSpace(to)
	if s == "" && len(args) >= 2 && strings.EqualFold(args[1], "TO") {
		if len(args) >= 3 {
			s = args[2]
		}
	}
	if s == "" && len(args) >= 2 {
		s = args[1]
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid ORDER")
	}
	return n, nil
}

// hookAfterMove runs relation after navigation.
func hookAfterMove(ctx *context.Context) error {
	return applyRelation(ctx)
}
