package cmd

import (
	"fmt"
	"os"

	"github.com/drakeaharper/gerrit-cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	verbose   bool
	version   string
	buildTime string
)

var rootCmd = &cobra.Command{
	Use:   "gerry",
	Short: "A CLI tool for Gerrit Code Review",
	Long: `gerry is a command-line interface for interacting with Gerrit Code Review.
It provides a terminal-friendly way to list changes, view comments, fetch code,
and manage your code review workflow without leaving your terminal.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			utils.SetLogLevel(utils.DebugLevel)
		}
	},
}

func Execute(ver, build string) error {
	version = ver
	buildTime = build
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gerry/config.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(teamCmd)
	rootCmd.AddCommand(commentsCmd)
	rootCmd.AddCommand(detailsCmd)
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(cherryPickCmd)
	rootCmd.AddCommand(treeCmd)
	rootCmd.AddCommand(treesCmd)
	rootCmd.AddCommand(failuresCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(retriggerCmd)
	rootCmd.AddCommand(shareCmd)
	rootCmd.AddCommand(rebaseCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.gerry")
		viper.SetConfigName("config")
		viper.SetConfigType("json")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		utils.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of gerry",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gerry version %s (built %s)\n", version, buildTime)
	},
}
