package main

import (
	"os"
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
		Use:   "badgerbackup",
		Short: "Backup a badger DB to STDOUT",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDBadgerBackup(v)
		},
	}

	CMDDBConfig(c, v)
	MAIN.AddCommand(c)
}

func CMDBadgerBackup(v *viper.Viper) {
	dbBadger := CMDDBRun(v)
	stream := dbBadger.DB.NewStream()

	start := time.Now()
	core.Log.Warnf("DBBadgerSnapshot.Backup: starting")

	// Run the stream (discard backup time)
	_, err := stream.Backup(os.Stdout, 1)
	if err != nil {
		core.Log.Errorf("error doing badger db backup: %w", err)
	}

	duration := time.Since(start)
	core.Log.Warnf("DBBadgerSnapshot.Backup: took %s", duration.String())
}
