package test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_E2E_Suspension(t *testing.T) {
	ts := newTestSuite(t)
	if ts.IsDummyTest {
		return
	}
	if ts.IsUpgradeTest {
		return
	}
	if ts.IsCleanupPhase {
		ts.Cleanup()
		return
	}
	configMap := ts.testConfigMap()
	dashbaordPattern := fmt.Sprintf("%s/\\?kubeconfigID=[0-9a-f\\-]{36}", ts.BusolaURL)

	ts.log.Info("Starting e2e suspension test")
	operationID, err := ts.brokerClient.ProvisionRuntime("")
	require.NoError(t, err)

	ts.log.Info(fmt.Sprintf("Creating config map %s with test data", ts.ConfigName))
	err = ts.configMapClient.Create(configMap)
	require.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded(operationID, ts.ProvisionTimeout)
	require.NoError(t, err)

	dashboardURL, err := ts.brokerClient.FetchDashboardURL()
	require.NoError(t, err)

	ts.log.Info(fmt.Sprintf("Updating config map %s with dashboardUrl", ts.ConfigName))
	configMap.Data[dashboardUrlKey] = dashboardURL
	err = ts.configMapClient.Update(configMap)
	require.NoError(t, err)

	match, err := regexp.MatchString(dashbaordPattern, dashboardURL)
	require.NoError(t, err)
	require.True(t, match)

	err = ts.brokerClient.SuspendRuntime()
	assert.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded("", ts.DeprovisionTimeout)
	require.NoError(t, err)

	err = ts.brokerClient.UnsuspendRuntime()
	assert.NoError(t, err)

	err = ts.brokerClient.AwaitOperationSucceeded("", ts.ProvisionTimeout)
	require.NoError(t, err)

	ts.log.Info("Fetching runtime's kubeconfig")
	config, err := ts.runtimeClient.FetchRuntimeConfig()
	require.NoError(t, err)

	ts.log.Info(fmt.Sprintf("Creating a secret %s with test data", ts.ConfigName))
	err = ts.secretClient.Create(ts.testSecret(config))
	require.NoError(t, err)

	match, err = regexp.MatchString(dashbaordPattern, dashboardURL)
	require.NoError(t, err)
	require.True(t, match)
}
