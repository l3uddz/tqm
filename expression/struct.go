package expression

import "github.com/antonmedv/expr/vm"

type Expressions struct {
	Ignores []*vm.Program
	Removes []*vm.Program
	Labels  []*LabelExpression
}

type LabelExpression struct {
	Name    string
	Updates []*vm.Program
}
