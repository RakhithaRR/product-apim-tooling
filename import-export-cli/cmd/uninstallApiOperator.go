/*
*  Copyright (c) WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
*
*  WSO2 Inc. licenses this file to you under the Apache License,
*  Version 2.0 (the "License"); you may not use this file except
*  in compliance with the License.
*  You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing,
* software distributed under the License is distributed on an
* "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
* KIND, either express or implied.  See the License for the
* specific language governing permissions and limitations
* under the License.
 */

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	k8sUtils "github.com/wso2/product-apim-tooling/import-export-cli/operator/utils"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
	"strings"
)

const uninstallApiOperatorCmdLiteral = "api-operator"
const uninstallApiOperatorCmdShortDesc = "Uninstall API Operator"
const uninstallApiOperatorCmdLongDesc = "Uninstall API Operator in the configured K8s cluster"
const uninstallApiOperatorCmdExamples = utils.ProjectName + ` ` + uninstallCmdLiteral + ` ` + uninstallApiOperatorCmdLiteral + `
` + utils.ProjectName + ` ` + uninstallCmdLiteral + ` ` + uninstallApiOperatorCmdLiteral + ` --force`

var flagForceUninstallApiOperator bool

// uninstallApiOperatorCmd represents the uninstall api-operator command
var uninstallApiOperatorCmd = &cobra.Command{
	Use:     uninstallApiOperatorCmdLiteral,
	Short:   uninstallApiOperatorCmdShortDesc,
	Long:    uninstallApiOperatorCmdLongDesc,
	Example: uninstallApiOperatorCmdExamples,
	Run: func(cmd *cobra.Command, args []string) {
		isConfirm := flagForceUninstallApiOperator

		var operatorNs string
		artifactsNs := k8sUtils.ApiOpWso2Namespace
		if flagOperatorArtifactsNamespace != "" {
			artifactsNs = flagOperatorArtifactsNamespace
		}
		if flagApiOperatorNamespace != "" {
			operatorNs = flagApiOperatorNamespace
		} else {
			operatorNs = artifactsNs
		}

		if !flagForceUninstallApiOperator {
			isConfirmStr, err := utils.ReadInputString(
				fmt.Sprintf("\nUninstall \"%s\" and all related resources: APIs, Securities, Rate Limitings and Target Endpoints\n"+
					"[WARNING] Remove the artifacts namespace: %s\n"+
					"[WARNING] Remove the operator namespace: %s\n"+
					"Are you sure",
					k8sUtils.ApiOperator, artifactsNs, operatorNs),
				utils.Default{Value: "N", IsDefault: true},
				"",
				false,
			)
			if err != nil {
				utils.HandleErrorAndExit("Error reading user input Confirmation", err)
			}

			isConfirmStr = strings.ToUpper(isConfirmStr)
			isConfirm = isConfirmStr == "Y" || isConfirmStr == "YES"
		}

		if isConfirm {
			fmt.Println("Deleting kubernetes resources for API Operator")

			// delete the namespace "wso2-system"
			// namespace, "wso2-system" contains all the artifacts and configs
			// deleting the namespace: "wso2-system", will remove all the artifacts and configs
			deleteErrors := []error{
				k8sUtils.ExecuteCommand(k8sUtils.Kubectl, k8sUtils.K8sDelete, "crd", k8sUtils.ApiOpCrdApi),
				k8sUtils.ExecuteCommand(k8sUtils.Kubectl, k8sUtils.K8sDelete, "crd", k8sUtils.ApiOpCrdSecurity),
				k8sUtils.ExecuteCommand(k8sUtils.Kubectl, k8sUtils.K8sDelete, "crd", k8sUtils.ApiOpCrdRateLimiting),
				k8sUtils.ExecuteCommand(k8sUtils.Kubectl, k8sUtils.K8sDelete, "crd", k8sUtils.ApiOpCrdTargetEndpoint),
			}

			fmt.Printf("Removing namespace: %s\nThis operation will take some minutes...\n", artifactsNs)
			deleteErrors = append(deleteErrors, k8sUtils.ExecuteCommand(k8sUtils.Kubectl, k8sUtils.K8sDelete, "namespace", artifactsNs))
			if !strings.EqualFold(artifactsNs, operatorNs) {
				fmt.Printf("Removing namespace: %s\nThis operation will take some minutes...\n", operatorNs)
				deleteErrors = append(deleteErrors, k8sUtils.ExecuteCommand(k8sUtils.Kubectl, k8sUtils.K8sDelete, "namespace", operatorNs))
			}

			for _, err := range deleteErrors {
				if err != nil {
					utils.HandleErrorAndExit("Error uninstalling API Operator", err)
				}
			}
		} else {
			fmt.Println("Cancelled")
		}
	},
}

func init() {
	uninstallCmd.AddCommand(uninstallApiOperatorCmd)
	uninstallApiOperatorCmd.Flags().BoolVar(&flagForceUninstallApiOperator, "force", false, "Force uninstall API Operator")
	uninstallApiOperatorCmd.Flags().StringVarP(&flagOperatorArtifactsNamespace, "namespace", "n", "", "Remove operator artifacts in a specific namespace")
	uninstallApiOperatorCmd.Flags().StringVarP(&flagApiOperatorNamespace, "operator-namespace", "o", "", "Uninstall operator in a specific namespace")
}
