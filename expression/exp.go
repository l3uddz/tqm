package expression

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/l3uddz/tqm/config"
)

func Compile(clientName string, filter *config.FilterConfiguration) (*Expressions, error) {
	exprEnv := &config.Torrent{}
	exp := new(Expressions)

	// compile ignores
	for _, ignoreExpr := range filter.Ignore {
		program, err := expr.Compile(ignoreExpr, expr.Env(exprEnv), expr.AsBool())
		if err != nil {
			return nil, fmt.Errorf("compile ignore expression: %q: %w", ignoreExpr, err)
		}

		exp.Ignores = append(exp.Ignores, program)
	}

	// compile removes
	for _, removeExpr := range filter.Remove {
		program, err := expr.Compile(removeExpr, expr.Env(exprEnv), expr.AsBool())
		if err != nil {
			return nil, fmt.Errorf("compile remove expression: %q: %w", removeExpr, err)
		}

		exp.Removes = append(exp.Removes, program)
	}

	// compile labels
	exp.Labels = make(map[string]*LabelExpression, 0)
	for n, labelExpr := range filter.Label {
		le := new(LabelExpression)

		// compile ignores
		for _, ignoreExpr := range labelExpr.Ignore {
			program, err := expr.Compile(ignoreExpr, expr.Env(exprEnv), expr.AsBool())
			if err != nil {
				return nil, fmt.Errorf("compile label ignore expression: %v: %q: %w", n, ignoreExpr, err)
			}

			le.Ignores = append(le.Ignores, program)
		}

		// compile updates
		for _, updateExpr := range labelExpr.Update {
			program, err := expr.Compile(updateExpr, expr.Env(exprEnv), expr.AsBool())
			if err != nil {
				return nil, fmt.Errorf("compile label update expression: %v: %q: %w", n, updateExpr, err)
			}

			le.Updates = append(le.Updates, program)
		}

		exp.Labels[n] = le
	}

	return exp, nil
}
