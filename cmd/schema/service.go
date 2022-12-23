package schema

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jkassis/jerrie/core"
	"github.com/jkassis/jerriedr/cmd/http"
	"github.com/jkassis/jerriedr/cmd/kube"
)

func ServiceNew() *Service {
	service := &Service{}
	return service
}

type Service struct {
	BackupURL     string
	Host          string
	KubeContainer string
	KubeName      string
	KubeNamespace string
	Name          string
	Port          int
	RestoreURL    string
	RestorePath   string
	Scheme        string
	Spec          string
}

func (a *Service) Parse(spec string) error {
	parts := strings.Split(spec, "|")
	a.Scheme = parts[0]

	if a.Scheme == "statefulset" {
		err := fmt.Errorf("%s must be statefulset|<kubeNamespace>/<kubeName>(/<container>)?|port|<backupURL>|<restoreURL>|<restoreDirPath>", spec)

		if len(parts) != 6 {
			return err
		}

		{
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
			a.Name = a.KubeName

			if len(statefulSetParts) == 3 {
				a.KubeContainer = statefulSetParts[2]
			}
		}

		{
			port, converr := strconv.Atoi(parts[2])
			if converr != nil {
				return err
			}
			a.Port = port
		}

		a.BackupURL = parts[3]
		if a.BackupURL == "" {
			return err
		}

		a.RestoreURL = parts[4]
		if a.RestoreURL == "" {
			return err
		}

		a.RestorePath = parts[5]
		if a.RestorePath == "" {
			return err
		}
	} else if a.Scheme == "pod" {
		err := fmt.Errorf("%s must be pod|<service>|<kubeNamespace>/<kubeName>(/<container>)?|<path>|<backupURL>|<restoreURL>|<restoreDirPath>", spec)

		if len(parts) != 7 {
			return err
		}

		a.Name = parts[1]
		if a.Name == "" {
			return err
		}

		{
			pod := parts[2]
			if !strings.Contains(pod, "/") {
				return err
			}

			{
				podParts := strings.Split(pod, "/")
				a.KubeNamespace = podParts[0]
				if a.KubeNamespace == "" {
					return err
				}

				a.KubeName = podParts[1]
				if a.KubeName == "" {
					return err
				}

				if len(podParts) == 3 {
					a.KubeContainer = podParts[2]
				}
			}
		}

		{
			port, converr := strconv.Atoi(parts[3])
			if converr != nil {
				return err
			}
			a.Port = port
		}

		a.BackupURL = parts[4]
		if a.BackupURL == "" {
			return err
		}

		a.RestoreURL = parts[5]
		if a.RestoreURL == "" {
			return err
		}

		a.RestorePath = parts[6]
		if a.RestorePath == "" {
			return err
		}
	} else if a.Scheme == "host" {
		err := fmt.Errorf("%s must be host|<service>|<hostName>|<port>|<backupURL>|<restoreURL>|<restorePath>", spec)

		if len(parts) != 7 {
			return err
		}

		a.Name = parts[1]
		if a.Name == "" {
			return err
		}

		{
			a.Host = parts[2]
			if a.Host == "" {
				return err
			}
		}

		{
			port, converr := strconv.Atoi(parts[3])
			if converr != nil {
				return err
			}
			a.Port = port
		}

		a.BackupURL = parts[4]
		if a.BackupURL == "" {
			return err
		}

		a.RestoreURL = parts[5]
		if a.RestoreURL == "" {
			return err
		}

		a.RestorePath = parts[6]
		if a.RestorePath == "" {
			return err
		}
	} else if a.Scheme == "local" {
		err := fmt.Errorf("%s must be local|<service>|<port>|<backupURL>|<restoreURL>|<restorePath>", spec)

		if len(parts) != 6 {
			return err
		}

		a.Host = "localhost"

		a.Name = parts[1]
		if a.Name == "" {
			return err
		}

		{
			port, convErr := strconv.Atoi(parts[2])
			if convErr != nil {
				return err
			}
			a.Port = port
		}

		a.BackupURL = parts[3]
		if a.BackupURL == "" {
			return err
		}
		a.RestoreURL = parts[4]
		if a.RestoreURL == "" {
			return err
		}
		a.RestorePath = parts[5]
		if a.RestorePath == "" {
			return err
		}
	} else {
		return fmt.Errorf("%s must be <scheme>|<schemeSpec> where <scheme> => statefulset | pod | host | local: %s", spec, a.Scheme)
	}

	a.Spec = spec
	return nil
}

func (a *Service) IsPod() bool {
	return a.Scheme == "pod"
}

func (a *Service) IsHost() bool {
	return a.Scheme == "host"
}

func (a *Service) IsLocal() bool {
	return a.Scheme == "local"
}

func (a *Service) IsStatefulSet() bool {
	return a.Scheme == "statefulset"
}

func (a *Service) PodServiceGet(replica int) (*Service, error) {
	if !a.IsStatefulSet() {
		return nil, fmt.Errorf("%s is not a statefulset spec", a.Spec)
	}
	podName := a.KubeName + "-" + strconv.Itoa(replica)
	return &Service{
		Host:          "",
		KubeContainer: a.KubeContainer,
		KubeName:      podName,
		KubeNamespace: a.KubeNamespace,
		Name:          a.Name,
		Port:          a.Port,
		Scheme:        "pod",
		Spec: fmt.Sprintf("pod|%s|%s/%s/%s|%d|%s|%s",
			a.Name,
			a.KubeNamespace,
			podName,
			a.KubeContainer,
			a.Port,
			a.BackupURL,
			a.RestoreURL),
	}, nil
}

