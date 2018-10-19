package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var configValidateCMD = &cobra.Command{
	Use:   "config-validate",
	Short: "start the collecting as deamon",

	Run: func(cmd *cobra.Command, args []string) {
		config, err := loadConfig()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(3)
		}
		if len(config.DataPaths) > 0 {
			os.Exit(0)
		} else {
			fmt.Println("no datapaths found")
			os.Exit(2)
		}
	},
}

func init() {
	configValidateCMD.Flags().StringVarP(&configPath, "config", "c", "config.toml", "Path to configuration file")
	RootCMD.AddCommand(configValidateCMD)
}
