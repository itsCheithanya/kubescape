package policyhandler

import (
	"fmt"
	"strings"

	"github.com/armosec/kubescape/v2/core/cautils"
	"github.com/armosec/kubescape/v2/core/cautils/getter"
	"github.com/armosec/kubescape/v2/core/cautils/logger"
	"github.com/armosec/kubescape/v2/core/cautils/logger/helpers"
	"github.com/armosec/opa-utils/reporthandling"
)

func (policyHandler *PolicyHandler) getPolicies(notification *reporthandling.PolicyNotification, policiesAndResources *cautils.OPASessionObj) error {
	logger.L().Info("Downloading/Loading policy definitions")

	cautils.StartSpinner()
	defer cautils.StopSpinner()

	policies, err := policyHandler.getScanPolicies(notification)
	if err != nil {
		return err
	}
	if len(policies) == 0 {
		return fmt.Errorf("failed to download policies: '%s'. Make sure the policy exist and you spelled it correctly. For more information, please feel free to contact ARMO team", strings.Join(policyIdentifierToSlice(notification.Rules), ", "))
	}

	policiesAndResources.Policies = policies

	// get exceptions
	exceptionPolicies, err := policyHandler.getters.ExceptionsGetter.GetExceptions(cautils.ClusterName)
	if err == nil {
		policiesAndResources.Exceptions = exceptionPolicies
	}

	// get account configuration
	controlsInputs, err := policyHandler.getters.ControlsInputsGetter.GetControlsInputs(cautils.ClusterName)
	if err == nil {
		policiesAndResources.RegoInputData.PostureControlInputs = controlsInputs
	}
	cautils.StopSpinner()

	logger.L().Success("Downloaded/Loaded policy")
	return nil
}

func (policyHandler *PolicyHandler) getScanPolicies(notification *reporthandling.PolicyNotification) ([]reporthandling.Framework, error) {
	frameworks := []reporthandling.Framework{}

	switch getScanKind(notification) {
	case reporthandling.KindFramework: // Download frameworks
		for _, rule := range notification.Rules {
			receivedFramework, err := policyHandler.getters.PolicyGetter.GetFramework(rule.Name)
			if err != nil {
				return frameworks, policyDownloadError(err)
			}
			if receivedFramework != nil {
				frameworks = append(frameworks, *receivedFramework)

				cache := getter.GetDefaultPath(rule.Name + ".json")
				if err := getter.SaveInFile(receivedFramework, cache); err != nil {
					logger.L().Warning("failed to cache file", helpers.String("file", cache), helpers.Error(err))
				}
			}
		}
	case reporthandling.KindControl: // Download controls
		f := reporthandling.Framework{}
		var receivedControl *reporthandling.Control
		var err error
		for _, rule := range notification.Rules {
			receivedControl, err = policyHandler.getters.PolicyGetter.GetControl(rule.Name)
			if err != nil {
				return frameworks, policyDownloadError(err)
			}
			if receivedControl != nil {
				f.Controls = append(f.Controls, *receivedControl)

				cache := getter.GetDefaultPath(rule.Name + ".json")
				if err := getter.SaveInFile(receivedControl, cache); err != nil {
					logger.L().Warning("failed to cache file", helpers.String("file", cache), helpers.Error(err))
				}
			}
		}
		frameworks = append(frameworks, f)
		// TODO: add case for control from file
	default:
		return frameworks, fmt.Errorf("unknown policy kind")
	}
	return frameworks, nil
}

func policyIdentifierToSlice(rules []reporthandling.PolicyIdentifier) []string {
	s := []string{}
	for i := range rules {
		s = append(s, fmt.Sprintf("%s: %s", rules[i].Kind, rules[i].Name))
	}
	return s
}