func (a *Service) Snap(kubeClient *kube.KubeClient) (err error) {
	core.Log.Warnf("running remote backup for %s", a.Spec)

	var reqURL string
	{
		if a.IsStatefulSet() {
			// we can start the snapshot on any one node since
			b, err := a.PodServiceGet(0)
			if err != nil {
				return err
			}
			return b.Snap(kubeClient)
		} else if a.IsPod() {
			// yes. make sure we have a kube client
			if kubeClient != nil {
				return fmt.Errorf("need kubeClient")
			}

			// forward a local port
			forwardedPort, err := kubeClient.PortForward(&kube.PortForwardRequest{
				LocalPort:    0,
				PodName:      a.KubeName,
				PodNamespace: a.KubeNamespace,
				PodPort:      a.Port,
			})
			if err != nil {
				return fmt.Errorf("could not port forward to kube service %s: %v", a.Spec, err)
			}
			localPort := forwardedPort.Local
			reqURL = fmt.Sprintf("%s://%s:%d/raft/leader/read", "http", "localhost", localPort)
		} else if a.IsHost() {
			reqURL = fmt.Sprintf("http://%s:%d/raft/leader/read", a.Host, a.Port)
		} else if a.IsLocal() {
			reqURL = fmt.Sprintf("http://localhost:%d/raft/leader/read", a.Port)
		}
	}

	// make the request
	reqBody := fmt.Sprintf(
		`{ "UUID": "%s", "Fn": "%s", "Body": {} }`,
		uuid.NewString(), a.BackupURL)

	if res, err := http.Post(reqURL, "application/json", reqBody); err != nil {
		return fmt.Errorf("could not request %s: %v", reqURL, err)
	} else {
		core.Log.Warnf("finished %s: %s", a.KubeName, res)
	}
	return nil
}

func (a *Service) Reset() error {
	// TODO handle statefulsets and pods

	// make the HTTP request to the reset endpoint
	reqURL := fmt.Sprintf("http://%s:%d/v1/Reset/App", a.Host, a.Port)
	core.Log.Warnf("trying: %s", reqURL)
	reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Reset/App", "Body": {} }`, uuid.NewString())
	if res, err := http.Post(reqURL, "application/json", reqBod); err != nil {
		core.Log.Fatalf("%s: %s: %v", reqURL, res, err)
	} else {
		core.Log.Warnf("%s: %s", reqURL, res)
	}

	return nil
}

func (a *Service) Stage(
	kubeClient *kube.KubeClient,
	srcArchiveFile *ArchiveFile) error {

	if a.IsStatefulSet() {
		// TODO handle this
		if kubeClient == nil {
			return fmt.Errorf("must have kubeClient")
		}

		return fmt.Errorf("Archive.Stage not allowed for statefulset archives")
	} else if a.IsPod() {
		// TODO handle this
		if kubeClient == nil {
			return fmt.Errorf("must have kubeClient")
		}

		return fmt.Errorf("Archive.Stage not allowed for pod archives")
	} else if a.IsLocal() {
		// clear the content of the restore folder
		if err := os.RemoveAll(a.RestorePath); err != nil {
			core.Log.Fatalf("cound not clear the content of the restore folder: %v", err)
		}

		// recreate it
		if err := os.MkdirAll(a.RestorePath, 0774); err != nil {
			core.Log.Fatalf("cound not create the restore folder: %v", err)
		}

		// can only do local to local
		if srcArchiveFile.Archive.Scheme != "local" {
			return fmt.Errorf("can only restore to local from local. srcArchive is %v", srcArchiveFile.Archive)
		}

		// make a symlink
		srcArchiveFilePath := srcArchiveFile.Archive.Path + "/" + srcArchiveFile.Name
		dstArchiveFilePath := a.RestorePath + "/" + srcArchiveFile.Name
		err := os.Symlink(srcArchiveFilePath, dstArchiveFilePath)
		if err != nil {
			core.Log.Fatalf("cound not create symlink: src %s to %s: %v", srcArchiveFilePath, dstArchiveFilePath, err)
		}
	}
	return nil
}

func (a *Service) Restore() error {
	core.Log.Warnf("restoring %s", a.Name)
	reqURL := fmt.Sprintf("http://%s:%d%s", a.Host, a.Port, a.RestoreURL)
	core.Log.Warnf("trying: %s", reqURL)
	reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Restore", "Body": {} }`, uuid.NewString())
	if res, err := http.Post(reqURL, "application/json", reqBod); err != nil {
		core.Log.Fatalf("%s: %s: %v", reqURL, res, err)
	} else {
		core.Log.Warnf("%s: %s", reqURL, res)
	}

	return nil
}
func (a *Service) RAFTReset() error {
	reqURL := fmt.Sprintf("http://%s:%d/v1/Reset/Raft", a.Host, a.Port)
	reqBod := fmt.Sprintf(`{ "UUID": "%s", "Fn": "/v1/Reset/Raft", "Body": {} }`, uuid.NewString())
	if res, err := http.Post(reqURL, "application/json", reqBod); err != nil {
		core.Log.Fatalf("%s: %s: %v", reqURL, res, err)
	} else {
		core.Log.Warnf("%s: %s", reqURL, res)
	}
	return nil
}
