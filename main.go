package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// setup k8s clientset
	clientset := getClientSet()

	// get a list of all pods
	podData := getPodData(*clientset)
	//podNames := getPodNames(podData)
	// create a new app

	// get current cluster context
	currentContext := getCurrentContext()

	app := app.New()

	// create a new window
	win := app.NewWindow("KUI") // use any title for app

	// resize fyne app window
	win.Resize(fyne.NewSize(900, 700)) // first width, then height

	// list binding
	data := binding.BindStringList(
		&podData,
	)
	list := widget.NewListWithData(data,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})

	//TODO - create func to get podStatus outside of above func or from global scope
	// will need to pass in "id widget.ListItemID"

	topLabel := widget.NewLabel("Cluster Context: " + currentContext)
	topLabel.TextStyle = fyne.TextStyle{Monospace: true}
	topLabel.Alignment = fyne.TextAlignCenter
	topLabel.Wrapping = fyne.TextWrapWord

	// right side of split
	rightWinContent := container.NewMax()
	title := widget.NewLabel("Application Name: ")
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	//title.Wrapping = fyne.TextWrapWord

	// get pod status on selected
	podStatus := widget.NewLabel("Application Status: ")
	podStatus.TextStyle = fyne.TextStyle{Monospace: true}
	podStatus.Wrapping = fyne.TextWrapWord

	list.OnSelected = func(id widget.ListItemID) {
		for i, podName := range podData {
			if i == id {
				title.Text = podName
				title.Text = ("Application Name: " + podName)
				podStatus.Text = "Application Status: " + getPodStatus(*clientset, id, data, podData)
				podStatus.Refresh()
				title.Refresh()
			}
		}
		podStatus.Refresh()
	}

	// reload pod list data when unselected
	list.OnUnselected = func(id widget.ListItemID) {
		podData = reloadPodData(*clientset, data)
	}

	// update pod list data
	refresh := widget.NewButton("Refresh", func() {
		podData = reloadPodData(*clientset, data)

	})

	//TODO: update right side with pod detail// initially pod.Status
	rightContainer := container.NewBorder(
		container.NewVBox(title, podStatus), nil, nil, nil, rightWinContent)

	// podData(list) left side, podData detail right side
	split := container.NewHSplit(list, rightContainer)
	split.Offset = 0.4

	go func() {
		for range time.Tick(time.Second * 5) {
			currentContext = getCurrentContext()
			//topLabel = widget.NewLabel(currentContext)
			fmt.Println(topLabel.Text)
			if strings.Contains(topLabel.Text, currentContext) {
				continue
			} else {
				topLabel.SetText("Cluster Context: " + currentContext)
			}
		}
	}()

	win.SetContent(container.NewBorder(topLabel, refresh, nil, nil, split))
	win.ShowAndRun()
}

func getPodStatus(c kubernetes.Clientset, listItemID int, data binding.ExternalStringList, podData []string) string {
	// get pods in all the namespaces by omitting ("") namespace
	// Or specify namespace to get pods in particular namespace
	pods, err := c.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, pod := range pods.Items {
		podName, err := data.GetValue(listItemID)
		if err != nil {
			panic(err.Error())
		}
		if pod.Name == podName {
			return string(pod.Status.Phase)
		}
	}
	return ""
}

// get pod names to populate initial list
func getPodData(c kubernetes.Clientset) (podData []string) {
	// get pods in all the namespaces by omitting ("") namespace
	// Or specify namespace to get pods in particular namespace
	pods, err := c.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, pod := range pods.Items {
		podData = append(podData, pod.Name)
	}
	return podData
}

func reloadPodData(c kubernetes.Clientset, data binding.ExternalStringList) []string {
	podData := getPodData(c)
	data.Reload()
	return podData
}

// moving logging to diff file and only log to stdout not file
var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

func init() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

//TODO parse cluster context name to drop everything after "anthos"
func getCurrentContext() string {
	// get current context
	clientConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		}).RawConfig()
	return clientConfig.CurrentContext
}

//TODO catch panic when cluster context not available:
// panic: Get "https://1.2.3.4:443/api/v1/pods": dial tcp 1.2.3.4:443: i/o timeout

//TODO test if kubeConfig not accessible/ not set
//TODO test if clusterContext not set / empty
