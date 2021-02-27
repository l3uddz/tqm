package expression

import "github.com/antonmedv/expr/vm"

type Expressions struct {
	Ignores []*vm.Program
	Removes []*vm.Program
	Labels  map[string]*LabelExpression
}

type LabelExpression struct {
	Ignores []*vm.Program
	Updates []*vm.Program
}
