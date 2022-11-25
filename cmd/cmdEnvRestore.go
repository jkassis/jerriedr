package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/schema"
	"github.com/spf13/viper"
)

func EnvRestore(v *viper.Viper, srcArchiveSpecs, dstArchiveSpecs, dstServiceSpecs []string) {
	var err error

	// get kube client
	kubeClient, _ := KubeClientGet(v)

	// get src and dst archiveSets and serviceSets
	var srcArchiveSet, dstArchiveSet *schema.ArchiveSet
	var dstServiceSet *schema.ServiceSet
	{
		srcArchiveSet := schema.ArchiveSetNew()
		err := srcArchiveSet.ArchiveAddAll(srcArchiveSpecs, "")
		if err != nil {
			core.Log.Fatalf("could not add srcArchive %v", err)
		}

		// get dstArchiveSet
		dstArchiveSet := schema.ArchiveSetNew()
		err = dstArchiveSet.ArchiveAddAll(dstArchiveSpecs, "restore")
		if err != nil {
			core.Log.Fatalf("could not add dstArchive %v", err)
		}

		// get dstServices
		dstServiceSet := schema.ServiceSetNew()
		err = dstServiceSet.ServiceAddAll(dstServiceSpecs)
		if err != nil {
			core.Log.Fatalf("could not add dstArchive %v", err)
		}
	}

	// pick a srcArchiveFileSet (snapshot)
	var srcArchiveFileSet *schema.ArchiveFileSet
	{
		err = srcArchiveSet.FilesFetch(nil)
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

		if srcArchiveFileSet == nil {
			core.Log.Fatalf("snapshot not picked... cancelling operation")
		}
	}

	// serviceReset resets a service, but only once per run
	servicesReset := make(map[string]bool)
	serviceReset := func(service *schema.Service) {
		if servicesReset[service.KubeName] {
			return
		}
		servicesReset[service.KubeName] = true

		reqURL := fmt.Sprintf("http://%s:%d/v1/Reset", service.Host, service.Port)
		reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Reset", "Body": {} }`, uuid.NewString())
		if res, err := HTTPPost(reqURL, "application/json", reqBod); err != nil {
			core.Log.Fatalf("reset dev service error: %s: %v", reqURL, err)
		} else {
			core.Log.Warnf("reset dev service success: %s", res)
		}
	}

	// restore one archive at a time
	for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
		// get the dstArchive and dstService
		var (
			dstArchive *schema.Archive
			dstService *schema.Service
		)
		{
			dstArchive, err = dstArchiveSet.ArchiveGetByService(srcArchiveFile.Archive.ServiceName)
			if err != nil {
				core.Log.Fatalf("cound find dstArchive to match srcArchiveFile '%s': %v", srcArchiveFile.Name, err)
			}

			dstService, err = dstServiceSet.ServiceGetByServiceName(srcArchiveFile.Archive.ServiceName)
			if err != nil {
				core.Log.Fatalf("cound find dstService to match srcArchiveFile '%s': %v", srcArchiveFile.Name, err)
			}
		}

		// stage the restore file
		{
			err = dstArchive.Stage(kubeClient, srcArchiveFile, dstArchive)
			if err != nil {
				core.Log.Fatalf("could not stage %s to %s: %v", srcArchiveFile.Name, dstArchive.Spec, err)
			}
		}

		// run the restore endpoint
		{
			serviceReset(dstService) // resets once per service
			core.Log.Warnf("restoring %s", srcArchiveFile.Path())
			reqURL := fmt.Sprintf(
				"http://%s:%d/%s",
				dstService.Host,
				dstService.Port,
				dstService.RestoreURL)
			reqBod := fmt.Sprintf(
				`{ "UUID": "%s", "Fn": "/v1/Restore", "Body": {} }`,
				uuid.NewString())
			if res, err := HTTPPost(reqURL, "application/json", reqBod); err != nil {
				core.Log.Fatalf("could not restore with %s: %v", reqURL, err)
			} else {
				core.Log.Warnf("finished. got this: %s", res)
			}
		}
	}
}
