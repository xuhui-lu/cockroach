// Copyright 2019 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// {{/*
// +build execgen_template
//
// This file is the execgen template for select_in.eg.go. It's formatted in a
// special way, so it's both valid Go and a valid text/template input. This
// permits editing this file with editor support.
//
// */}}

package colexec

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/col/coldata"
	"github.com/cockroachdb/cockroach/pkg/col/typeconv"
	"github.com/cockroachdb/cockroach/pkg/sql/colexec/execgen"
	"github.com/cockroachdb/cockroach/pkg/sql/colexecbase"
	"github.com/cockroachdb/cockroach/pkg/sql/colexecbase/colexecerror"
	"github.com/cockroachdb/cockroach/pkg/sql/colmem"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/types"
	"github.com/cockroachdb/errors"
)

// Remove unused warnings.
var (
	_ = colexecerror.InternalError
)

// {{/*

type _GOTYPESLICE interface{}
type _GOTYPE interface{}
type _TYPE interface{}

// _CANONICAL_TYPE_FAMILY is the template variable.
const _CANONICAL_TYPE_FAMILY = types.UnknownFamily

// _TYPE_WIDTH is the template variable.
const _TYPE_WIDTH = 0

func _COMPARE(_, _, _, _, _ string) bool {
	colexecerror.InternalError("")
}

// */}}

// Enum used to represent comparison results.
type comparisonResult int

const (
	siTrue comparisonResult = iota
	siFalse
	siNull
)

func GetInProjectionOperator(
	allocator *colmem.Allocator,
	t *types.T,
	input colexecbase.Operator,
	colIdx int,
	resultIdx int,
	datumTuple *tree.DTuple,
	negate bool,
) (colexecbase.Operator, error) {
	input = newVectorTypeEnforcer(allocator, input, types.Bool, resultIdx)
	switch typeconv.TypeFamilyToCanonicalTypeFamily(t.Family()) {
	// {{range .}}
	case _CANONICAL_TYPE_FAMILY:
		switch t.Width() {
		// {{range .WidthOverloads}}
		case _TYPE_WIDTH:
			obj := &projectInOp_TYPE{
				OneInputNode: NewOneInputNode(input),
				allocator:    allocator,
				colIdx:       colIdx,
				outputIdx:    resultIdx,
				negate:       negate,
			}
			obj.filterRow, obj.hasNulls = fillDatumRow_TYPE(t, datumTuple)
			return obj, nil
			// {{end}}
		}
		// {{end}}
	}
	return nil, errors.Errorf("unhandled type: %s", t.Name())
}

func GetInOperator(
	t *types.T, input colexecbase.Operator, colIdx int, datumTuple *tree.DTuple, negate bool,
) (colexecbase.Operator, error) {
	switch typeconv.TypeFamilyToCanonicalTypeFamily(t.Family()) {
	// {{range .}}
	case _CANONICAL_TYPE_FAMILY:
		switch t.Width() {
		// {{range .WidthOverloads}}
		case _TYPE_WIDTH:
			obj := &selectInOp_TYPE{
				OneInputNode: NewOneInputNode(input),
				colIdx:       colIdx,
				negate:       negate,
			}
			obj.filterRow, obj.hasNulls = fillDatumRow_TYPE(t, datumTuple)
			return obj, nil
			// {{end}}
		}
		// {{end}}
	}
	return nil, errors.Errorf("unhandled type: %s", t.Name())
}

// {{range .}}
// {{range .WidthOverloads}}

type selectInOp_TYPE struct {
	OneInputNode
	colIdx    int
	filterRow []_GOTYPE
	hasNulls  bool
	negate    bool
}

var _ colexecbase.Operator = &selectInOp_TYPE{}

type projectInOp_TYPE struct {
	OneInputNode
	allocator *colmem.Allocator
	colIdx    int
	outputIdx int
	filterRow []_GOTYPE
	hasNulls  bool
	negate    bool
}

var _ colexecbase.Operator = &projectInOp_TYPE{}

func fillDatumRow_TYPE(t *types.T, datumTuple *tree.DTuple) ([]_GOTYPE, bool) {
	conv := GetDatumToPhysicalFn(t)
	var result []_GOTYPE
	hasNulls := false
	for _, d := range datumTuple.D {
		if d == tree.DNull {
			hasNulls = true
		} else {
			convRaw := conv(d)
			converted := convRaw.(_GOTYPE)
			result = append(result, converted)
		}
	}
	return result, hasNulls
}

func cmpIn_TYPE(
	targetElem _GOTYPE, targetCol _GOTYPESLICE, filterRow []_GOTYPE, hasNulls bool,
) comparisonResult {
	// Filter row input is already sorted due to normalization, so we can use a
	// binary search right away.
	lo := 0
	hi := len(filterRow)
	for lo < hi {
		i := (lo + hi) / 2
		var cmpResult int
		_COMPARE(cmpResult, targetElem, filterRow[i], targetCol, _)
		if cmpResult == 0 {
			return siTrue
		} else if cmpResult > 0 {
			lo = i + 1
		} else {
			hi = i
		}
	}

	if hasNulls {
		return siNull
	} else {
		return siFalse
	}
}

