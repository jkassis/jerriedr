package main

import (
	"log"

	badger "github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerrie/core/kittie"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// A general configuration object (feed with flags, conf files, etc.)
	v := viper.New()

	// CLI Command with flag parsing
	c := &cobra.Command{
		Use:   "raftIndexSet",
		Short: "Set the index of the last processed raft proposal",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDRaftIndexSet(v)
		},
	}

	c.PersistentFlags().StringP("dbDir", "d", "", "database dir")
	v.BindPFlag("dbDir", c.PersistentFlags().Lookup("dbDir"))

	c.PersistentFlags().IntP("index", "i", 0, "target index")
	v.BindPFlag("index", c.PersistentFlags().Lookup("index"))

	MAIN.AddCommand(c)
}

func CMDRaftIndexSet(v *viper.Viper) {
	dbDir := v.GetString("dbDir")
	index := uint64(v.GetInt64("index"))

	core.Log.Warnf("opening database at %s/data", dbDir)
	opts := badger.DefaultOptions(dbDir)
	opts = opts.WithLogger(core.Log)
	opts = opts.WithSyncWrites(false)
	opts = opts.WithValueLogLoadingMode(options.FileIO)
	opts = opts.WithTableLoadingMode(options.FileIO)
	opts = opts.WithMaxCacheSize(1 << 27)
	opts = opts.WithNumVersionsToKeep(0)
	dbBadger := core.NewDBBadger(&opts, core.Log)

	writeErr := dbBadger.TxnW(func(dbTxn core.DBTxn) error {
		proposalIdxK := kittie.DBRaftProposalIDXK
		proposalIdxV := &core.DBInt64V{Value: index}
		return dbTxn.ObjPut(proposalIdxK, proposalIdxV, 0)
	})

	if writeErr != nil {
		log.Fatalf("%v", writeErr)
	}
}
