package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCMD = &cobra.Command{
	Use:   "meshviewer-collector",
	Short: "collector multiple meshviewer.json (ffrgb) to a single on",
}

func Execute() {
	if err := RootCMD.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
