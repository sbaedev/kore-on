package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"kore-on/pkg/logger"
	"kore-on/pkg/utils"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"kore-on/cmd/koreonctl/conf"
	"kore-on/cmd/koreonctl/conf/templates"

	"github.com/spf13/cobra"
	"github.com/zcalusic/sysinfo"
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

	current, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	if current.Uid != "0" {
		log.Fatal("requires superuser privilege")
	}

	var si sysinfo.SysInfo

	si.GetSysInfo()

	data, err := json.MarshalIndent(&si, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))

	repoPath := "/etc/yum.repos.d"
	err = ioutil.WriteFile(repoPath+"/bastion-local.repo", buff.Bytes(), 0644)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Println("aa == ", runtime.GOOS)

	commandArgs := []string{
		"yum",
		workDir + "/local",
	}

	if !koreonToml.KoreOn.ClosedNetwork {
		commandArgs = append(commandArgs, workDir+"/local")
	}

	commandArgsVol := []string{
		"-v",
		fmt.Sprintf("%s:%s", workDir, "/"+koreOnConfigFilePath),
	}

	commandArgsKoreonctl := []string{
		koreOnImage,
		"./" + koreonImageName,
		"init",
	}

	if c.verbose {
		commandArgsKoreonctl = append(commandArgsKoreonctl, "--vvv")
	}

	commandArgs = append(commandArgs, commandArgsVol...)
	commandArgs = append(commandArgs, commandArgsKoreonctl...)

	// binary, lookErr := exec.LookPath("yum")
	// if lookErr != nil {
	// 	logger.Fatal(lookErr)
	// }

	// err = syscall.Exec(binary, commandArgs, os.Environ())
	// if err != nil {
	// 	log.Printf("Command finished with error: %v", err)
	// }

	return nil
}
