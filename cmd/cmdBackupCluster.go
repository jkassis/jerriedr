package main

import (
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "backupcluster",
		Short: "Backup cluster services using http backup reqeusts",
		Long:  `This command is a shortcut for backupremote.`,
		Run: func(cmd *cobra.Command, args []string) {
			CMDBackupCluster(v)
		},
	}

	// kube
	CMDKubeConfig(c, v)
	CMDProtocolConfig(c, v)
	CMDVersionConfig(c, v)

	MAIN.AddCommand(c)
}

func CMDBackupCluster(v *viper.Viper) {
	start := time.Now()
	core.Log.Warnf("BackupCluster: starting")

	// for each service
	serviceSpecs := []string{
		"kube|fg/dockie-0:10000",
	}
	BackupFromServiceSpecs(v, serviceSpecs)

	duration := time.Since(start)
	core.Log.Warnf("BackupCluster: took %s", duration.String())
}
