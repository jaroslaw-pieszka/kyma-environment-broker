package provider

import (
	"fmt"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
)

type Provider interface {
	Provide() internal.ProviderValues
}

func GetPlanSpecificValues(
	operation *internal.Operation,
	multiZoneCluster bool,
	defaultTrialProvider pkg.CloudProvider,
	useSmallerMachineTypes bool,
	trialPlatformRegionMapping map[string]string,
	defaultPurpose string,
	commercialFailureTolerance string,
) (internal.ProviderValues, error) {
	var p Provider
	switch operation.ProvisioningParameters.PlanID {
	case broker.AWSPlanID, broker.BuildRuntimeAWSPlanID:
		p = &AWSInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
			FailureTolerance:       commercialFailureTolerance,
		}
	case broker.PreviewPlanID:
		p = &AWSInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
			FailureTolerance:       commercialFailureTolerance,
		}
	case broker.AzurePlanID, broker.BuildRuntimeAzurePlanID:
		p = &AzureInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
			FailureTolerance:       commercialFailureTolerance,
		}
	case broker.AzureLitePlanID:
		p = &AzureLiteInputProvider{
			Purpose:                defaultPurpose,
			UseSmallerMachineTypes: useSmallerMachineTypes,
			ProvisioningParameters: operation.ProvisioningParameters,
		}
	case broker.GCPPlanID, broker.BuildRuntimeGCPPlanID:
		p = &GCPInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
			FailureTolerance:       commercialFailureTolerance,
		}
	case broker.FreemiumPlanID:
		switch operation.ProvisioningParameters.PlatformProvider {
		case pkg.AWS:
			p = &AWSFreemiumInputProvider{
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case pkg.Azure:
			p = &AzureFreemiumInputProvider{
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		default:
			return internal.ProviderValues{}, fmt.Errorf("freemium provider for '%s' is not supported", operation.ProvisioningParameters.PlatformProvider)
		}
	case broker.SapConvergedCloudPlanID:
		p = &SapConvergedCloudInputProvider{
			Purpose:                defaultPurpose,
			MultiZone:              multiZoneCluster,
			ProvisioningParameters: operation.ProvisioningParameters,
			FailureTolerance:       commercialFailureTolerance,
		}
	case broker.TrialPlanID:
		var trialProvider pkg.CloudProvider
		if operation.ProvisioningParameters.Parameters.Provider == nil {
			trialProvider = defaultTrialProvider
		} else {
			trialProvider = *operation.ProvisioningParameters.Parameters.Provider
		}
		switch trialProvider {
		case pkg.AWS:
			p = &AWSTrialInputProvider{
				PlatformRegionMapping:  trialPlatformRegionMapping,
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case pkg.GCP:
			p = &GCPTrialInputProvider{
				PlatformRegionMapping:  trialPlatformRegionMapping,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		case pkg.Azure:
			p = &AzureTrialInputProvider{
				PlatformRegionMapping:  trialPlatformRegionMapping,
				UseSmallerMachineTypes: useSmallerMachineTypes,
				ProvisioningParameters: operation.ProvisioningParameters,
			}
		default:
			return internal.ProviderValues{}, fmt.Errorf("trial provider for %s not yet implemented", trialProvider)
		}

	case broker.OwnClusterPlanID:
		p = &OwnClusterinputProvider{}
	default:
		return internal.ProviderValues{}, fmt.Errorf("plan %s not supported", operation.ProvisioningParameters.PlanID)
	}
	return p.Provide(), nil
}

func ProviderToCloudProvider(providerType string) pkg.CloudProvider {
	switch providerType {
	case "azure":
		return pkg.Azure
	case "aws":
		return pkg.AWS
	case "gcp":
		return pkg.GCP
	case "openstack":
		return pkg.SapConvergedCloud
	default:
		return pkg.UnknownProvider
	}
}
