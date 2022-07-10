package main

import (
	"fmt"
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

	c := core.DBInt64{
		K: kittie.DBRaftProposalIDXK,
		V: &core.DBInt64V{Value: uint64(v.GetInt64(FLAG_INDEX))},
	}
	if err := dbBadger.TxnW(func(dbTxn core.DBTxn) error {
		return dbTxn.ObjPut(c.K, c.V, 0)
	}); err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Printf("Raft proposal index in DB set to %d", c.V.Value)
}
