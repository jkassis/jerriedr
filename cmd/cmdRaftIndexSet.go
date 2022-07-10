package main

import (
	"log"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerrie/core/kittie"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FLAG_INDEX = "index"
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

	CMDDBConfig(c, v)

	c.PersistentFlags().IntP(FLAG_INDEX, "i", 0, "target index")
	c.MarkPersistentFlagRequired(FLAG_INDEX)
	v.BindPFlag(FLAG_INDEX, c.PersistentFlags().Lookup(FLAG_INDEX))

	MAIN.AddCommand(c)
}

func CMDRaftIndexSet(v *viper.Viper) {
	dbBadger := CMDDBRun(v)

	index := uint64(v.GetInt64(FLAG_INDEX))
	writeErr := dbBadger.TxnW(func(dbTxn core.DBTxn) error {
		proposalIdxK := kittie.DBRaftProposalIDXK
		proposalIdxV := &core.DBInt64V{Value: index}
		return dbTxn.ObjPut(proposalIdxK, proposalIdxV, 0)
	})

	if writeErr != nil {
		log.Fatalf("%v", writeErr)
	}
}
