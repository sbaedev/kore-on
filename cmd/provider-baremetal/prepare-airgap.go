package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"kore-on/pkg/logger"
	"kore-on/pkg/utils"
	"os"

	"github.com/apenella/go-ansible/pkg/execute"
	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/apenella/go-ansible/pkg/stdoutcallback/results"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Commands structure
type strAirGapCmd struct {
	dryRun        bool
	verbose       bool
	inventory     string
	tags          string
	playbookFiles []string
	privateKey    string
	user          string
	extravars     map[string]interface{}
}

func AirGapCmd() *cobra.Command {
	prepareAirgap := &strAirGapCmd{}

	cmd := &cobra.Command{
		Use:          "prepare-airgap [flags]",
		Short:        "Preparing a kubernetes cluster and registry for AirGap network",
		Long:         "",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return prepareAirgap.run()
		},
	}

	cmd.AddCommand(DownLoadArchiveCmd())

	// Default value for command struct
	prepareAirgap.tags = ""
	prepareAirgap.inventory = "./internal/playbooks/koreon-playbook/inventory/inventory.ini"
	prepareAirgap.playbookFiles = []string{
		"./internal/playbooks/koreon-playbook/prepare-airgap.yaml",
	}

	f := cmd.Flags()
	f.BoolVarP(&prepareAirgap.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&prepareAirgap.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVar(&prepareAirgap.tags, "tags", prepareAirgap.tags, "Ansible options tags")
	f.StringVarP(&prepareAirgap.privateKey, "private-key", "p", "", "Specify ansible playbook privateKey")
	f.StringVarP(&prepareAirgap.user, "user", "u", "", "SSH login user")

	return cmd
}

func DownLoadArchiveCmd() *cobra.Command {
	downLoadArchive := &strAirGapCmd{}

	cmd := &cobra.Command{
		Use:          "download-archive [flags]",
		Short:        "Download archive files to localhost",
		Long:         "",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return downLoadArchive.run()
		},
	}

	// Default value for command struct
	downLoadArchive.tags = ""
	downLoadArchive.inventory = "./internal/playbooks/koreon-playbook/inventory/inventory.ini"
	downLoadArchive.playbookFiles = []string{
		"./internal/playbooks/koreon-playbook/download-archive-to-local.yaml",
	}

	f := cmd.Flags()
	f.BoolVarP(&downLoadArchive.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&downLoadArchive.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVar(&downLoadArchive.tags, "tags", downLoadArchive.tags, "Ansible options tags")
	f.StringVarP(&downLoadArchive.privateKey, "private-key", "p", "", "Specify ansible playbook privateKey")
	f.StringVarP(&downLoadArchive.user, "user", "u", "", "SSH login user")

	return cmd
}

func (c *strAirGapCmd) run() error {
	koreOnConfigFileName := viper.GetString("KoreOn.KoreOnConfigFile")
	koreOnConfigFilePath := utils.IskoreOnConfigFilePath(koreOnConfigFileName)
	koreonToml, value := utils.ValidateKoreonTomlConfig(koreOnConfigFilePath, "prepare-airgap")

	if value {
		b, err := json.Marshal(koreonToml)
		if err != nil {
			logger.Fatal(err)
			os.Exit(1)
		}
		if err := json.Unmarshal(b, &c.extravars); err != nil {
			logger.Fatal(err.Error())
			os.Exit(1)
		}
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
		Inventory: c.inventory,
		Verbose:   c.verbose,
		Tags:      c.tags,
		ExtraVars: c.extravars,
	}

	playbook := &playbook.AnsiblePlaybookCmd{
		Playbooks:         c.playbookFiles,
		ConnectionOptions: ansiblePlaybookConnectionOptions,
		Options:           ansiblePlaybookOptions,
		Exec: execute.NewDefaultExecute(
			execute.WithTransformers(
				results.Prepend("Prepare AirGap"),
			),
		),
	}

	options.AnsibleForceColor()

	err := playbook.Run(context.TODO())
	if err != nil {
		return err
	}

	return nil
}