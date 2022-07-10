package main

import (
	"fmt"
	"log"

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
		Use:   "raftIndexGet",
		Short: "Set the index of the last processed raft proposal",
		// Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			CMDRaftIndexGet(v)
		},
	}

	CMDDBConfig(c, v)

	MAIN.AddCommand(c)
}

func CMDRaftIndexGet(v *viper.Viper) {
	dbBadger := CMDDBRun(v)

	index := uint64(v.GetInt64(FLAG_INDEX))
	writeErr := dbBadger.TxnR(func(dbTxn core.DBTxn) error {
		proposalIdxK := kittie.DBRaftProposalIDXK
		proposalIdxV := &core.DBInt64V{Value: index}
		return dbTxn.ObjGet(proposalIdxK, proposalIdxV)
	})

	if writeErr != nil {
		log.Fatalf("%v", writeErr)
	}

	fmt.Printf("Raft proposal index in DB is %d", index)
}
