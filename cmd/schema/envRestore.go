package schema

import (
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
)

func EnvRestore(kubeClient *kube.Client, srcArchiveSpecs, dstServiceSpecs []string) {
	var err error

	// get src and dst archiveSets and serviceSets from specs
	var srcArchiveSet *ArchiveSet
	var dstServiceSet *ServiceSet
	{
		srcArchiveSet = ArchiveSetNew()
		err := srcArchiveSet.ArchiveAddAll(srcArchiveSpecs, "")
		if err != nil {
			core.Log.Fatalf("could not add srcArchive %v", err)
		}

		// get dstServices
		dstServiceSet = ServiceSetNew()
		err = dstServiceSet.ServiceAddAll(dstServiceSpecs)
		if err != nil {
			core.Log.Fatalf("could not add dstArchive %v", err)
		}
	}

	// let the user pick a srcArchiveFileSet (snapshot)
	srcArchiveFileSet, err := srcArchiveSet.PickSnapshot()
	if err != nil {
		core.Log.Fatalf("snapshot not picked... cancelling operation")
	}

	// servicesReset tracks which services have been reset
	servicesReset := make(map[string]bool)

	// one archive at a time...
	for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
		// get the dstArchive and dstService
		var (
			dstArchive *Archive
			dstService *Service
		)

		dstService, err = dstServiceSet.ServiceGetByServiceName(srcArchiveFile.Archive.ServiceName)
		if err != nil {
			core.Log.Fatalf("could not find dstService to match srcArchiveFile '%s': %v", srcArchiveFile.Name, err)
		}

		err = dstService.Stage(kubeClient, srcArchiveFile)
		if err != nil {
			core.Log.Fatalf("could not stage %s to %s: %v", srcArchiveFile.Name, dstArchive.Spec, err)
		}

		// reset the service
		// we do this deduping because sometimes we multiplex many
		// service snapshots / backups into a single service (eg. prod to dev)
		if _, ok := servicesReset[dstService.KubeName]; !ok {
			servicesReset[dstService.KubeName] = true
			dstService.Reset(kubeClient)
		}

		// run the restore endpoint
		dstService.Restore(kubeClient)
	}

	// finally... reset the raft index of each service. one for each archive.
	raftsReset := make(map[string]bool)
	for _, srcArchiveFile := range srcArchiveFileSet.ArchiveFiles {
		var dstService *Service
		dstService, err = dstServiceSet.ServiceGetByServiceName(srcArchiveFile.Archive.ServiceName)
		if err != nil {
			core.Log.Fatalf("could not find dstService to match srcArchiveFile '%s': %v", srcArchiveFile.Name, err)
		}

		if _, ok := raftsReset[dstService.Name]; !ok {
			raftsReset[dstService.Name] = true
			dstService.RAFTReset(kubeClient)
		}
	}
}
