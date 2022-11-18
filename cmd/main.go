package main

import (
	"fmt"
	"os"

	_ "embed"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
	FLAG_KUBE_MASTER_URL  = "kubeMasterURL"
	FLAG_KUBE_CONFIG_PATH = "kubeConfigPath"
	FLAG_DB_DIR           = "dbDir"
	FLAG_HOSTPORT         = "serverHostport"
	FLAG_PROTOCOL         = "protocol"
	FLAG_VERSION          = "version"
	FLAG_SRC_ARCHIVE      = "srcArchive"
	FLAG_DST_ARCHIVE      = "dstArchive"
	FLAG_RESTORE_ARCHIVE  = "restoreArchive"
)

func FlagsAddDBFlags(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_DB_DIR, "d", "", "database dir")
	c.MarkPersistentFlagRequired(FLAG_DB_DIR)
	v.BindPFlag(FLAG_DB_DIR, c.PersistentFlags().Lookup(FLAG_DB_DIR))
}

func FlagsAddKubeFlags(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_KUBE_CONFIG_PATH, "c", "", "absolute path to the kubernetes config file")
	// c.MarkPersistentFlagRequired(FLAG_KUBE)
	v.BindPFlag(FLAG_KUBE_CONFIG_PATH, c.PersistentFlags().Lookup(FLAG_KUBE_CONFIG_PATH))

	c.PersistentFlags().StringP(FLAG_KUBE_MASTER_URL, "m", "https://api.live.shinetribe.media:6443", "URL to the kubernetes master")
	// c.MarkPersistentFlagRequired(FLAG_KUBE)
	v.BindPFlag(FLAG_KUBE_MASTER_URL, c.PersistentFlags().Lookup(FLAG_KUBE_MASTER_URL))
}

func FlagsAddHostFlags(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_HOSTPORT, "u", "localhost:10000", "server hostport")
	// c.MarkPersistentFlagRequired(FLAG_SERVER_HOSTPORT)
	v.BindPFlag(FLAG_HOSTPORT, c.PersistentFlags().Lookup(FLAG_HOSTPORT))
}

func FlagsAddProtocolFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_PROTOCOL, "p", "http", "protocol: http | https")
	// c.MarkPersistentFlagRequired(FLAG_PROTOCOL)
	v.BindPFlag(FLAG_PROTOCOL, c.PersistentFlags().Lookup(FLAG_PROTOCOL))
}

func FlagsAddAPIVersionFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_VERSION, "v", "v1", "backup protocol version")
	v.BindPFlag(FLAG_VERSION, c.PersistentFlags().Lookup(FLAG_VERSION))
}

func FlagsAddSrcArchiveFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_SRC_ARCHIVE, "sa", "", "source archive")
	c.MarkPersistentFlagRequired(FLAG_SRC_ARCHIVE)
	v.BindPFlag(FLAG_SRC_ARCHIVE, c.PersistentFlags().Lookup(FLAG_SRC_ARCHIVE))
}

func FlagsAddDstArchiveFlag(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_DST_ARCHIVE, "da", "", "destination archive")
	c.MarkPersistentFlagRequired(FLAG_DST_ARCHIVE)
	v.BindPFlag(FLAG_DST_ARCHIVE, c.PersistentFlags().Lookup(FLAG_DST_ARCHIVE))
}

func FlagsAddRestoreArchivesDir(c *cobra.Command, v *viper.Viper) {
	c.PersistentFlags().StringP(FLAG_RESTORE_ARCHIVE, "ra", "/tmp/jerrie/restore", "restore archive")
	c.MarkPersistentFlagRequired(FLAG_RESTORE_ARCHIVE)
	v.BindPFlag(FLAG_RESTORE_ARCHIVE, c.PersistentFlags().Lookup(FLAG_RESTORE_ARCHIVE))
}

func KubeClientGet(v *viper.Viper) (*kube.KubeClient, error) {
	// use the current context in kubeconfig
	kubeMasterURL := v.GetString(FLAG_KUBE_MASTER_URL)
	kubeConfigPath := v.GetString(FLAG_KUBE_CONFIG_PATH)
	return kube.NewKubeClient(kubeMasterURL, kubeConfigPath)
}

func KubeConfGet(v *viper.Viper) (*restclient.Config, error) {
	// use the current context in kubeconfig
	kubeConfigPath := v.GetString(FLAG_KUBE_CONFIG_PATH)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("could not read kube config from %s: %w", kubeConfigPath, err)
	}
	return kubeConfig, nil
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
