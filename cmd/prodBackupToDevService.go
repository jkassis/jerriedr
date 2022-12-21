package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// conf for the dev service
var prodBackupToDevServiceSpecs []string = []string{
	"local|dockie|10001|/v1/Backup|/v1/Restore/Dockie|/var/multi/single/local-server-0",
	"local|dubbie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0",
	"local|keevie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0",
	"local|ledgie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0",
	"local|permie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0",
	"local|tickie|10001|/v1/Backup|/v1/Restore/Other|/var/multi/single/local-server-0",
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
			CMDProdBackupToDevService(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDProdBackupToDevService(v *viper.Viper) {
	srcArchiveSpecs := prodBackupArchiveSpecs
	dstServiceSpecs := prodBackupToDevServiceSpecs
	EnvRestore(v, srcArchiveSpecs, dstServiceSpecs)
}
