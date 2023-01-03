package main

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// setup k8s clientset
	clientset := getClientSet()

	// get a list of all pods
	podData := getPodData(*clientset)

	// get current cluster context
	currentContext := getCurrentContext()

	// create a new app
	app := app.New()
	// create a new window with app title
	win := app.NewWindow("KUI")
	// resize fyne app window
	win.Resize(fyne.NewSize(1200, 700)) // first width, then height

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

	// top window label
	topLabel := canvas.NewText(("Cluster Context: " + currentContext), color.NRGBA{R: 57, G: 112, B: 228, A: 255})
	topLabel.TextStyle = fyne.TextStyle{Monospace: true}
	topContent := container.New(layout.NewCenterLayout(), topLabel)

	// right side of split
	rightWinContent := container.NewMax()
	title := widget.NewLabel("Select application (pod)...")
	title.TextStyle = fyne.TextStyle{Bold: true, Italic: true, Monospace: true}

	// pod status
	podStatus := widget.NewLabel("")
	podStatus.TextStyle = fyne.TextStyle{Monospace: true}
	podStatus.Wrapping = fyne.TextWrapWord

	// get pod labels, annotations, events for tabs
	podLabelsLabel, podLabels, podLabelsScroll := getPodTabData("Labels")
	podAnnotationsLabel, podAnnotations, podAnnotationsScroll := getPodTabData("Annotations")
	podEventsLabel, podEvents, podEventsScroll := getPodTabData("Events")

	// setup pod tabs
	podTabs := container.NewAppTabs(
		container.NewTabItem(podLabelsLabel.Text, podLabelsScroll),
		container.NewTabItem(podAnnotationsLabel.Text, podAnnotationsScroll),
		container.NewTabItem(podEventsLabel.Text, podEventsScroll),
	)

	// setup pod log tabs
	podLogsLabel := widget.NewLabel("Select container log... ")
	podLogsLabel.TextStyle = fyne.TextStyle{Monospace: true}
	defaultTabItem := container.NewTabItem("Logs", podLogsLabel)
	podLogTabs := container.NewAppTabs(defaultTabItem)

	// update pod list data
	refresh := widget.NewButton("Refresh", func() {
		podData = getPodData(*clientset)
		list.UnselectAll()
		data.Reload()
	})

	list.OnSelected = func(id widget.ListItemID) {
		for index := range podData {
			if index == id {
				selectedPod, err := data.GetValue(id)
				if err != nil {
					panic(err.Error())
				}
				title.Text = "Application (Pod): " + selectedPod
				title.Refresh()

				newPodStatus, newPodAge, newPodNamespace, newPodLabels, newPodAnnotations, newNodeName, newContainers := getPodDetail(*clientset, id, selectedPod)

				podStatus.Text = "Status: " + newPodStatus + "\n" +
					"Age: " + newPodAge + "\n" +
					"Namespace: " + newPodNamespace + "\n" +
					"Node: " + newNodeName
				podStatus.Refresh()

				podLabels.Text = newPodLabels
				podLabels.Refresh()

				podAnnotations.Text = newPodAnnotations
				podAnnotations.Refresh()

				// get pod events
				newPodEvents := getPodEvents(*clientset, selectedPod)
				strNewPodEvents := strings.Join(newPodEvents, "\n")
				podEvents.Text = strNewPodEvents
				podEvents.Refresh()

				fmt.Print(newContainers)

				for _, tabContainerName := range newContainers {
					podLogStream := getPodLogs(*clientset, newPodNamespace, selectedPod, tabContainerName)
					podLog := widget.NewLabel(podLogStream)
					podLog.TextStyle = fyne.TextStyle{Monospace: true}
					podLog.Wrapping = fyne.TextWrapBreak
					podLogScroll := container.NewScroll(podLog)
					podLogScroll.SetMinSize(fyne.Size{Height: 200})
					podLogTabs.Append(container.NewTabItem(tabContainerName, podLogScroll))
					podLog.Refresh()
				}
				podLogTabs.Refresh()
			}
		}
	}

	list.OnUnselected = func(id widget.ListItemID) {
		for _, tabItem := range podLogTabs.Items {
			if tabItem != defaultTabItem {
				podLogTabs.Remove(tabItem)
			}
		}
	}

	rightContainer := container.NewBorder(
		container.NewVBox(title, podStatus, podTabs, podLogTabs),
		nil, nil, nil, rightWinContent)

	listTitle := widget.NewLabel("Application (Pod)")
	listTitle.Alignment = fyne.TextAlignCenter
	listTitle.TextStyle = fyne.TextStyle{Monospace: true}

	// search application name (input list field)
	input := widget.NewEntry()
	input.SetPlaceHolder("Search application...")
	// submit to func input string (pod name), return new pod list
	input.OnSubmitted = func(s string) {
		inputText := input.Text
		var inputTextList []string
		if inputText == "" {
			podData = getPodData(*clientset)
			data.Reload()
			list.UnselectAll()
		} else {
			for _, pod := range podData {
				if strings.Contains(pod, inputText) {
					inputTextList = append(inputTextList, pod)
				}
			}
			podData = inputTextList
			data.Reload()
			list.UnselectAll()
		}
	}

	listContainer := container.NewBorder(container.NewVBox(listTitle, input), nil, nil, nil, list)

	// podData(list) left side, podData detail right side
	split := container.NewHSplit(listContainer, rightContainer)
	split.Offset = 0.3

	// check current cluster context to update top window label
	go func() {
		for range time.Tick(time.Second * 5) {
			currentContext = getCurrentContext()
			if strings.Contains(topLabel.Text, currentContext) {
				continue
			} else {
				topLabel.Text = ("Cluster Context: " + currentContext)
				topLabel.Refresh()
			}
		}
	}()

	win.SetContent(container.NewBorder(topContent, refresh, nil, nil, split))
	win.ShowAndRun()
}

