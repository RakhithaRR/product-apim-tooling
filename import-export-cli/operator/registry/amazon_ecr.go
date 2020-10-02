package registry

import (
	"fmt"
	k8sUtils "github.com/wso2/product-apim-tooling/import-export-cli/operator/utils"
	"github.com/wso2/product-apim-tooling/import-export-cli/utils"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

// validation regex for repository URI validation
const amazonRepoRegex = `\.amazonaws\.com\/.*$`

// AmazonEcrRegistry represents Amazon ECR registry
var AmazonEcrRegistry = &Registry{
	Name:       "AMAZON_ECR",
	Caption:    "Amazon ECR",
	Repository: Repository{},
	Option:     2,
	Read: func(reg *Registry, flagValues *map[string]FlagValue) {
		var repository, credFile string

		// check input mode: interactive or batch
		if flagValues == nil {
			// get inputs in interactive mode
			repository, credFile = readAmazonEcrInputs()
		} else {
			// get inputs in batch mode
			repository = (*flagValues)[k8sUtils.FlagBmRepository].Value.(string)
			credFile = (*flagValues)[k8sUtils.FlagBmKeyFile].Value.(string)

			// validate required inputs
			if !utils.ValidateValue(repository, amazonRepoRegex) {
				utils.HandleErrorAndExit("Invalid repository uri: "+repository, nil)
			}
			if !utils.IsFileExist(credFile) {
				utils.HandleErrorAndExit("Invalid credential file: "+credFile, nil)
			}
		}

		reg.Repository.Name = repository
		reg.Repository.KeyFile = credFile
	},
	Run: func(reg *Registry) {
		createAmazonEcrConfig()
		k8sUtils.K8sCreateSecretFromFile(
			k8sUtils.AwsCredentialsSecret, k8sUtils.ApiOpWso2Namespace,
			reg.Repository.KeyFile, k8sUtils.AwsCredentialsFile,
		)
	},
	Flags: Flags{
		RequiredFlags: &map[string]bool{k8sUtils.FlagBmRepository: true, k8sUtils.FlagBmKeyFile: true},
		OptionalFlags: &map[string]bool{},
	},
}

// readAmazonEcrInputs reads file path for amazon credential file
func readAmazonEcrInputs() (string, string) {
	isConfirm := false
	repository := ""
	credFile := ""
	var err error

	for !isConfirm {
		repository, err = utils.ReadInputString(
			"Enter Repository URI (<aws_account_id.dkr.ecr.region.amazonaws.com>/repository)",
			utils.Default{IsDefault: false}, amazonRepoRegex, true,
		)
		if err != nil {
			utils.HandleErrorAndExit("Error reading DockerHub repository name from user", err)
		}

		defaultLocation, err := os.UserHomeDir()
		if err == nil {
			defaultLocation = filepath.Join(defaultLocation, ".aws", "credentials")
		} // else ignore and make defaultLocation = ""

		credFile, err = utils.ReadInput("Amazon credential file", utils.Default{Value: defaultLocation, IsDefault: true},
			utils.IsFileExist, "Invalid file", true)
		if err != nil {
			utils.HandleErrorAndExit("Error reading amazon credential file from user", err)
		}

		fmt.Println("\nRepository     : " + repository)
		fmt.Println("Credential File: " + credFile)

		isConfirmStr, err := utils.ReadInputString("Confirm configurations",
			utils.Default{Value: "Y", IsDefault: true}, "", false)
		if err != nil {
			utils.HandleErrorAndExit("Error reading user input Confirmation", err)
		}

		isConfirm = strings.EqualFold(isConfirmStr, "y") || strings.EqualFold(isConfirmStr, "yes")
	}

	return repository, credFile
}

// createAmazonEcrConfig creates K8S secret with credentials for Amazon ECR
func createAmazonEcrConfig() {
	configJson := `{ "credsStore": "ecr-login" }`

	tempFile, err := utils.CreateTempFile("config-*.json", []byte(configJson))
	if err != nil {
		utils.HandleErrorAndExit("Error writing configs to temporary file", err)
	}
	defer os.Remove(tempFile)

	// render configmap
	ecrConfimapMap := k8sUtils.RenderSecretTemplate(k8sUtils.AmazonCredHelperConfMap, k8sUtils.ApiOpWso2Namespace, k8sUtils.K8sConfigMap)
	ecrConfimapMap["data"] = make(map[interface{}]interface{})
	ecrConfimapMap["data"].(map[interface{}]interface{})["config.json"] = configJson

	configMap, err := yaml.Marshal(ecrConfimapMap)
	if err != nil {
		utils.HandleErrorAndExit("Error rendering ECR configmap", err)
	}

	// apply config map
	if err = k8sUtils.K8sApplyFromStdin(string(configMap)); err != nil {
		utils.HandleErrorAndExit("Error creating docker config for Amazon ECR", err)
	}
}

func init() {
	add(AmazonEcrRegistry)
}
