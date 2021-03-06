package rancher

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/invitae-ankit/ylp/util"
	"github.com/rancherio/go-rancher/client"
	"github.com/spf13/cobra"
)

var (
	stack         string
	clientOptions client.ClientOpts
)

func New() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "rancher",
		Short: "Rancher related helper commands",
		Run: func(cmd *cobra.Command, args []string) {
			process()
		},
	}
	cmd.Flags().StringVarP(&stack, "stack", "s", "", "Name of the stack")
	cmd.MarkFlagRequired("stack")
	return cmd
}

func process() {
	readPropfile()
	api, err := client.NewRancherClient(&clientOptions)
	util.HandleError(err)
	fmt.Println(api)
}

func readPropfile() {
	homeDir, err := os.UserHomeDir()
	util.HandleError(err)

	prop, err := os.ReadFile(path.Join(homeDir, ".rancher/dev.json"))
	util.HandleError(err)

	json.Unmarshal(prop, &clientOptions)
}
