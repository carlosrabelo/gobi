package repl

import (
	"fmt"
	"strings"

	"github.com/carlosrabelo/gobi/gobi/internal/context"
	"github.com/carlosrabelo/gobi/gobi/pkg/command"
	"github.com/carlosrabelo/gobi/gobi/pkg/expr"
)

func cmdCOUNT(ctx *context.Context, cmd command.Command) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	saved := a.Cursor
	n, err := scanCount(ctx, cmd)
	if err != nil {
		return err
	}
	a.Cursor = saved
	_ = a.LoadCurrent()
	if err := storePendingTO(ctx, cmd.Clauses["TO"], []float64{float64(n)}); err != nil {
		return err
	}
	if ctx.Config.Talk {
		fmt.Fprintln(ctx.Stdout, n)
	}
	return nil
}

func cmdSUM(ctx *context.Context, cmd command.Command) error {
	return sumOrAvg(ctx, cmd, false)
}

func cmdAVERAGE(ctx *context.Context, cmd command.Command) error {
	return sumOrAvg(ctx, cmd, true)
}

func sumOrAvg(ctx *context.Context, cmd command.Command, average bool) error {
	a := ctx.Active()
	if !a.HasTable() {
		return fail(ctx, "no database in use")
	}
	exprs, toNames, err := parseSumArgs(cmd)
	if err != nil {
		return fail(ctx, "%s", err.Error())
	}
	saved := a.Cursor
	sums, count, err := scanSums(ctx, cmd, exprs)
	if err != nil {
		return err
	}
	a.Cursor = saved
	_ = a.LoadCurrent()
	vals := make([]float64, len(sums))
	for i, s := range sums {
		if average {
			if count == 0 {
				vals[i] = 0
			} else {
				vals[i] = s / float64(count)
			}
		} else {
			vals[i] = s
		}
	}
	if len(toNames) > 0 {
		if err := storePendingTO(ctx, strings.Join(toNames, ","), vals); err != nil {
			return err
		}
	} else {
		ctx.PendingTO = nil
		ctx.PendingVals = vals
	}
	if ctx.Config.Talk {
		for i, v := range vals {
			if i > 0 {
				fmt.Fprint(ctx.Stdout, " ")
			}
			fmt.Fprint(ctx.Stdout, expr.NumberVal(v).String())
		}
		fmt.Fprintln(ctx.Stdout)
	}
	return nil
}

func storePendingTO(ctx *context.Context, toClause string, vals []float64) error {
	toClause = strings.TrimSpace(toClause)
	var names []string
	if toClause != "" {
		for _, p := range strings.Split(toClause, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				names = append(names, strings.ToUpper(p))
			}
		}
	}
	ctx.PendingTO = names
	ctx.PendingVals = vals
	if len(names) == 0 {
		return nil
	}
	if len(names) != len(vals) {
		return fail(ctx, "TO list length must match values")
	}
	for i, name := range names {
		if err := ctx.Symbols.Set(name, expr.NumberVal(vals[i])); err != nil {
			return fail(ctx, "%s", err.Error())
		}
	}
	return nil
}

func parseSumArgs(cmd command.Command) (exprs []string, toNames []string, err error) {
	args := strings.TrimSpace(cmd.Args)
	if args == "" {
		return nil, nil, errString("SUM/AVERAGE requires an expression")
	}
	exprs = splitCommaExprs(args)
	if len(exprs) == 0 {
		return nil, nil, errString("SUM/AVERAGE requires an expression")
	}
	to := strings.TrimSpace(cmd.Clauses["TO"])
	if to != "" {
		for _, p := range strings.Split(to, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				toNames = append(toNames, strings.ToUpper(p))
			}
		}
		if len(toNames) != len(exprs) {
			return nil, nil, errString("TO list length must match expressions")
		}
	}
	return exprs, toNames, nil
}

func scanCount(ctx *context.Context, cmd command.Command) (int, error) {
	return scanMatch(ctx, cmd, nil)
}

func scanSums(ctx *context.Context, cmd command.Command, exprSrcs []string) (sums []float64, count int, err error) {
	trees := make([]expr.Expression, len(exprSrcs))
	for i, s := range exprSrcs {
		tree, errs := expr.Parse(s)
		if len(errs) > 0 || tree == nil {
			msg := "invalid expression"
			if len(errs) > 0 {
				msg = errs[0]
			}
			return nil, 0, fail(ctx, "%s", msg)
		}
		trees[i] = tree
	}
	sums = make([]float64, len(trees))
	n, err := scanMatch(ctx, cmd, func(env expr.Env) error {
		for i, tree := range trees {
			v, err := expr.Eval(tree, env)
			if err != nil {
				return err
			}
			if v.Kind != expr.KindNumber {
				return fmt.Errorf("non-numeric SUM/AVERAGE result")
			}
			sums[i] += v.Num
		}
		return nil
	})
	return sums, n, err
}

func scanMatch(ctx *context.Context, cmd command.Command, each func(expr.Env) error) (int, error) {
	a := ctx.Active()
	forExpr, err := parseOptionalLogic(cmd.Clauses["FOR"])
	if err != nil {
		return 0, fail(ctx, "%s", err.Error())
	}
	whileExpr, err := parseOptionalLogic(cmd.Clauses["WHILE"])
	if err != nil {
		return 0, fail(ctx, "%s", err.Error())
	}
	env := newEnv(ctx)
	count := a.RecCount()
	start := 1
	if whileExpr != nil {
		start = a.Cursor
		if start < 1 {
			start = 1
		}
	}
	n := 0
	for i := start; i <= count; i++ {
		a.Cursor = i
		a.BOF = false
		if err := a.LoadCurrent(); err != nil {
			return 0, fail(ctx, "%s", err.Error())
		}
		if ctx.Config.Deleted && a.Record.Deleted {
			continue
		}
		if whileExpr != nil {
			ok, err := evalLogic(whileExpr, env)
			if err != nil {
				return 0, fail(ctx, "%s", err.Error())
			}
			if !ok {
				break
			}
		}
		if forExpr != nil {
			ok, err := evalLogic(forExpr, env)
			if err != nil {
				return 0, fail(ctx, "%s", err.Error())
			}
			if !ok {
				continue
			}
		}
		n++
		if each != nil {
			if err := each(env); err != nil {
				return 0, fail(ctx, "%s", err.Error())
			}
		}
	}
	return n, nil
}
