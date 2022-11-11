package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	a := app.New()
	w := a.NewWindow("KUI")

	//ctx := context.Background()
	config := ctrl.GetConfigOrDie()
	clientset := kubernetes.NewForConfigOrDie(config)

	// get pods in all the namespaces by omitting namespace
	// Or specify namespace to get pods in particular namespace
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	numberOfPods := fmt.Sprintf("There are %d pods in the cluster\n", len(pods.Items))

	namespace := namespace(*clientset)

	kui := widget.NewLabel("KUI")
	w.SetContent(container.NewVBox(
		kui,
		widget.NewButton("number of pods?", func() {
			kui.SetText(numberOfPods)
		}),

		widget.NewButton("kube-system namespace present?", func() {
			kui.SetText(namespace)
		}),
	))
	w.ShowAndRun()
}

func namespace(c kubernetes.Clientset) string {
	nsList, err := c.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	//TODO create err function - replace all != nill
	if err != nil {
		panic(err.Error())
	}

	for _, n := range nsList.Items {
		if n.Name == "kube-system" {
			fmt.Println(n)
			return n.Name
		}
	}

	return ""

}
