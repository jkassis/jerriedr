package kube

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/alessio/shellescape"
	"github.com/jkassis/jerrie/core"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

// KubeClient is a kube client
type KubeClient struct {
	Clientset      *kubernetes.Clientset
	Config         *rest.Config
	KubeConfigPath string
	MasterURL      string
	Rand           *rand.Rand
}

// NewKubeClient returns a new, init'd kube client
func NewKubeClient(masterURL, kubeConfigPath string) (*KubeClient, error) {
	kubeClient := &KubeClient{
		KubeConfigPath: kubeConfigPath,
		MasterURL:      masterURL,
	}
	err := kubeClient.Init()
	if err != nil {
		return nil, err
	}
	return kubeClient, nil
}

// Init starts the kubernetes client
func (c *KubeClient) Init() error {
	config, err := clientcmd.BuildConfigFromFlags(c.MasterURL, c.KubeConfigPath)
	if err != nil {
		return fmt.Errorf("could not load kube config from %s: %v", c.KubeConfigPath, err)
	}

	c.Config = config
	gv := v1.SchemeGroupVersion
	c.Config.GroupVersion = &gv
	// config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	// create the clientset
	clientset, err := kubernetes.NewForConfig(c.Config)
	if err != nil {
		return err
	}

	c.Clientset = clientset

	c.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return nil
}

// DeploymentGetByNamespace returns deployments
func (c *KubeClient) DeploymentGetByNamespace(namespace string, pattern string) ([]string, error) {
	matchingDeploymentNames := []string{}
	deploymentsList, err := c.Clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, deployment := range deploymentsList.Items {
		matched, err := regexp.MatchString(pattern, deployment.GetName())
		if err != nil {
			return nil, err
		}
		if matched {
			matchingDeploymentNames = append(matchingDeploymentNames, deployment.Name)
		}
	}

	sort.Slice(matchingDeploymentNames, func(i, j int) bool {
		return matchingDeploymentNames[i] > matchingDeploymentNames[j]
	})
	return matchingDeploymentNames, nil
}

// StatefulSetGetByNamespace returns deployments
func (c *KubeClient) StatefulSetGetByNamespace(namespace string, pattern string) ([]string, error) {
	matchingDeploymentNames := []string{}
	statefulSetList, err := c.Clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, statefulSet := range statefulSetList.Items {
		matched, err := regexp.MatchString(pattern, statefulSet.GetName())
		if err != nil {
			return nil, err
		}
		if matched {
			matchingDeploymentNames = append(matchingDeploymentNames, statefulSet.Name)
		}
	}

	sort.Slice(matchingDeploymentNames, func(i, j int) bool {
		return matchingDeploymentNames[i] > matchingDeploymentNames[j]
	})
	return matchingDeploymentNames, nil
}

// StatefulSetGetByName returns deployments
func (c *KubeClient) StatefulSetGetByName(namespace string, name string) (*v1.StatefulSet, error) {
	statefulSet, err := c.Clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		return nil, fmt.Errorf("pod %s in namespace %s not found", statefulSet, namespace)
	} else if statusError, isStatus := err.(*k8sErrors.StatusError); isStatus {
		return nil, fmt.Errorf("error getting pod %s in namespace %s: %v", name, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, err
	}
	return statefulSet, nil
}

// PodGetByName returns a pod
func (c *KubeClient) PodGetByName(namespace, name string) (*corev1.Pod, error) {
	pod, err := c.Clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		return nil, fmt.Errorf("pod %s in namespace %s not found", pod, namespace)
	} else if statusError, isStatus := err.(*k8sErrors.StatusError); isStatus {
		return nil, fmt.Errorf("error getting pod %s in namespace %s: %v", name, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		return nil, err
	}
	return pod, nil
}

