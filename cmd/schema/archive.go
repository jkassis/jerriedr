package schema

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/kube"
	"golang.org/x/sync/errgroup"
)

const FLAG_ARCHIVE = "archive"

func ArchiveNew() *Archive {
	archive := &Archive{}
	archive.Files = make([]*ArchiveFile, 0)
	archive.Filters = make([]*TimeFilter, 0)
	return archive
}

type Archive struct {
	Files         []*ArchiveFile
	FilesFiltered []*ArchiveFile
	Filters       []*TimeFilter
	Host          string
	KubeContainer string
	KubeName      string
	KubeNamespace string
	Path          string
	Parent        *Archive
	Scheme        string
	Spec          string
}

func (a *Archive) Parse(spec string) error {
	parts := strings.Split(spec, "|")
	a.Scheme = parts[0]
	if a.Scheme == "statefulset" {
		err := fmt.Errorf("%s must be statefulset|<kubeNamespace>/<kubeName>|<pathPattern> where <pathPattern> can contain '<pod>' to insert the pod Name", spec)

		if len(parts) != 3 {
			return err
		}

		statefulSet := parts[1]
		if !strings.Contains(statefulSet, "/") {
			return err
		}

		statefulSetParts := strings.Split(statefulSet, "/")
		a.KubeNamespace = statefulSetParts[0]
		if a.KubeNamespace == "" {
			return err
		}

		a.KubeName = statefulSetParts[1]
		if a.KubeName == "" {
			return err
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else if a.Scheme == "pod" {
		err := fmt.Errorf("%s must be pod|<kubeNamespace>/<kubeName>|<path>", spec)

		if len(parts) != 3 {
			return err
		}

		host := parts[1]
		if !strings.Contains(host, "/") {
			return err
		}

		hostParts := strings.Split(host, "/")
		a.KubeNamespace = hostParts[0]
		if a.KubeNamespace == "" {
			return err
		}

		a.KubeName = hostParts[1]
		if a.KubeName == "" {
			return err
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else if a.Scheme == "host" {
		err := fmt.Errorf("%s must be host|<hostName>|<path>", spec)

		if len(parts) != 3 {
			return err
		}
		a.Host = parts[1]
		if a.Host == "" {
			return err
		}

		a.Path = parts[2]
		if a.Path == "" {
			return err
		}
	} else if a.Scheme == "local" {
		err := fmt.Errorf("%s must be local|<path>", spec)

		if len(parts) != 2 {
			return err
		}
		a.Path = parts[1]
		if a.Path == "" {
			return err
		}
	} else {
		return fmt.Errorf("%s must be <scheme>|<schemeSpec> where <scheme> => statefulset | pod | host | local: %s", spec, a.Scheme)
	}
	a.Spec = spec
	return nil
}

func (a *Archive) IsPod() bool {
	return a.Scheme == "pod"
}

func (a *Archive) IsHost() bool {
	return a.Scheme == "host"
}

func (a *Archive) IsLocal() bool {
	return a.Scheme == "local"
}

func (a *Archive) IsStatefulSet() bool {
	return a.Scheme == "statefulset"
}

func (a *Archive) PodArchiveGet(replica int) (*Archive, error) {
	if !a.IsStatefulSet() {
		return nil, fmt.Errorf("%s is not a statefulset spec", a.Spec)
	}
	podName := a.KubeName + "-" + strconv.Itoa(replica)
	podPath := strings.ReplaceAll(a.Path, "<pod>", podName)
	return &Archive{
		Files:         make([]*ArchiveFile, 0),
		Host:          "",
		KubeContainer: a.KubeContainer,
		KubeName:      podName,
		KubeNamespace: a.KubeNamespace,
		Path:          podPath,
		Parent:        a,
		Scheme:        "pod",
		Spec:          fmt.Sprintf("pod|%s/%s|%s", a.KubeNamespace, podName, podPath),
	}, nil
}

func (a *Archive) FilesFetch(kubeClient *kube.KubeClient) error {
	files := make([]*ArchiveFile, 0)

	if a.IsStatefulSet() {
		// get the statefulSet
		statefulSet, err := kubeClient.StatefulSetGetByName(a.KubeNamespace, a.KubeName)
		if err != nil {
			return fmt.Errorf("could not get statefulset %s: %w", a.KubeName, err)
		}

		// for each replica...
		eg := errgroup.Group{}
		replicas := statefulSet.Spec.Replicas
		for i := 0; i < int(*replicas); i++ {
			i := i
			eg.Go(func() error {
				podArchive, err := a.PodArchiveGet(i)
				if err != nil {
					return fmt.Errorf("could not get podArchiveSpec from statefulSetArchiveSpec: %v", err)
				}

				err = podArchive.FilesFetch(kubeClient)
				if err != nil {
					return fmt.Errorf("could not get files for %s of %s: %v", podArchive.Spec, a.Spec, err)
				}
				files = append(files, podArchive.Files...)
				return nil
			})
		}
		if err = eg.Wait(); err != nil {
			return err
		}
	} else if a.IsPod() {
		// get the pod
		pod, err := kubeClient.PodGetByName(a.KubeNamespace, a.KubeName)
		if err != nil {
			return fmt.Errorf("could not get pod: %v", err)
		}

		core.Log.Warnf("fetching file list for pod archive %s", a.Spec)
		podFileNames, err := kubeClient.DirLs(&kube.FileSpec{
			PodNamespace: a.KubeNamespace,
			PodName:      a.KubeName,
			Path:         a.Path,
		}, pod, a.KubeContainer)
		if err != nil {
			return fmt.Errorf("could not list files for podSpec %s: %v", a.Spec, err)
		}

		for _, podFileName := range podFileNames {
			file := &ArchiveFile{
				Archive: a,
				Name:    podFileName,
			}
			err := file.TimestampParseFromName()
			if err == nil {
				files = append(files, file)
				core.Log.Debugf("found %s/%s", a.Spec, podFileName)
			}
		}
	} else if a.IsLocal() {
		dirEntries, err := os.ReadDir(a.Path)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range dirEntries {
			if file.IsDir() {
				continue
			}
			fileName := &ArchiveFile{
				Archive: a,
				Name:    file.Name(),
			}
			err := fileName.TimestampParseFromName()
			if err == nil {
				files = append(files, fileName)
				core.Log.Debugf("found %s/%s", a.Spec, fileName)
			}
		}
	}

	sort.Sort(ByMostRecent(files))
	a.Files = files
	a.FilesFiltered = files
	return nil
}

func (a *Archive) FilterAdd(tf *TimeFilter) {
	a.Filters = append(a.Filters, tf)
}

func (a *Archive) FiltersClear() {
	a.Filters = make([]*TimeFilter, 0)
}

func (a *Archive) Filter() {
	filteredArchiveFiles := make([]*ArchiveFile, 0)

	for _, file := range a.Files {
		for _, filter := range a.Filters {
			if !file.FilterIsOK(filter) {
				continue
			}
		}
		filteredArchiveFiles = append(filteredArchiveFiles, file)
	}

	a.FilesFiltered = filteredArchiveFiles
}

func (ss *Archive) FileGetFilteredBefore(t time.Time) *ArchiveFile {
	for _, file := range ss.FilesFiltered {
		if file.Time.Before(t) {
			return file
		}
	}

	return nil
}

type BySpec []*Archive

func (afs BySpec) Len() int {
	return len(afs)
}

func (afs BySpec) Less(i, j int) bool {
	return afs[i].Spec < afs[j].Spec
}

func (afs BySpec) Swap(i, j int) {
	t := afs[i]
	afs[i] = afs[j]
	afs[j] = t
}