func getPodTabData(widgetLabelName string) (widgetNameLabel *widget.Label, widgetName *widget.Label, widgetNameScroll *container.Scroll) {
	widgetNameLabel = widget.NewLabel(widgetLabelName)
	widgetNameLabel.TextStyle = fyne.TextStyle{Monospace: true}
	widgetName = widget.NewLabel("")
	widgetName.TextStyle = fyne.TextStyle{Monospace: true}
	widgetName.Wrapping = fyne.TextWrapBreak
	widgetNameScroll = container.NewScroll(widgetName)
	widgetNameScroll.SetMinSize(fyne.Size{Height: 100})
	return widgetNameLabel, widgetName, widgetNameScroll
}

// get pod names to populate initial list
func getPodData(c kubernetes.Clientset) (podData []string) {
	pods, err := c.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	for _, pod := range pods.Items {
		podData = append(podData, pod.Name)
	}
	return podData
}

func getPodDetail(c kubernetes.Clientset, listItemID int, selectedPod string) (string, string, string, string, string, string, []string) {
	pods, err := c.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	for _, pod := range pods.Items {
		if pod.Name == selectedPod {
			var containers []string
			for _, container := range pod.Spec.Containers {
				containers = append(containers, container.Name)
			}
			podCreationTime := pod.GetCreationTimestamp()
			age := time.Since(podCreationTime.Time).Round(time.Second)

			return string(pod.Status.Phase), age.String(), string(pod.Namespace), convertMapToString(pod.Labels), convertMapToString(pod.Annotations),
				pod.Spec.NodeName, containers
		}
	}
	return "", "", "", "", "", "", []string{}
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

//TODO parse cluster context name to drop unnecessary text
func getCurrentContext() string {
	// get current context
	clientConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		}).RawConfig()
	return clientConfig.CurrentContext
}

// used by labels, annotations, ...
func convertMapToString(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s=\"%s\"\n", key, value)
	}
	return b.String()
}

func getPodEvents(c kubernetes.Clientset, selectedPod string) (podEvents []string) {
	events, _ := c.CoreV1().Events("").List(context.TODO(), v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%s", selectedPod), TypeMeta: v1.TypeMeta{Kind: "Pod"}})
	for _, item := range events.Items {
		podEvents = append(podEvents, "~> "+item.EventTime.Time.Format("2006-01-02 15:04:05")+", "+item.Message)
	}
	return podEvents
}

func getPodLogs(c kubernetes.Clientset, podNamespace string, selectedPod string, containerName string) (podLog string) {
	podLogReq := c.CoreV1().Pods(podNamespace).GetLogs(selectedPod, &corev1.PodLogOptions{Container: containerName})
	podStream, err := podLogReq.Stream(context.TODO())
	if err != nil {
		return fmt.Sprintf("error opening pod log stream, %v", err)
	}
	defer podStream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podStream)
	if err != nil {
		return "error copying pod log stream to buf"
	}
	podLog = buf.String()

	return podLog
}

//TODO catch panic when cluster context not available:
// panic: Get "https://1.2.3.4:443/api/v1/pods": dial tcp 1.2.3.4:443: i/o timeout

//TODO test if kubeConfig not accessible/ not set
//TODO test if clusterContext not set / empty
//TODO add copy capability
//TODO: clear podTab data on refresh, similar to podLogTab data on refresh
