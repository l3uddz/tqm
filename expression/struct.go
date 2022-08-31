package expression

import "github.com/antonmedv/expr/vm"

type Expressions struct {
	Ignores []*vm.Program
	Removes []*vm.Program
	Labels  []*LabelExpression
	Tags    []*TagExpression
}

type LabelExpression struct {
	Name    string
	Updates []*vm.Program
}

type TagExpression struct {
	Name    string
	Mode    string
	Updates []*vm.Program
}