func (si *selectInOp_TYPE) Init() {
	si.input.Init()
}

func (pi *projectInOp_TYPE) Init() {
	pi.input.Init()
}

func (si *selectInOp_TYPE) Next(ctx context.Context) coldata.Batch {
	for {
		batch := si.input.Next(ctx)
		if batch.Length() == 0 {
			return coldata.ZeroBatch
		}

		vec := batch.ColVec(si.colIdx)
		col := vec.TemplateType()
		var idx int
		n := batch.Length()

		compVal := siTrue
		if si.negate {
			compVal = siFalse
		}

		if vec.MaybeHasNulls() {
			nulls := vec.Nulls()
			if sel := batch.Selection(); sel != nil {
				sel = sel[:n]
				for _, i := range sel {
					v := col.Get(i)
					if !nulls.NullAt(i) && cmpIn_TYPE(v, col, si.filterRow, si.hasNulls) == compVal {
						sel[idx] = i
						idx++
					}
				}
			} else {
				batch.SetSelection(true)
				sel := batch.Selection()
				_ = col.Get(n - 1)
				for i := 0; i < n; i++ {
					v := col.Get(i)
					if !nulls.NullAt(i) && cmpIn_TYPE(v, col, si.filterRow, si.hasNulls) == compVal {
						sel[idx] = i
						idx++
					}
				}
			}
		} else {
			if sel := batch.Selection(); sel != nil {
				sel = sel[:n]
				for _, i := range sel {
					v := col.Get(i)
					if cmpIn_TYPE(v, col, si.filterRow, si.hasNulls) == compVal {
						sel[idx] = i
						idx++
					}
				}
			} else {
				batch.SetSelection(true)
				sel := batch.Selection()
				_ = col.Get(n - 1)
				for i := 0; i < n; i++ {
					v := col.Get(i)
					if cmpIn_TYPE(v, col, si.filterRow, si.hasNulls) == compVal {
						sel[idx] = i
						idx++
					}
				}
			}
		}

		if idx > 0 {
			batch.SetLength(idx)
			return batch
		}
	}
}

func (pi *projectInOp_TYPE) Next(ctx context.Context) coldata.Batch {
	batch := pi.input.Next(ctx)
	if batch.Length() == 0 {
		return coldata.ZeroBatch
	}

	vec := batch.ColVec(pi.colIdx)
	col := vec.TemplateType()

	projVec := batch.ColVec(pi.outputIdx)
	projCol := projVec.Bool()
	projNulls := projVec.Nulls()
	if projVec.MaybeHasNulls() {
		// We need to make sure that there are no left over null values in the
		// output vector.
		projNulls.UnsetNulls()
	}

	n := batch.Length()

	cmpVal := siTrue
	if pi.negate {
		cmpVal = siFalse
	}

	if vec.MaybeHasNulls() {
		nulls := vec.Nulls()
		if sel := batch.Selection(); sel != nil {
			sel = sel[:n]
			for _, i := range sel {
				if nulls.NullAt(i) {
					projNulls.SetNull(i)
				} else {
					v := col.Get(i)
					cmpRes := cmpIn_TYPE(v, col, pi.filterRow, pi.hasNulls)
					if cmpRes == siNull {
						projNulls.SetNull(i)
					} else {
						projCol[i] = cmpRes == cmpVal
					}
				}
			}
		} else {
			col = execgen.SLICE(col, 0, n)
			for i := 0; i < n; i++ {
				if nulls.NullAt(i) {
					projNulls.SetNull(i)
				} else {
					v := col.Get(i)
					cmpRes := cmpIn_TYPE(v, col, pi.filterRow, pi.hasNulls)
					if cmpRes == siNull {
						projNulls.SetNull(i)
					} else {
						projCol[i] = cmpRes == cmpVal
					}
				}
			}
		}
	} else {
		if sel := batch.Selection(); sel != nil {
			sel = sel[:n]
			for _, i := range sel {
				v := col.Get(i)
				cmpRes := cmpIn_TYPE(v, col, pi.filterRow, pi.hasNulls)
				if cmpRes == siNull {
					projNulls.SetNull(i)
				} else {
					projCol[i] = cmpRes == cmpVal
				}
			}
		} else {
			col = execgen.SLICE(col, 0, n)
			for i := 0; i < n; i++ {
				v := col.Get(i)
				cmpRes := cmpIn_TYPE(v, col, pi.filterRow, pi.hasNulls)
				if cmpRes == siNull {
					projNulls.SetNull(i)
				} else {
					projCol[i] = cmpRes == cmpVal
				}
			}
		}
	}
	return batch
}

// {{end}}
// {{end}}
