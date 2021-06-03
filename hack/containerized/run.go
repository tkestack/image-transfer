package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"tkestack.io/image-transfer/pkg/apis/ccrapis"
	"tkestack.io/image-transfer/pkg/log"
)

const (
	secretFilePath   = "/tmp/secret.yaml"
	securityFilePath = "/tmp/security.yaml"
	binaryFilePath   = "/image-transfer"

	securityFileTemplate = `{{.TCRDomian}}:
  username: {{.TCRUsername}}
  password: "{{.TCRPassword}}"
{{.CCRDomain}}:
  username: {{.CCRUsername}}
  password: "{{.CCRPassword}}"`
	secretFileTemplate = `ccr:
  secretId: {{.CCRSecretId}}
  secretKey: {{.CCRSecretKey}}
tcr:
  secretId: {{.TCRSecretId}}
  secretKey: {{.TCRSecretKey}}`
)

var (
	// for same region and account
	secretId   string
	secretKey  string
	regionName string
	// for different region and account
	ccrSecretId   string
	ccrSecretKey  string
	tcrSecretId   string
	tcrSecretKey  string
	ccrRegionName string
	tcrRegionName string

	// common flag
	tcrName  string
	ccrAuth  string
	tcrAuth  string
	tagNum   int
	routines int

	rootCmd = &cobra.Command{
		Use:   "Example: run --tcrName=test-transfer --secretId=xxxx --secretKey=xxxx --regionName=ap-guangzhou --ccrAuth=user:pass --tcrAuth=user:pass --tagNum=50",
		Short: "run",
		Long:  "Image migration tool for CCR and TCR",
		Run:   run(),
	}
)

type renderArgs struct {
	CCRSecretId  string
	CCRSecretKey string
	TCRSecretId  string
	TCRSecretKey string
	TCRDomian    string
	CCRDomain    string
	CCRUsername  string
	CCRPassword  string
	TCRUsername  string
	TCRPassword  string
}

func init() {
	rootCmd.Flags().StringVar(&secretId, "secretId", "",
		"yunapi secret Id for ccr and tcr")
	rootCmd.Flags().StringVar(&secretKey, "secretKey", "",
		"yunapi secret key for ccr and tcr")
	rootCmd.Flags().StringVar(&regionName, "regionName", "ap-guangzhou",
		"yunapi region name for ccr and tcr")

	rootCmd.Flags().StringVar(&ccrSecretId, "ccrSecretId", "",
		"yunapi secret Id for ccr")
	rootCmd.Flags().StringVar(&ccrSecretKey, "ccrSecretKey", "",
		"yunapi secret key for ccr")
	rootCmd.Flags().StringVar(&tcrSecretId, "tcrSecretId", "",
		"yunapi secret Id for tcr")
	rootCmd.Flags().StringVar(&tcrSecretKey, "tcrSecretKey", "",
		"yunapi secret key for tcr")

	rootCmd.Flags().StringVar(&ccrRegionName, "ccrRegionName", regionName,
		"yunapi region name for ccr")
	rootCmd.Flags().StringVar(&tcrRegionName, "tcrRegionName", regionName,
		"yunapi region name for tcr")

	rootCmd.Flags().StringVar(&tcrName, "tcrName", "",
		"tcr instance name")
	rootCmd.Flags().StringVar(&ccrAuth, "ccrAuth", "",
		"ccr auth secret, format is username:password")
	rootCmd.Flags().StringVar(&tcrAuth, "tcrAuth", "",
		"tcr auth secret, format is username:password")
	rootCmd.Flags().IntVar(&tagNum, "tagNum", 100,
		"number of recent tags in migration")
	rootCmd.Flags().IntVar(&routines, "routines", 5,
		"number of concurrent task in migration")

}

