package main

import (
	"fmt"
	"os"
	"path"

	"github.com/google/uuid"
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
		Use:   "devservicerestorefromprodarch",
		Short: "Clear and load a dev service from a snapshot of prod services within a prod archive.",
		Long:  "Clear a dev monoservice and load data from an archive containing a snapshot of prod microservices.",
		Run: func(cmd *cobra.Command, args []string) {
			CMDDevServiceRestoreFromProd(v)
		},
	}

	FlagsAddKubeFlags(c, v)
	MAIN.AddCommand(c)
}

func CMDDevServiceRestoreFromProd(v *viper.Viper) {
	// pick a snapshot set
	var srcArchiveFileSet *schema.ArchiveFileSet
	{
		srcArchiveSet := schema.ArchiveSetNew()
		for _, srcArchiveSpec := range localProdArchiveSpecs {
			srcArchiveSet.ArchiveAdd(srcArchiveSpec)
		}

		err := srcArchiveSet.FilesFetch(nil)
		if err != nil {
			core.Log.Fatalf("failed to get files for cluster archive set: %v", err)
		}

		var hasFiles bool
		for _, srcArchive := range srcArchiveSet.Archives {
			if len(srcArchive.Files) > 0 {
				hasFiles = true
				break
			}
		}
		if !hasFiles {
			core.Log.Fatalf("found no snapshots in %v", localProdArchiveSpecs)
		}

		picker := ArchiveFileSetPickerNew()
		picker.ArchiveSetPut(srcArchiveSet)
		picker.Run()
		srcArchiveFileSet = picker.SelectedSnapshotArchiveFileSet
	}
	if srcArchiveFileSet == nil {
		core.Log.Fatalf("snapshot not picked... cancelling operation")
	}

	// get the localDevService
	var devService *schema.Service
	{
		devService = &schema.Service{}
		err := devService.Parse(localDevServiceSpec)
		if err != nil {
			core.Log.Fatalf("could not parse the localDevServiceSpec: %v", err)
		}
	}

	// clear the dev service data
	{
		reqURL := fmt.Sprintf("http://%s:%d/v1/Reset", devService.Host, devService.Port)
		reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Reset", "Body": {} }`, uuid.NewString())
		if res, err := HTTPPost(reqURL, "application/json", reqBod); err != nil {
			core.Log.Fatalf("reset dev service error: %s: %v", reqURL, err)
		} else {
			core.Log.Warnf("reset dev service success: %s", res)
		}
	}

	// load one archive at a time
	{
		for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
			// clear the content of the restore folder
			if err := os.RemoveAll(localDevRestoreFolder); err != nil {
				core.Log.Fatalf("cound not clear the content of the restore folder: %v", err)
			}

			// recreate it
			if err := os.MkdirAll(localDevRestoreFolder, 0774); err != nil {
				core.Log.Fatalf("cound not create the restore folder: %v", err)
			}

			// symlink to the the srcArchive
			{
				srcArchiveFilePath := srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name
				dstArchiveFilePath := localDevRestoreFolder + "/" + srcArchiveFile.Name
				err := os.Symlink(srcArchiveFilePath, dstArchiveFilePath)
				if err != nil {
					core.Log.Fatalf("cound not create symlink: src %s to %s: %v", srcArchiveFilePath, dstArchiveFilePath, err)
				}

				core.Log.Warnf("restoring %s", dstArchiveFilePath)
			}

			// run the restore endpoint
			{
				var reqURL string
				if path.Base(srcArchiveFile.Archive.Path) == "dockie" {
					reqURL = fmt.Sprintf("http://%s:%d/v1/Restore/Dockie", devService.Host, devService.Port)
				} else {
					reqURL = fmt.Sprintf("http://%s:%d/v1/Restore/Other", devService.Host, devService.Port)
				}
				reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Restore", "Body": {} }`, uuid.NewString())
				if res, err := HTTPPost(reqURL, "application/json", reqBod); err != nil {
					core.Log.Fatalf("could not restore with %s: %v", reqURL, err)
				} else {
					core.Log.Warnf("finished. got this: %s", res)
				}
			}
		}
	}
}
