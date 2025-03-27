package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/spf13/cobra"
)

var UsageError = errors.New("UsageError")
var InvalidRuleError = errors.New("InvalidRuleError")

type ParseCommand struct {
	cobraCmd     *cobra.Command
	rule         string
	parser       rules.Parser
	ruleFilePath string
	sort         bool
	unique       bool
	match        string
	signature    bool
	noColor      bool
}

func NewParseCmd() *cobra.Command {
	cmd := ParseCommand{}
	cobraCmd := &cobra.Command{
		Use:     "parse",
		Aliases: []string{"p"},
		Short:   "Parses a HAP rule entry validating its format.",
		Long:    "Parses a HAP rule entry validating its format.",
		Example: `
	# Parse multiple rules from command line arguments
	hap parse -e 'azure(PR=westeurope); aws->EU' 

	# Parse multiple rules from a file:
	# --- rules.yaml
	# rule:
	# - azure(PR=westeurope)
	# - aws->EU 
	# ---
	hap parse -f rules.yaml

	# Check which rule will be matched and triggered against the provided provisioning data
	hap parse  -f ./correct-rules.yaml -m '{"plan": "aws", "platformRegion": "cf-eu11", "hyperscalerRegion": "westeurope"}'
		`,
		RunE: func(_ *cobra.Command, args []string) error {
			return cmd.Run()
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.rule, "entry", "e", "", "A rule to validate where each rule entry is separated by comma.")
	cobraCmd.Flags().StringVarP(&cmd.match, "match", "m", "", "Check what rule will be matched and triggered against the provided test data. Only valid entries are taking into account when matching. Data is passed in json format, example: '{\"plan\": \"aws\", \"platformRegion\": \"cf-eu11\"}'.")
	cobraCmd.Flags().StringVarP(&cmd.ruleFilePath, "file", "f", "", "Read rules from a file pointed to by parameter value. The file must contain a valid yaml list, where each rule entry starts with '-' and is placed in its own line.")
	cobraCmd.MarkFlagsOneRequired("entry", "file")

	return cobraCmd
}

func (cmd *ParseCommand) Run() error {

	// create enabled plans
	enabledPlans := broker.EnablePlans{}
	for _, plan := range broker.PlanNamesMapping {
		enabledPlans = append(enabledPlans, plan)
	}

	var rulesService *rules.RulesService
	var err error
	if cmd.ruleFilePath != "" {
		cmd.cobraCmd.Printf("Parsing rules from file: %s\n", cmd.ruleFilePath)
		rulesService, err = rules.NewRulesServiceFromFile(cmd.ruleFilePath, sets.New(maps.Keys(broker.PlanIDsMapping)...), sets.New[string]())
	} else {
		rulesService, err = rules.NewRulesServiceFromSlice(strings.Split(cmd.rule, ";"), sets.New(maps.Keys(broker.PlanIDsMapping)...), sets.New[string]())
	}

	if err != nil {
		cmd.cobraCmd.Printf("Error: %s\n", err)
		return UsageError
	}

	rulesetValid := rulesService.ValidRules == nil || len(rulesService.ValidRules.Rules) == 0
	if rulesetValid {
		cmd.cobraCmd.Printf("There are errors in your rule configuration.\n")
		return InvalidRuleError
	} else {
		var dataToMatch *rules.ProvisioningAttributes
		cmd.cobraCmd.Printf("Your rule configuration is OK.\n")
		if cmd.match != "" {
			dataToMatch, err = getDataForMatching(cmd.match)
			if err != nil {
				cmd.cobraCmd.Printf("Provided data to match is not valid: %s\n", err)
				return UsageError
			}
			matchingResults, matched := rulesService.MatchProvisioningAttributesWithValidRuleset(dataToMatch)
			if matched {
				cmd.cobraCmd.Printf("Matched rule: %s\n", matchingResults.Rule())
			} else {
				cmd.cobraCmd.Printf("No rule matched the provided data.\n")
				return InvalidRuleError
			}
		}
	}
	return nil
}

func getDataForMatching(content string) (*rules.ProvisioningAttributes, error) {
	data := &rules.ProvisioningAttributes{}
	err := json.Unmarshal([]byte(content), data)
	if err != nil {
		return nil, err
	}

	if data.Plan == "" {
		return nil, fmt.Errorf("Plan is a required field.")
	}
	if data.PlatformRegion == "" {
		return nil, fmt.Errorf("PlatformRegion is a required field.")
	}
	if data.HyperscalerRegion == "" {
		return nil, fmt.Errorf("HyperscalerRegion is a required field.")
	}
	if data.Hyperscaler == "" {
		return nil, fmt.Errorf("Hyperscaler is a required field.")
	}
	return data, nil
}
