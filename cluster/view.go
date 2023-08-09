//go:generate mockgen -source=view.go -destination=../../../internal/mock/cluster/view/view.go -package=mock_view

package cluster

import (
	api "github.com/wetware/ww/api/cluster"
)

type (
	SelectorStruct   = api.View_Selector
	ConstraintStruct = api.View_Constraint

	Selector   func(SelectorStruct) error
	Constraint func(ConstraintStruct) error
)

type QueryParams interface {
	NewSelector() (api.View_Selector, error)
	NewConstraints(int32) (api.View_Constraint_List, error)
}

type Query func(QueryParams) error

func NewQuery(s Selector, cs ...Constraint) Query {
	return func(ps QueryParams) error {
		if err := bindSelector(s, ps); err != nil {
			return err
		}

		return bindConstraints(cs, ps)
	}
}

func bindSelector(s Selector, ps QueryParams) error {
	sel, err := ps.NewSelector()
	if err != nil {
		return err
	}

	return s(sel)
}

func bindConstraints(cs []Constraint, ps QueryParams) error {
	constraint, err := ps.NewConstraints(int32(len(cs)))
	if err != nil {
		return err
	}

	for i, bind := range cs {
		if err = bind(constraint.At(i)); err != nil {
			break
		}
	}

	return err
}
