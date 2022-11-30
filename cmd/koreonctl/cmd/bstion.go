package cmd

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"kore-on/pkg/logger"
	"kore-on/pkg/utils"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"kore-on/cmd/koreonctl/conf"
	"kore-on/cmd/koreonctl/conf/templates"

	"github.com/spf13/cobra"
)

type strBstionCmd struct {
	verbose         bool
	archiveFilePath string
	command         string
}

func bastionCmd() *cobra.Command {
	bastionCmd := &strBstionCmd{}
	cmd := &cobra.Command{
		Use:          "bastion [flags]",
		Short:        "Install docker in bastion host",
		Long:         "This command a installation docker on bastion host.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bastionCmd.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&bastionCmd.verbose, "vvv", false, "verbose")
	f.StringVar(&bastionCmd.archiveFilePath, "archive-file-path", "", "archive file path")

	return cmd
}

func (c *strBstionCmd) run() error {
	workDir, _ := os.Getwd()
	var err error = nil
	logger.Infof("Start provisioning for cloud infrastructure")

	if err = c.bastion(workDir); err != nil {
		return err
	}
	return nil
}

func (c *strBstionCmd) bastion(workDir string) error {
	if runtime.GOOS != "linux" {
		logger.Fatal("This command option is only supported on the Linux platform.")
	}

	koreOnConfigFilePath, err := filepath.Abs(conf.KoreOnConfigFile)
	if err != nil {
		logger.Fatal(err)
	}

	koreonToml, err := utils.GetKoreonTomlConfig(koreOnConfigFilePath)
	if err != nil {
		logger.Fatal(err)
	}

	if !koreonToml.KoreOn.ClosedNetwork {
		logger.Fatal("This command is only supported on the clese network")
		os.Exit(1)
	}

	// Doker check
	_, dockerCheck := exec.LookPath("docker")
	if dockerCheck == nil {
		logger.Fatal("Docker already exists.")
	}

	// mkdir local directory
	path := "local"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			logger.Fatal(err)
		}
	}

	// Processing template
	bastionText := template.New("bastionLocalRepoText")
	temp, err := bastionText.Parse(templates.BastionLocalRepoText)
	if err != nil {
		logger.Errorf("Template has errors. cause(%s)", err.Error())
		return err
	}

	// TODO: 진행상황을 어떻게 클라이언트에 보여줄 것인가?
	var buff bytes.Buffer
	localPath, _ := filepath.Abs(path)
	err = temp.Execute(&buff, localPath)
	if err != nil {
		logger.Errorf("Template execution failed. cause(%s)", err.Error())
		return err
	}

	repoPath := "/etc/yum.repos.d"
	err = ioutil.WriteFile(repoPath+"/bastion-local.repo", buff.Bytes(), 0644)
	if err != nil {
		logger.Fatal(err)
	}

	commandArgs := []string{
		"yum",
		"install",
		"-y",
		"--disablerepo=*",
		"--enablerepo=bastion-local-to-file",
		"docker-ce docker-cli",
	}

	err = utils.SyscallExec(commandArgs[0], commandArgs)
	if err != nil {
		logger.Fatal("Command finished with error: %v", err)
	}

	dockerLoad()

	return nil
}

func dockerLoad() error {
	commandArgs := []string{
		"docker",
		"load",
		"--input",
		conf.KoreOnImageArchive,
	}
	err := utils.SyscallExec(commandArgs[0], commandArgs)
	if err != nil {
		logger.Fatal("Command finished with error: %v", err)
	}
	return nil
}
