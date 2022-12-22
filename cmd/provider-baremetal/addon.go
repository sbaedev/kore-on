package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"kore-on/pkg/logger"
	"kore-on/pkg/utils"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/execute/measure"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Commands structure
type strAddonCmd struct {
	dryRun         bool
	verbose        bool
	installHelm    bool
	helmBinaryFile string
	inventory      string
	tags           string
	playbookFiles  []string
	privateKey     string
	user           string
	extravars      map[string]interface{}
	addonExtravars map[string]interface{}
	result         map[string]interface{}
}

func AddonCmd() *cobra.Command {
	addon := &strAddonCmd{}

	cmd := &cobra.Command{
		Use:   "addon [flags]",
		Short: "Deployment Applications in kubernetes cluster",
		Long: "This command deploys the application to Kubernetes.\n" +
			"Use helm as the package manager for Kubernetes.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addon.run()
		},
	}

	// Default value for command struct
	addon.tags = ""
	addon.inventory = "./internal/playbooks/koreon-playbook/inventory/inventory.ini"
	addon.playbookFiles = []string{
		"./internal/playbooks/koreon-playbook/add-on.yaml",
	}
	f := cmd.Flags()
	f.BoolVar(&addon.verbose, "verbose", false, "verbose")
	f.BoolVarP(&addon.dryRun, "dry-run", "d", false, "dryRun")
	f.BoolVar(&addon.installHelm, "install-helm", false, "Helm installation options")
	f.StringVar(&addon.helmBinaryFile, "helm-binary-file", "", "helm binary file")
	f.StringVar(&addon.tags, "tags", addon.tags, "Ansible options tags")
	f.StringVarP(&addon.privateKey, "private-key", "p", "", "Specify ssh key path")
	f.StringVarP(&addon.user, "user", "u", "", "login user")

	return cmd
}

func (c *strAddonCmd) run() error {
	addonConfigFileName := viper.GetString("Addon.AddonConfigFile")
	addonPath := utils.IskoreOnConfigFilePath(addonConfigFileName)
	addonToml, err := utils.GetAddonTomlConfig(addonPath)
	if err != nil {
		logger.Fatal(err)
	} else {
		// Install Helm
		if c.installHelm {
			addonToml.Addon.HelmInstall = c.installHelm
		}
		if c.helmBinaryFile != "" {
			addonToml.Addon.HelmBinaryFile = c.helmBinaryFile
		}

		// Prompt user for more input
		if addonToml.Apps.CsiDriverNfs.Install {
			id := utils.InputPrompt("# Enter the username for the private registry.\nusername:")
			addonToml.Apps.CsiDriverNfs.ChartRefID = base64.StdEncoding.EncodeToString([]byte(id))

			pw := utils.SensitivePrompt("# Enter the password for the private registry.\npassword:")
			addonToml.Apps.CsiDriverNfs.ChartRefPW = base64.StdEncoding.EncodeToString([]byte(pw))
		}

		addonToml.Addon.HelmVersion = utils.IsSupportVersion("", "SupportHelmVersion")
		if addonToml.Addon.AddonDataDir == "" {
			addonToml.Addon.AddonDataDir = "/data/addon"
		}

		b, err := json.Marshal(addonToml)
		if err != nil {
			logger.Fatal(err)
		}
		if err := json.Unmarshal(b, &c.addonExtravars); err != nil {
			logger.Fatal(err.Error())
		}

		result := make(map[string]interface{})
		// for k, v := range c.extravars {
		// 	if _, ok := c.extravars[k]; ok {
		// 		result[k] = v
		// 	}
		// }
		for k, v := range c.addonExtravars {
			if _, ok := c.addonExtravars[k]; ok {
				result[k] = v
			}
		}
		c.result = result
	}

	if len(c.playbookFiles) < 1 {
		return fmt.Errorf("[ERROR]: %s", "To run ansible-playbook playbook file path must be specified")
	}

	if len(c.inventory) < 1 {
		return fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an inventory must be specified")
	}

	if len(c.privateKey) < 1 {
		return fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an privateKey must be specified")
	}

	if len(c.user) < 1 {
		return fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an ssh login user must be specified")
	}

	ansiblePlaybookConnectionOptions := &options.AnsibleConnectionOptions{
		PrivateKey: c.privateKey,
		User:       c.user,
	}

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		Inventory:     c.inventory,
		Verbose:       c.verbose,
		Tags:          c.tags,
		ExtraVars:     c.result,
		ExtraVarsFile: []string{"@internal/playbooks/koreon-playbook/download/test-values.yaml"},
	}

	executorTimeMeasurement := measure.NewExecutorTimeMeasurement(
		execute.NewDefaultExecute(
			execute.WithEnvVar("ANSIBLE_FORCE_COLOR", "true"),
			execute.WithTransformers(
				utils.OutputColored(),
				results.Prepend("Addon deployment in cluster"),
				// results.LogFormat(results.DefaultLogFormatLayout, results.Now),
			),
		),
		// measure.WithShowDuration(),
	)

	playbook := &playbook.AnsiblePlaybookCmd{
		Playbooks:         c.playbookFiles,
		ConnectionOptions: ansiblePlaybookConnectionOptions,
		Options:           ansiblePlaybookOptions,
		Exec:              executorTimeMeasurement,
	}

	options.AnsibleForceColor()

	err = playbook.Run(context.TODO())
	if err != nil {
		return err
	}

	return nil
}
