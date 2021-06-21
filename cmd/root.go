package cmd

import (
	"github.com/invitae-ankit/ylp/cmd/rancher"
	"github.com/invitae-ankit/ylp/cmd/vault"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ylp",
	Short: "Help managinng LIMS Classis project",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.AddCommand(vault.Cmd)
	rootCmd.AddCommand(rancher.Cmd)
}
