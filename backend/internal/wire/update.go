package wire

import (
	"bjoernblessin.de/go-utils/util/assert"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/update"
)

// stages maps this app's stages onto the contract's.
//
// A table rather than a switch, so a stage added without a wire value fails at the conversion
// rather than crossing as unspecified.
var stages = map[update.Stage]mirrormev1.UpdateStage{
	update.StageOff:       mirrormev1.UpdateStage_UPDATE_STAGE_OFF,
	update.StageUnchecked: mirrormev1.UpdateStage_UPDATE_STAGE_UNCHECKED,
	update.StageChecking:  mirrormev1.UpdateStage_UPDATE_STAGE_CHECKING,
	update.StageCurrent:   mirrormev1.UpdateStage_UPDATE_STAGE_CURRENT,
	update.StageAvailable: mirrormev1.UpdateStage_UPDATE_STAGE_AVAILABLE,
	update.StageFetching:  mirrormev1.UpdateStage_UPDATE_STAGE_FETCHING,
	update.StageReady:     mirrormev1.UpdateStage_UPDATE_STAGE_READY,
	update.StageFailed:    mirrormev1.UpdateStage_UPDATE_STAGE_FAILED,
}

// UpdateState converts what this install knows about the published release.
func UpdateState(state update.State) *mirrormev1.UpdateState {
	stage, declared := stages[state.Stage]
	assert.Assert(declared, "every stage an install reaches crosses the contract", int(state.Stage))
	assert.Assert(state.Percent >= 0 && state.Percent <= 100,
		"a download crosses as a fraction of itself", state.Percent)

	return &mirrormev1.UpdateState{
		Stage:         stage,
		Running:       state.Running,
		Latest:        state.Latest,
		Page:          state.Page,
		Percent:       int32(state.Percent),
		Unchecked:     state.Unchecked,
		Uninstallable: state.Uninstallable,
		Failure:       state.Failure,
		Detail:        state.Detail,
	}
}

// UpdateStateEvent announces what this install knows about the published release, whole.
//
// On every step of a check and as a download fills in,
// so a second window learns that a release is staged without having asked for one.
func UpdateStateEvent(state update.State) *mirrormev1.Event {
	return &mirrormev1.Event{
		Payload: &mirrormev1.Event_UpdateState{UpdateState: UpdateState(state)},
	}
}
