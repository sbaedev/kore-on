package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"kore-on/pkg/logger"
	"kore-on/pkg/model"
	"kore-on/pkg/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"kore-on/cmd/koreonctl/conf"
	"kore-on/cmd/koreonctl/conf/templates"

	"github.com/spf13/cobra"
)

type strDestroyCmd struct {
	verbose    bool
	dryRun     bool
	privateKey string
	user       string
	command    string
}

func destroyCmd() *cobra.Command {
	destroy := &strDestroyCmd{}
	cmd := &cobra.Command{
		Use:   "destroy [flags]",
		Short: "Delete kubernetes cluster, registry, prepare-airgap",
		Long: "This command can delete [Kubernetes cluster / registry / storage].\n" +
			"* If you do not use the sub command, it is all deleted.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return destroy.run()
		},
	}

	destroy.command = "reset-all"

	cmd.AddCommand(
		destroyPrepareAirGapCmd(),
		destroyClusterCmd(),
		destroyRegistryCmd(),
		destroyStorageCmd(),
	)

	f := cmd.Flags()
	f.BoolVar(&destroy.verbose, "verbose", false, "verbose")
	f.BoolVarP(&destroy.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&destroy.privateKey, "private-key", "p", "", "Specify ssh key path")
	f.StringVarP(&destroy.user, "user", "u", "", "login user")

	return cmd
}

func destroyPrepareAirGapCmd() *cobra.Command {
	destroyPrepareAirGapCmd := &strDestroyCmd{}

	cmd := &cobra.Command{
		Use:          "prepare-airgap [flags]",
		Short:        "Destroy prepare-airgap",
		Long:         "This command deletes the registry of the prepare-airgap host and deletes related directories.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return destroyPrepareAirGapCmd.run()
		},
	}

	destroyPrepareAirGapCmd.command = "reset-prepare-airgap"

	f := cmd.Flags()
	f.BoolVarP(&destroyPrepareAirGapCmd.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&destroyPrepareAirGapCmd.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&destroyPrepareAirGapCmd.privateKey, "private-key", "p", "", "Specify ssh key path")
	f.StringVarP(&destroyPrepareAirGapCmd.user, "user", "u", "", "login user")

	return cmd
}

func destroyClusterCmd() *cobra.Command {
	destroyClusterCmd := &strDestroyCmd{}

	cmd := &cobra.Command{
		Use:          "cluster [flags]",
		Short:        "Destroy cluster",
		Long:         "This command only deletes the Kubernetes cluster.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return destroyClusterCmd.run()
		},
	}

	destroyClusterCmd.command = "reset-cluster"

	f := cmd.Flags()
	f.BoolVarP(&destroyClusterCmd.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&destroyClusterCmd.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&destroyClusterCmd.privateKey, "private-key", "p", "", "Specify ssh key path")
	f.StringVarP(&destroyClusterCmd.user, "user", "u", "", "login user")

	return cmd
}

func destroyRegistryCmd() *cobra.Command {
	destroyRegistryCmd := &strDestroyCmd{}

	cmd := &cobra.Command{
		Use:          "registry [flags]",
		Short:        "Destroy registry",
		Long:         "This command deletes the installed registry(harbor) and deletes related services and directories.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return destroyRegistryCmd.run()
		},
	}

	destroyRegistryCmd.command = "reset-registry"

	f := cmd.Flags()
	f.BoolVarP(&destroyRegistryCmd.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&destroyRegistryCmd.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&destroyRegistryCmd.privateKey, "private-key", "p", "", "Specify ssh key path")
	f.StringVarP(&destroyRegistryCmd.user, "user", "u", "", "login user")

	return cmd
}

func destroyStorageCmd() *cobra.Command {
	destroyStorageCmd := &strDestroyCmd{}

	cmd := &cobra.Command{
		Use:          "storage [flags]",
		Short:        "Destroy storage",
		Long:         "This command deletes the installed storage(NFS) and deletes related services and directories.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return destroyStorageCmd.run()
		},
	}

	destroyStorageCmd.command = "reset-storage"

	f := cmd.Flags()
	f.BoolVarP(&destroyStorageCmd.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&destroyStorageCmd.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&destroyStorageCmd.privateKey, "private-key", "p", "", "Specify ssh key path")
	f.StringVarP(&destroyStorageCmd.user, "user", "u", "", "login user")

	return cmd
}