func generateRenderArgs() (*renderArgs, error) {
	ccrIdx := strings.Index(ccrAuth, ":")
	if ccrIdx == -1 {
		return nil, errors.New("Invalid ccrAuthSecret")
	}
	tcrIdx := strings.Index(tcrAuth, ":")
	if tcrIdx == -1 {
		return nil, errors.New("Invalid tcrAuthSecret")
	}
	ccrDomainPrefix, ok := ccrapis.RegionPrefix[ccrRegionName]
	if !ok {
		return nil, errors.New("Invalid ccrRegionName")
	}
	rargs := renderArgs{
		CCRSecretId:  ccrSecretId,
		CCRSecretKey: ccrSecretKey,
		TCRSecretId:  tcrSecretId,
		TCRSecretKey: tcrSecretKey,
		TCRDomian:    fmt.Sprintf("%s.tencentcloudcr.com", tcrName),
		CCRDomain:    fmt.Sprintf("%s.ccs.tencentyun.com", ccrDomainPrefix),
		CCRUsername:  ccrAuth[:ccrIdx],
		CCRPassword:  ccrAuth[ccrIdx+1:],
		TCRUsername:  tcrAuth[:tcrIdx],
		TCRPassword:  tcrAuth[tcrIdx+1:],
	}

	return &rargs, nil
}

func render(args *renderArgs) error {

	renderInternal := func(templateStr, path string) error {

		tmpl, err := template.New(path).Parse(templateStr)
		if err != nil {
			log.Errorf("template.Parse error: %v", err)
			return err
		}
		outputFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
		if err != nil {
			log.Errorf("os.OpenFile error: %v", err)
			return err
		}
		err = tmpl.Execute(outputFile, args)
		if err != nil {
			log.Errorf("tmpl.Execute error: %v", err)
			return err
		}
		return nil
	}

	// render secret.yaml
	if err := renderInternal(secretFileTemplate, secretFilePath); err != nil {
		return err
	}
	// render security.yaml
	if err := renderInternal(securityFileTemplate, securityFilePath); err != nil {
		return err
	}
	return nil
}

func validateArgs() {
	if ccrSecretId == "" || ccrSecretKey == "" || tcrSecretId == "" || tcrSecretKey == "" {
		log.Error("Require ccrSecretId or secretId, ccrSecretKey or secretKey")
		os.Exit(1)
	}
	if ccrRegionName == "" || tcrRegionName == "" {
		log.Error("Require ccrRegionName,tcrRegionName or regionName")
		os.Exit(1)
	}
	if tcrName == "" {
		log.Error("Require tcrName")
		os.Exit(1)
	}

	if ccrAuth == "" {
		log.Error("Require ccrAuth")
		os.Exit(1)
	}
	if tcrAuth == "" {
		log.Error("Require tcrAuth")
		os.Exit(1)
	}

}

func run() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		// adapt same region or not
		if ccrSecretId == "" {
			ccrSecretId = secretId
		}
		if ccrSecretKey == "" {
			ccrSecretKey = secretKey
		}
		if tcrSecretId == "" {
			tcrSecretId = secretId
		}
		if tcrSecretKey == "" {
			tcrSecretKey = secretKey
		}
		if ccrRegionName == "" {
			ccrRegionName = regionName
		}
		if tcrRegionName == "" {
			tcrRegionName = regionName
		}

		validateArgs()

		rargs, err := generateRenderArgs()
		if err != nil {
			log.Errorf("generateRenderArgs error: %v", err)
			os.Exit(1)

		}
		if err := render(rargs); err != nil {
			log.Error("Render template error")
			os.Exit(1)
		}
		commandArgs := fmt.Sprintf("--ccrToTcr=true --retry=3 --routines=%d --ccrTagNums=%d --tcrName=%s --ccrRegion=%s --tcrRegion=%s --securityFile=%s --secretFile=%s",
			routines, tagNum, tcrName, ccrRegionName, tcrRegionName, securityFilePath, secretFilePath)

		transferCMD := exec.Command(binaryFilePath, strings.Split(commandArgs, " ")...)
		transferCMD.Stderr = os.Stderr
		transferCMD.Stdout = os.Stdout
		if err := transferCMD.Run(); err != nil {
			log.Errorf("transferCMD.Run() error: %v", err)
			os.Exit(1)
		}
	}
}

func main() {
	rootCmd.Execute()
}
