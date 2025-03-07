package provisioning

import (
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/process/provisioning/automock"
	"github.com/stretchr/testify/assert"
)

func TestSkipForTrialPlanStepShouldSkip(t *testing.T) {
	// Given
	wantSkipTime := time.Duration(0)
	wantOperation := fixOperationWithPlanID(broker.TrialPlanID)

	mockStep := new(automock.Step)
	mockStep.On("Name").Return("Test")
	skipStep := NewSkipForTrialPlanStep(mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := skipStep.Run(wantOperation, fixLogger())

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation, gotOperation)
}

func TestSkipForTrialPlanStepShouldNotSkip(t *testing.T) {
	// Given
	wantSkipTime := time.Duration(10)
	givenOperation1 := fixOperationWithPlanID("operation1")
	wantOperation2 := fixOperationWithPlanID("operation2")

	mockStep := new(automock.Step)
	mockStep.On("Run", givenOperation1, fixLogger()).Return(wantOperation2, wantSkipTime, nil)
	skipStep := NewSkipForTrialPlanStep(mockStep)

	// When
	gotOperation, gotSkipTime, gotErr := skipStep.Run(givenOperation1, fixLogger())

	// Then
	mockStep.AssertExpectations(t)
	assert.Nil(t, gotErr)
	assert.Equal(t, wantSkipTime, gotSkipTime)
	assert.Equal(t, wantOperation2, gotOperation)
}
