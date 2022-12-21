package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
	dstServiceSpecs := prodBackupToDevServiceSpecs
	EnvRestore(v, srcArchiveSpecs, dstServiceSpecs)
}
