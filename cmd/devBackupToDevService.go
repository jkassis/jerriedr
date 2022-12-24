package main

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "devBackupToDevService",
		Short: "",
		Long:  "",
		Run: func(cmd *cobra.Command, args []string) {
			kubeClient, err := KubeClientGet(v)
			if err != nil {
				core.Log.Warnf("could not init kubeClient: %v", err)
			}

			srcArchiveSpecs := devBackupArchiveSpecs
			dstServiceSpecs := devServiceSpecs
			schema.EnvRestore(kubeClient, srcArchiveSpecs, dstServiceSpecs)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}
