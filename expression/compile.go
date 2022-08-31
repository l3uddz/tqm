package expression

import (
	"fmt"

	"github.com/antonmedv/expr"

	"github.com/l3uddz/tqm/config"
)

func Compile(filter *config.FilterConfiguration) (*Expressions, error) {
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
	for _, labelExpr := range filter.Label {
		le := &LabelExpression{Name: labelExpr.Name}

		// compile updates
		for _, updateExpr := range labelExpr.Update {
			program, err := expr.Compile(updateExpr, expr.Env(exprEnv), expr.AsBool())
			if err != nil {
				return nil, fmt.Errorf("compile label update expression: %v: %q: %w", labelExpr.Name, updateExpr, err)
			}

			le.Updates = append(le.Updates, program)
		}

		exp.Labels = append(exp.Labels, le)
	}

	// compile tags
	for _, tagExpr := range filter.Tag {
		le := &TagExpression{Name: tagExpr.Name, Mode: tagExpr.Mode}

		// compile updates
		for _, updateExpr := range tagExpr.Update {
			program, err := expr.Compile(updateExpr, expr.Env(exprEnv), expr.AsBool())
			if err != nil {
				return nil, fmt.Errorf("compile tag update expression: %v: %q: %w", tagExpr.Name, updateExpr, err)
			}

			le.Updates = append(le.Updates, program)
		}

		exp.Tags = append(exp.Tags, le)
	}

	return exp, nil
}
