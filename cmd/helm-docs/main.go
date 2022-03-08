package main

import (
	"os"
	"path"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/norwoodj/helm-docs/pkg/document"
	"github.com/norwoodj/helm-docs/pkg/helm"
)

func retrieveInfoAndPrintDocumentation(chartDirectory string, chartSearchRoot string, templateFiles []string, customValuesFiles []string, waitGroup *sync.WaitGroup, dryRun bool) {
	defer waitGroup.Done()
	chartDocumentationInfo, err := helm.ParseChartInformation(path.Join(chartSearchRoot, chartDirectory), customValuesFiles)

	if err != nil {
		log.Warnf("Error parsing information for chart %s, skipping: %s", chartDirectory, err)
		return
	}

	document.PrintDocumentation(chartDocumentationInfo, chartSearchRoot, templateFiles, dryRun, version)

}

func helmDocs(cmd *cobra.Command, _ []string) {
	initializeCli()

	chartSearchRoot := viper.GetString("chart-search-root")
	var fullChartSearchRoot string

	if path.IsAbs(chartSearchRoot) {
		fullChartSearchRoot = chartSearchRoot
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			log.Warnf("Error getting working directory: %s", err)
			return
		}

		fullChartSearchRoot = path.Join(cwd, chartSearchRoot)
	}

	customValuesFiles := viper.GetStringSlice("custom-values-file")
	log.Debugf("Searching for custom values files called [%s]", strings.Join(customValuesFiles, ", "))

	chartDirs, err := helm.FindChartDirectories(fullChartSearchRoot)
	if err != nil {
		log.Errorf("Error finding chart directories: %s", err)
		os.Exit(1)
	}

	log.Infof("Found Chart directories [%s]", strings.Join(chartDirs, ", "))

	templateFiles := viper.GetStringSlice("template-files")
	log.Debugf("Rendering from optional template files [%s]", strings.Join(templateFiles, ", "))

	dryRun := viper.GetBool("dry-run")
	waitGroup := sync.WaitGroup{}

	for _, c := range chartDirs {
		waitGroup.Add(1)

		// On dry runs all output goes to stdout, and so as to not jumble things, generate serially
		if dryRun {
			retrieveInfoAndPrintDocumentation(c, fullChartSearchRoot, templateFiles, customValuesFiles, &waitGroup, dryRun)
		} else {
			go retrieveInfoAndPrintDocumentation(c, fullChartSearchRoot, templateFiles, customValuesFiles, &waitGroup, dryRun)
		}
	}

	waitGroup.Wait()
}

func main() {
	command, err := newHelmDocsCommand(helmDocs)
	if err != nil {
		log.Errorf("Failed to create the CLI commander: %s", err)
		os.Exit(1)
	}

	if err := command.Execute(); err != nil {
		log.Errorf("Failed to start the CLI: %s", err)
		os.Exit(1)
	}
}
