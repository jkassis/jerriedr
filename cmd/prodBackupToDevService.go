package main

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// conf for the dev service
var prodBackupToDevServiceSpecs []string = []string{
	"local|dockie|10001|/v1/Backup|/v1/Restore/Dockie|/var/multi/single/local-server-0/restore",
	"local|dubbie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0/restore",
	"local|keevie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0/restore",
	"local|ledgie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0/restore",
	"local|permie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0/restore",
	"local|tickie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0/restore",
}

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "prodBackupToDevService",
		Short: "",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			kubeClient, err := KubeClientGet(v)
			if err != nil {
				core.Log.Warnf("could not init kubeClient: %v", err)
			}

			srcArchiveSpecs := prodBackupArchiveSpecs
			dstServiceSpecs := prodBackupToDevServiceSpecs
			schema.EnvRestore(kubeClient, srcArchiveSpecs, dstServiceSpecs)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}
