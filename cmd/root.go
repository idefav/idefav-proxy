package cmd

import (
	"github.com/spf13/cobra"
	"idefav-proxy/cmd/mgr"
	_ "idefav-proxy/cmd/proxy"
	"idefav-proxy/cmd/server"
	"idefav-proxy/cmd/upgrade"
	"idefav-proxy/pkg/log"
)

var RootCmd = &cobra.Command{
	Use:   "idefav-proxy",
	Short: "idefav proxy是一个代理服务",
	Long:  `Idefav Proxy 是一个高性能代理服务`,
	Run: func(cmd *cobra.Command, args []string) {
		err := server.IdefavServerManager.Startup()
		if err != nil {
			log.Fatal(err)
		}

		upgrade.Ready()

		upgrade.Stop(func() {
			server.IdefavServerManager.Shutdown()
		})
	},
}

func Execute() {
	RootCmd.Execute()
}
func init() {
	RootCmd.AddCommand(mgr.VersionCmd)
}
