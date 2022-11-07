package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	a := app.New()
	w := a.NewWindow("KUI")

	hello := widget.NewLabel("KUI!")
	w.SetContent(container.NewVBox(
		hello,
		widget.NewButton("run", func() {
			hello.SetText("nothing to see here :)")
		}),
	))

	ctx := context.Background()
	config := ctrl.GetConfigOrDie()
	clientset := kubernetes.NewForConfigOrDie(config)

	namespace := "kube-system"
	items, err := GetDeployments(clientset, ctx, namespace)
	if err != nil {
		fmt.Println(err)
	} else {
		for _, item := range items {
			fmt.Printf("%+v\n", item)
		}
	}

	w.ShowAndRun()
}

func GetDeployments(clientset *kubernetes.Clientset, ctx context.Context,
	namespace string) ([]v1.Deployment, error) {

	list, err := clientset.AppsV1().Deployments(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