// PodGetByNamespace returns all pods in a cluster
func (c *KubeClient) PodGetByNamespace(namespace string) (*corev1.PodList, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// PodGetByDeploymentName returns all pods for a deployment
func (c *KubeClient) PodGetByDeploymentName(namespace, deploymentName string) (*corev1.PodList, error) {
	pods, err := c.Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "deployment=" + deploymentName,
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// PodGetRandomByDeploymentName return a random pod in the namespace / deployment
func (c *KubeClient) PodGetRandomByDeploymentName(namespace, deploymentName string) (*corev1.Pod, error) {
	// Get a deployment to operate on... we might be wrong
	deployments, err := c.DeploymentGetByNamespace(namespace, deploymentName)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Found these remote deployments: %s\n", strings.Join(deployments, " "))
	firstDeployment := deployments[0]

	// PodList
	logrus.Info("Getting pods for " + namespace + "/" + firstDeployment + "\n")
	podList, err := c.PodGetByDeploymentName(namespace, firstDeployment)
	if err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		return nil, errors.New("found no pods")
	}
	logrus.Infof("Got %d pods for %s/%s\n", len(podList.Items), namespace, deploymentName)

	pod := podList.Items[c.Rand.Intn(len(podList.Items))]
	return &pod, nil
}

// Exec executes a command asynchronously on a given pod
// stdin is piped to the remote shell if provided or nothing if nil
// returns the output from stdout and stderr
func (c *KubeClient) Exec(
	pod *corev1.Pod,
	containerName string,
	command []string,
	stdin io.Reader) (io.Reader, io.Reader, error) {
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	request := c.Clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		Param("container", containerName).
		VersionedParams(&corev1.PodExecOptions{
			Command: command,
			Stdin:   stdin != nil,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(c.Config, "POST", request.URL())
	if err != nil {
		return nil, nil, err
	}

	go func() {
		_ = exec.Stream(remotecommand.StreamOptions{
			Stdin:  stdin,
			Stdout: stdoutWriter,
			Stderr: stderrWriter,
			Tty:    false,
		})
		stdoutWriter.Close()
		stderrWriter.Close()
	}()

	// return stdout, stderr
	return stdoutReader, stderrReader, nil
}

// ExecSync executes a command synchronously on a given pod
// stdin is piped to the remote shell if provided or nothing if nil
// returns the output from stdout or err as a string and err
func (c *KubeClient) ExecSync(pod *corev1.Pod, containerName string, command []string, stdin io.Reader) (string, error) {
	stdoutReader, stderrReader, err := c.Exec(pod, containerName, command, stdin)
	if err != nil {
		return "", err
	}

	results, err := ReadAll(stdoutReader, stderrReader)
	if err != nil {
		return "", err
	}
	stdout := results[0]
	stderr := results[1]
	if len(stderr) > 0 {
		return string(stdout), errors.New(string(stderr))
	}

	return string(stdout), nil
}

// ExecSyncAndLog executes a command synchronously on a given pod
// stdin is piped to the remote shell if provided or nothing if nil
// streams the remote stdout / stderr to local logs and returns error if any are discovered
func (c *KubeClient) ExecSyncAndLog(pod *corev1.Pod, containerName string, command []string, stdin io.Reader) error {
	logrus.Info("Running... ", strings.Join(command, " "))
	stdoutReader, stderrReader, err := c.Exec(pod, "php-fpm", command, nil)
	if err != nil {
		return err
	}
	StreamAllToLog(pod.Name+" : ", stdoutReader, stderrReader)
	return nil
}

// ExecSyncAndLogOnRandomPod executes a command synchronously on a random pod
// stdin is piped to the remote shell if provided or nothing if nil
// returns the output from stdout and stderr
func (c *KubeClient) ExecSyncAndLogOnRandomPod(namespace, deployment, container string, command []string, stdin io.Reader) error {
	// get a random pod
	pod, err := c.PodGetRandomByDeploymentName(namespace, deployment)
	if err != nil {
		return err
	}

	return c.ExecSyncAndLog(pod, container, command, stdin)
}

// FileSpec holds a location for a remote or local file
type FileSpec struct {
	PodNamespace string
	PodName      string
	Path         string
}

func (f *FileSpec) String() string {
	return fmt.Sprintf("ns: %s | podName: %s | path: %s", f.PodNamespace, f.PodName, f.Path)
}

// DirLs returns a list of files on the pod in the given dir
func (c *KubeClient) DirLs(src *FileSpec, pod *corev1.Pod, containerName string) ([]string, error) {
	srcFile := shellescape.Quote(src.Path)
	cmdArr := []string{"/bin/sh", "-c", "ls " + srcFile}
	stdout, err := c.ExecSync(pod, containerName, cmdArr, nil)
	if err != nil {
		return nil, err
	}
	return strings.Split(stdout, "\n"), nil
}

// DirMake copies a file from local dir to remote
func (c *KubeClient) DirMake(src, dest *FileSpec, pod *corev1.Pod, containerName string) (io.Reader, io.Reader, error) {
	destFile := shellescape.Quote(dest.Path)
	cmdArr := []string{"/bin/sh", "-c", "mkdir -p " + destFile}
	logrus.Info("making directory in pod : '" + pod.Name + "'")
	return c.Exec(pod, containerName, cmdArr, nil)
}

// FileWriterGet gets a writer to a file on a pod
// Use this to open a file and stream to the writer
// f, err := os.Open(src.File)
//
//	if err != nil {
//		return err
//	}
//
// defer f.Close()
// io.Copy(dstWriter, srcFile)
func (c *KubeClient) FileWrite(src io.Reader, dst *FileSpec, pod *corev1.Pod, containerName string) (err error) {
	dstFile := shellescape.Quote(dst.Path)
	// cmdArr := []string{"/bin/sh", "-c", "mkdir -p " + filepath.Dir(dstFile) + " ; cat > " + dstFile}
	cmdArr := []string{"env", "cat", ">", dstFile}

	stdoutReader, stderrReader, err := c.Exec(pod, containerName, cmdArr, src)
	if err != nil {
		return err
	}

	eg := &errgroup.Group{}

	// stream to dst
	eg.Go(func() error {
		stdout := bytes.NewBuffer(nil)
		_, err = io.Copy(stdout, stdoutReader)
		if err != nil {
			return err
		}
		if stdout.Len() > 0 {
			return fmt.Errorf("FileWrite: got data from stdout: %v", stdout)
		}

		return err
	})

	// stream to err
	eg.Go(func() error {
		stderr := bytes.NewBuffer(nil)
		_, err = io.Copy(stderr, stderrReader)
		if err != nil {
			return err
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("FileWrite: got data from stderr: %v", stderr)
		}
		return nil
	})

	err = eg.Wait()
	return err
}

// FileReaderGet gets a reader to a file on a pod
// Use this to write to a local file...
// // open the dstFile
// f, err := os.OpenFile(dst.File, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
//
//	if err != nil {
//		return err
//	}
//
//	defer func() {
//		f.Sync()
//		f.Close()
//	}()
//
// io.Copy(dstFile, reader)
func (c *KubeClient) FileRead(src *FileSpec, dst io.WriteCloser, pod *corev1.Pod, containerName string) (err error) {
	srcFile := shellescape.Quote(src.Path)
	cmdArr := []string{"env", "cat", srcFile}

	defer dst.Close()

	stdoutReader, stderrReader, err := c.Exec(pod, containerName, cmdArr, nil)
	if err != nil {
		return err
	}

	errs := &errgroup.Group{}

	// stream to dst
	errs.Go(func() error {
		_, err = io.Copy(dst, stdoutReader)
		return err
	})

	// stream to err
	errs.Go(func() error {
		stderr := bytes.NewBuffer(nil)
		_, err = io.Copy(stderr, stderrReader)
		if err != nil {
			return err
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("CopyFromPod: got data from stderr: %v", stderr)
		}
		return nil
	})

	err = errs.Wait()
	return err
}

// FileRm removes a file from a remote
func (c *KubeClient) FileRm(dst *FileSpec, pod *corev1.Pod, containerName string) (io.Reader, io.Reader, error) {
	cmdArr := []string{"/bin/sh", "-c", "rm -rf " + dst.Path}
	fmt.Println(strings.Join(cmdArr, " "))
	return c.Exec(pod, containerName, cmdArr, nil)
}

// tarMake is not used, but reserved for the future
func tarMake(srcPath, destPath string, writer io.Writer) error {
	// TODO: use compression here?
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	srcPath = path.Clean(srcPath)
	destPath = path.Clean(destPath)
	return tarMakeRecursive(path.Dir(srcPath), path.Base(srcPath), path.Dir(destPath), path.Base(destPath), tarWriter)
}

// tarMakeRecursive is not used, but reserved for the future
func tarMakeRecursive(srcBase, srcFile, destBase, destFile string, tw *tar.Writer) error {
	srcPath := path.Join(srcBase, srcFile)
	matchedPaths, err := filepath.Glob(srcPath)
	if err != nil {
		return err
	}
	for _, fpath := range matchedPaths {
		stat, err := os.Lstat(fpath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			files, err := os.ReadDir(fpath)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				//case empty directory
				hdr, _ := tar.FileInfoHeader(stat, fpath)
				hdr.Name = destFile
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			for _, f := range files {
				if err := tarMakeRecursive(srcBase, path.Join(srcFile, f.Name()), destBase, path.Join(destFile, f.Name()), tw); err != nil {
					return err
				}
			}
			return nil
		} else if stat.Mode()&os.ModeSymlink != 0 {
			//case soft link
			hdr, _ := tar.FileInfoHeader(stat, fpath)
			target, err := os.Readlink(fpath)
			if err != nil {
				return err
			}

			hdr.Linkname = target
			hdr.Name = destFile
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		} else {
			//case regular file or other file type like pipe
			hdr, err := tar.FileInfoHeader(stat, fpath)
			if err != nil {
				return err
			}
			hdr.Name = destFile

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			f, err := os.Open(fpath)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
			return f.Close()
		}
	}
	return nil
}

type PortForwardRequest struct {
	LocalPort    int // LocalPort is the local port that will be selected to expose the PodPort
	PodName      string
	PodNamespace string
	PodPort      int // PodPort is the target port for the pod
}

// It is to forward port, and return the forwarder.
func (c *KubeClient) PortForward(req *PortForwardRequest) (*portforward.ForwardedPort, error) {
	// get the pod
	pod, err := c.PodGetByName(req.PodNamespace, req.PodName)
	if err != nil {
		return nil, err
	}

	// check the status
	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("unable to forward port because pod %s is not running. Current status=%v", req.PodName, pod.Status.Phase)
	}

	// make the dialer to establish port forwarding
	kubeAPIUrl, err := url.Parse(c.Config.Host)
	if err != nil {
		return nil, err
	}
	kubeAPIUrl.Path = path.Join(
		"api", "v1",
		"namespaces", req.PodNamespace,
		"pods", req.PodName,
		"portforward",
	)
	transport, upgrader, err := spdy.RoundTripperFor(c.Config)
	if err != nil {
		return nil, err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, kubeAPIUrl)

	// make the stop and ready channels
	stopCh := make(chan struct{}, 1)
	readyCh := make(chan struct{})

	// make pipes
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()

	fw, err := portforward.New(dialer,
		[]string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)},
		stopCh,
		readyCh,
		outW,
		errW)
	if err != nil {
		return nil, err
	}
	// fw.GetPorts()

	// send all output to logs
	go StreamAllToLog(fmt.Sprintf("%s/%s: ", req.PodNamespace, req.PodName), outR, errR)

	// stop forwarding if the os closes
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	go func() {
		<-signals
		if stopCh != nil {
			close(stopCh)
		}
	}()

	// forward
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			core.Log.Fatalf("could not forward port: %v", err)
		}
	}()

	// wait on the ready channel
	<-readyCh

	// get the forwarded ports
	ports, err := fw.GetPorts()
	if err != nil {
		return nil, err
	}
	port := ports[0]
	return &port, nil
}
