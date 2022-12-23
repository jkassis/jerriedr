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
		Use:   "prodSnapToProdBackup",
		Short: "",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			srcArchiveSpecs := prodSnapArchiveSpecs
			dstArchiveSpecs := prodBackupArchiveSpecs
			EnvCopy(v, srcArchiveSpecs, dstArchiveSpecs)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}
