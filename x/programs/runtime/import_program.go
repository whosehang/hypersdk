// Copyright (C) 2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package runtime

import (
	"context"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/near/borsh-go"
)

const (
	callProgramCost = 10000
	setResultCost   = 10000
)

type callProgramInput struct {
	ProgramID    ids.ID
	FunctionName string
	Params       []byte
	Fuel         uint64
}

func NewProgramModule(r *WasmRuntime) *ImportModule {
	return &ImportModule{
		Name: "program",
		HostFunctions: map[string]HostFunction{
			"call_program": {FuelCost: callProgramCost, Function: FunctionWithOutput(func(callInfo *CallInfo, input []byte) ([]byte, error) {
				newInfo := *callInfo
				parsedInput := &callProgramInput{}
				if err := borsh.Deserialize(parsedInput, input); err != nil {
					return nil, err
				}

				// make sure there is enough fuel in current store to give to the new call
				if callInfo.RemainingFuel() < parsedInput.Fuel {
					return nil, errors.New("remaining fuel is less than requested fuel")
				}

				newInfo.ProgramID = parsedInput.ProgramID
				newInfo.FunctionName = parsedInput.FunctionName
				newInfo.Params = parsedInput.Params
				newInfo.Fuel = parsedInput.Fuel

				result, err := r.CallProgram(
					context.Background(),
					&newInfo)
				if err != nil {
					return nil, err
				}

				// subtract the fuel used during this call from the calling program
				remainingFuel := newInfo.RemainingFuel()
				if err := callInfo.ConsumeFuel(parsedInput.Fuel - remainingFuel); err != nil {
					return nil, err
				}

				return result, nil
			})},
			"set_call_result": {FuelCost: setResultCost, Function: FunctionNoOutput(func(callInfo *CallInfo, input []byte) error {
				callInfo.inst.result = input
				return nil
			})},
		},
	}
}
