package main

import (
	"fmt"
	"os"

	_ "embed"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/jkassis/jerrie/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MAIN represents the base command when called without any subcommands
var MAIN = &cobra.Command{
	Use:   "jerriedr",
	Short: "A CLI for operations on jerrie services.",
}

func main() {
	err := MAIN.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const (
	FLAG_DB_DIR          = "dbDir"
	FLAG_SERVER_HOSTPORT = "serverHostport"
	FLAG_SERVER_SCHEME   = "http"
)

func CMDDBConfig(c *cobra.Command, v *viper.Viper) {
	core.Log.Warnf("CMDDBConfig")
	c.PersistentFlags().StringP(FLAG_DB_DIR, "d", "", "database dir")
	c.MarkPersistentFlagRequired(FLAG_DB_DIR)
	v.BindPFlag(FLAG_DB_DIR, c.PersistentFlags().Lookup(FLAG_DB_DIR))
}

func CMDServerConfig(c *cobra.Command, v *viper.Viper) {
	core.Log.Warnf("CMDServerConfig")
	c.PersistentFlags().StringP(FLAG_SERVER_HOSTPORT, "u", "localhost:10100", "server hostport")
	// c.MarkPersistentFlagRequired(FLAG_SERVER_HOSTPORT)
	v.BindPFlag(FLAG_SERVER_HOSTPORT, c.PersistentFlags().Lookup(FLAG_SERVER_HOSTPORT))

	c.PersistentFlags().StringP(FLAG_SERVER_SCHEME, "s", "http", "server scheme: http | https")
	// c.MarkPersistentFlagRequired(FLAG_SERVER_SCHEME)
	v.BindPFlag(FLAG_SERVER_SCHEME, c.PersistentFlags().Lookup(FLAG_SERVER_SCHEME))
}

func CMDDBRun(v *viper.Viper) *core.DBBadger {
	dbDir := v.GetString(FLAG_DB_DIR)

	core.Log.Warnf("opening database at %s", dbDir)
	opts := badger.DefaultOptions(dbDir)
	opts = opts.WithLogger(core.Log)
	opts = opts.WithSyncWrites(false)
	opts = opts.WithValueLogLoadingMode(options.FileIO)
	opts = opts.WithTableLoadingMode(options.FileIO)
	opts = opts.WithMaxCacheSize(1 << 27)
	opts = opts.WithNumVersionsToKeep(0)
	dbBadger := core.NewDBBadger(&opts, core.Log)
	return dbBadger
}
