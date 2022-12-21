package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// conf for the dev service
var prodToDevServiceSpecs []string = []string{
	"local|dockie|10001|/v1/Backup|/v1/Restore/Dockie",
	"local|dubbie|10001|/v1/Backup|/v1/Restore/Other",
	"local|keevie|10001|/v1/Backup|/v1/Restore/Other",
	"local|ledgie|10001|/v1/Backup|/v1/Restore/Other",
	"local|permie|10001|/v1/Backup|/v1/Restore/Other",
	"local|tickie|10001|/v1/Backup|/v1/Restore/Other",
}

var prodToDevArchiveSpecs []string = []string{
	"local|dockie|/var/multi/single/local-server-0",
	"local|dubbie|/var/multi/single/local-server-0",
	"local|keevie|/var/multi/single/local-server-0",
	"local|ledgie|/var/multi/single/local-server-0",
	"local|permie|/var/multi/single/local-server-0",
	"local|tickie|/var/multi/single/local-server-0",
}

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "prodSnapToProdService",
		Short: "",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			CMDProdBackupToDevService(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDProdSnapToProdService(v *viper.Viper) {
	srcArchiveSpecs := prodBackupArchiveSpecs
	dstArchiveSpecs := prodToDevArchiveSpecs
	dstServiceSpecs := prodToDevServiceSpecs
	EnvRestore(v, srcArchiveSpecs, dstArchiveSpecs, dstServiceSpecs)
}