func (c *strDestroyCmd) run() error {

	workDir, _ := os.Getwd()
	var err error = nil
	logger.Infof("Start provisioning for cloud infrastructure")

	if err = c.destroy(workDir); err != nil {
		return err
	}
	return nil
}

func (c *strDestroyCmd) destroy(workDir string) error {
	// Doker check
	utils.CheckDocker()

	koreonImageName := conf.KoreOnImageName
	koreOnImage := conf.KoreOnImage
	koreOnConfigFileName := conf.KoreOnConfigFile
	koreOnConfigFilePath := conf.KoreOnConfigFileSubDir

	koreonToml, err := utils.GetKoreonTomlConfig(workDir + "/" + koreOnConfigFileName)
	if err != nil {
		logger.Fatal(err)
		os.Exit(1)
	}

	// Make provision data
	data := model.KoreonctlText{}
	data.KoreOnTemp = koreonToml
	data.Command = c.command

	// Processing template
	var textVar string
	switch data.Command {
	case "reset-all":
		textVar = templates.DestroyAllText
	case "reset-cluster":
		textVar = templates.DestroyClusterText
	case "reset-registry":
		textVar = templates.DestroyRegistryText
	case "reset-storage":
		textVar = templates.DestroyStorageText
	case "reset-prepare-airgap":
		textVar = templates.DestroyPrepareAirgapText
		koreonToml.KoreOn.ClosedNetwork = false
	}
	koreonctlText := template.New("DestroyText")
	temp, err := koreonctlText.Parse(textVar)
	if err != nil {
		logger.Errorf("Template has errors. cause(%s)", err.Error())
		return err
	}

	// TODO: 진행상황을 어떻게 클라이언트에 보여줄 것인가?
	var buff bytes.Buffer
	err = temp.Execute(&buff, data)
	if err != nil {
		logger.Errorf("Template execution failed. cause(%s)", err.Error())
		return err
	}

	if !utils.CheckUserInput(buff.String(), "y") {
		fmt.Println("nothing to changed. exit")
		os.Exit(1)
	}

	commandArgs := []string{
		"docker",
		"run",
		"--rm",
		"--privileged",
		"-it",
	}

	if !koreonToml.KoreOn.ClosedNetwork {
		commandArgs = append(commandArgs, "--pull")
		commandArgs = append(commandArgs, "always")
	}

	commandArgsVol := []string{
		"-v",
		fmt.Sprintf("%s:%s", workDir, "/"+koreOnConfigFilePath),
	}

	commandArgsKoreonctl := []string{
		koreOnImage,
		"./" + koreonImageName,
		"destroy",
	}

	if c.privateKey != "" {
		key := filepath.Base(c.privateKey)
		keyPath, _ := filepath.Abs(c.privateKey)
		commandArgsVol = append(commandArgsVol, "--mount")
		commandArgsVol = append(commandArgsVol, fmt.Sprintf("type=bind,source=%s,target=/home/%s,readonly", keyPath, key))
	}

	if c.command == "reset-prepare-airgap" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--tags")
		commandArgsKoreonctl = append(commandArgsKoreonctl, c.command)
	}

	if c.command == "reset-cluster" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--tags")
		commandArgsKoreonctl = append(commandArgsKoreonctl, c.command)
	}

	if c.command == "reset-registry" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--tags")
		commandArgsKoreonctl = append(commandArgsKoreonctl, c.command)
	}

	if c.command == "reset-storage" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--tags")
		commandArgsKoreonctl = append(commandArgsKoreonctl, c.command)
	}

	if c.verbose {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--verbose")
	}

	if c.dryRun {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--dry-run")
	}

	if c.privateKey != "" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--private-key")
		key := filepath.Base(c.privateKey)
		commandArgsKoreonctl = append(commandArgsKoreonctl, "/home/"+key)
	} else {
		logger.Fatal(fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an privateKey must be specified"))
	}

	if c.user != "" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--user")
		commandArgsKoreonctl = append(commandArgsKoreonctl, c.user)
	} else {
		logger.Fatal(fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an ssh login user must be specified"))
	}

	commandArgs = append(commandArgs, commandArgsVol...)
	commandArgs = append(commandArgs, commandArgsKoreonctl...)

	binary, lookErr := exec.LookPath("docker")
	if lookErr != nil {
		logger.Fatal(lookErr)
	}

	err = syscall.Exec(binary, commandArgs, os.Environ())
	if err != nil {
		log.Printf("Command finished with error: %v", err)
	}

	return nil
}
