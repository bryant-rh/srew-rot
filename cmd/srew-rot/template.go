package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/bryant-rh/srew-rot/pkg/source"
	"github.com/bryant-rh/srew/pkg/client"
	"github.com/bryant-rh/srew/pkg/index"
	"github.com/bryant-rh/srew/pkg/installation"
	"github.com/bryant-rh/srew/pkg/installation/scanner"
	"github.com/bryant-rh/srew/pkg/installation/validation"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	tagName      string
	templateFile string
	debug        bool
)

var (
	SREW_SERVER_BASEURL  = os.Getenv("SREW_SERVER_BASEURL")
	SREW_SERVER_USERNAME = os.Getenv("SREW_SERVER_USERNAME")
	SREW_SERVER_PASSWORD = os.Getenv("SREW_SERVER_PASSWORD")
	Client               *client.SrewClient
)

func init() {
	rootCmd.AddCommand(templateCmd)

	templateCmd.Flags().StringVar(&tagName, "tag", "", "tag name to use for templating")
	templateCmd.MarkFlagRequired("tag")

	templateCmd.Flags().StringVar(&templateFile, "template-file", ".srew.yaml", "template file to use for templating")
	templateCmd.MarkFlagRequired("template-file")

	templateCmd.Flags().BoolVar(&debug, "debug", false, "print debug level logs")
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "template helps validate the krew index template file without going through github actions workflow",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := Validate(cmd, args)
		if err != nil {
			logrus.Fatal(err)
		}

		Client = client.NewGithubClient(SREW_SERVER_BASEURL)

		logrus.Debugf("登录生成token")
		res, err := Client.User_Login(SREW_SERVER_USERNAME, SREW_SERVER_PASSWORD)
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Debugf("执行 LoginWithToken")
		Client.LoginWithToken(res.Data)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		releaseRequest := source.ReleaseRequest{
			TagName: tagName,
		}

		pluginName, spec, err := source.ProcessTemplate(templateFile, releaseRequest)
		if err == nil {
			fmt.Println(pluginName)
			fmt.Println()
			fmt.Println(string(spec))
			fmt.Println()

			logrus.Debugf("Start Validate Plugin Manifest")

			//plugin, err := scanner.ReadPlugin(bytes.NewReader(spec))
			plugin, err := scanner.ReadPlugin(io.NopCloser(bytes.NewReader(spec)))
			if err != nil {
				logrus.Fatal(err)
			}
			// validate plugin manifest
			if err := validation.ValidatePlugin(pluginName, plugin); err != nil {
				logrus.Fatal(err, "plugin validation error")
			}
			logrus.Debugf("structural validation OK")

			// make sure each platform matches a supported platform
			for i, p := range plugin.Spec.Platforms {
				if env := findAnyMatchingPlatform(p.Selector); env.OS == "" || env.Arch == "" {
					logrus.Fatalf("spec.platform[%d]'s selector (%v) doesn't match any supported platforms", i, p.Selector)
				}
			}
			logrus.Debugf("all spec.platform[] items are used")

			// validate no supported <os,arch> is matching multiple platform specs
			if err := isOverlappingPlatformSelectors(plugin.Spec.Platforms); err != nil {
				logrus.Fatal(err, "overlapping platform selectors found")
			}
			logrus.Debugf("no overlapping spec.platform[].selector")

			logrus.Debugf("start create plugin")
			logrus.Debugf("Check if plugin: [%s] already exists", pluginName)

			res, err := Client.ListPlugin(pluginName, "")

			if err != nil {
				if len(res.Data) == 0 {
					logrus.Debugf("plugin: [%s] is not exists, start create", pluginName)
					res, err := Client.CreatePlugin(plugin)
					if err != nil {
						logrus.Fatal(err)
					}
					logrus.Debugf(res.Msg)

				}
			} else {
				if len(res.Data) != 0 {
					logrus.Debugf("plugin: [%s] is exists, start update", pluginName)
					res, err := Client.UpdatePlugin(plugin)
					if err != nil {
						logrus.Fatal(err)
					}
					logrus.Debugf(res.Msg)
				}

			}

			os.Exit(0)
		}

		if invalidSpecError, ok := err.(source.InvalidPluginSpecError); ok {
			fmt.Println(invalidSpecError.Spec)
			logrus.Fatal(invalidSpecError.Error())
		}

		logrus.Fatal(err)
	},
}

func Validate(cmd *cobra.Command, args []string) error {

	if SREW_SERVER_BASEURL == "" {
		return fmt.Errorf("环境变量: [SREW_SERVER_BASEURL] 为空:'%s',请设置", SREW_SERVER_BASEURL)
	}
	if SREW_SERVER_USERNAME == "" {
		return fmt.Errorf("环境变量: [SREW_SERVER_USERNAME] 为空:'%s',请设置", SREW_SERVER_USERNAME)
	}
	if SREW_SERVER_PASSWORD == "" {
		return fmt.Errorf("环境变量: [SREW_SERVER_PASSWORD] 为空:'%s',请设置", SREW_SERVER_PASSWORD)
	}
	return nil

}

// findAnyMatchingPlatform finds an <os,arch> pair matches to given selector
func findAnyMatchingPlatform(selector *metav1.LabelSelector) installation.OSArchPair {
	for _, p := range allPlatforms() {
		if selectorMatchesOSArch(selector, p) {
			logrus.Debugf("%s MATCHED <%s>", selector, p)
			return p
		}
		logrus.Debugf("%s didn't match <%s>", selector, p)
	}
	return installation.OSArchPair{}
}

// isOverlappingPlatformSelectors validates if multiple platforms have selectors
// that match to a supported <os,arch> pair.
func isOverlappingPlatformSelectors(platforms []index.Platform) error {
	for _, env := range allPlatforms() {
		var matchIndex []int
		for i, p := range platforms {
			if selectorMatchesOSArch(p.Selector, env) {
				matchIndex = append(matchIndex, i)
			}
		}

		if len(matchIndex) > 1 {
			return errors.Errorf("multiple spec.platforms (at indexes %v) have overlapping selectors that select %s", matchIndex, env)
		}
	}
	return nil
}

func selectorMatchesOSArch(selector *metav1.LabelSelector, env installation.OSArchPair) bool {
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		// this should've been caught by validation.ValidatePlatform() earlier
		logrus.Debugf("Failed to convert label selector: %+v", selector)
		return false
	}
	return sel.Matches(labels.Set{
		"os":   env.OS,
		"arch": env.Arch,
	})
}

// allPlatforms returns all <os,arch> pairs krew is supported on.
func allPlatforms() []installation.OSArchPair {
	// TODO(ahmetb) find a more authoritative source for this list
	return []installation.OSArchPair{
		{OS: "windows", Arch: "386"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
		{OS: "linux", Arch: "386"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm"},
		{OS: "linux", Arch: "arm64"},
		{OS: "linux", Arch: "ppc64le"},
		{OS: "darwin", Arch: "386"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
	}
}
