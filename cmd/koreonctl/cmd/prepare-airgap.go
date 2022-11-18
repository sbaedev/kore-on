package cmd

import (
	"fmt"
	"kore-on/pkg/logger"
	"kore-on/pkg/utils"
	"log"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type strAirGapCmd struct {
	dryRun     bool
	verbose    bool
	privateKey string
	user       string
	command    string
}

func airGapCmd() *cobra.Command {
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

	cmd.AddCommand(downLoadArchiveCmd())

	f := cmd.Flags()
	f.BoolVar(&prepareAirgap.verbose, "vvv", false, "verbose")
	f.BoolVarP(&prepareAirgap.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&prepareAirgap.privateKey, "private-key", "p", "", "Specify ansible playbook privateKey")
	f.StringVarP(&prepareAirgap.user, "user", "u", "", "SSH login user")

	return cmd
}

func downLoadArchiveCmd() *cobra.Command {
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

	downLoadArchive.command = "download-archive"

	f := cmd.Flags()
	f.BoolVarP(&downLoadArchive.verbose, "verbose", "v", false, "verbose")
	f.BoolVarP(&downLoadArchive.dryRun, "dry-run", "d", false, "dryRun")
	f.StringVarP(&downLoadArchive.privateKey, "private-key", "p", "", "Specify ansible playbook privateKey")
	f.StringVarP(&downLoadArchive.user, "user", "u", "", "SSH login user")

	return cmd
}

func (c *strAirGapCmd) run() error {

	//if !utils.CheckUserInput("Do you really want to create? Only 'yes' will be accepted to confirm: ", "yes") {
	//	fmt.Println("nothing to changed. exit")
	//	os.Exit(1)
	//}

	workDir, _ := os.Getwd()
	var err error = nil
	logger.Infof("Start provisioning for preparing a kubernetes cluster and registry")

	if err = c.airgap(workDir); err != nil {
		return err
	}
	return nil
}

func (c *strAirGapCmd) airgap(workDir string) error {
	// Doker check
	utils.CheckDocker()

	koreonImageName := viper.GetString("KoreOn.KoreonImageName")
	koreOnImage := viper.GetString("KoreOn.KoreOnImage")
	koreOnConfigFileName := viper.GetString("KoreOn.KoreOnConfigFile")
	koreOnConfigFilePath := utils.IskoreOnConfigFilePath(koreOnConfigFileName)

	commandArgs := []string{
		"docker",
		"run",
		"--name",
		koreonImageName,
		"--rm",
		"--privileged",
		"-it",
	}

	commandArgsVol := []string{
		"-v",
		fmt.Sprintf("%s:%s", workDir, koreOnConfigFilePath),
	}

	commandArgsKoreonctl := []string{
		koreOnImage,
		"prepare-airgap",
	}

	if c.command == "download-archive" {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "download-archive")
	}

	commandArgs = append(commandArgs, commandArgsVol...)
	commandArgs = append(commandArgs, commandArgsKoreonctl...)

	if c.verbose {
		commandArgs = append(commandArgs, "--vvv")
	}

	if c.dryRun {
		commandArgs = append(commandArgs, "--dry-run")
	}

	if c.privateKey != "" {
		commandArgs = append(commandArgs, "--private-key")
	} else {
		logger.Fatal(fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an privateKey must be specified"))
	}

	if c.user != "" {
		commandArgs = append(commandArgs, "--user")
	} else {
		logger.Fatal(fmt.Errorf("[ERROR]: %s", "To run ansible-playbook an ssh login user must be specified"))
	}

	err := syscall.Exec("/usr/local/bin/docker", commandArgs, os.Environ())
	if err != nil {
		log.Printf("Command finished with error: %v", err)
	}

	return nil
}