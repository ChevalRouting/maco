package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/m-vinc/maco/pkg/db/generated"
	"github.com/m-vinc/maco/pkg/types"
)

func vmState(row generated.VmState) types.VMState {
	return types.VMState{
		ID:       row.ID,
		Phase:    row.Phase,
		PID:      int(row.Pid),
		BootTime: row.BootTime,
		SeenAt:   row.SeenAt,
	}
}

func (db *DB) SetVMState(ctx context.Context, state types.VMState) error {
	return db.queries.UpsertVMState(ctx, generated.UpsertVMStateParams{
		ID:       state.ID,
		Phase:    state.Phase,
		Pid:      int64(state.PID),
		BootTime: state.BootTime,
		SeenAt:   state.SeenAt,
	})
}

func (db *DB) GetVMState(ctx context.Context, id string) (types.VMState, bool, error) {
	row, err := db.queries.GetVMState(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return types.VMState{}, false, nil
	}

	if err != nil {
		return types.VMState{}, false, err
	}

	return vmState(row), true, nil
}

func (db *DB) ListVMStates(ctx context.Context) (map[string]types.VMState, error) {
	rows, err := db.queries.ListVMStates(ctx)
	if err != nil {
		return nil, err
	}

	states := make(map[string]types.VMState, len(rows))
	for _, row := range rows {
		states[row.ID] = vmState(row)
	}

	return states, nil
}

func (db *DB) DeleteVMState(ctx context.Context, id string) error {
	return db.queries.DeleteVMState(ctx, id)
}
