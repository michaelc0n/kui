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

	//namespace := "kube-system"

	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config
	//config, _ := clientcmd.BuildConfigFromFlags("", "<path-to-kubeconfig>")
	// creates the clientset
	//clientset, _ := kubernetes.NewForConfig(config)
	// access the API to list pods
	pods, _ := clientset.CoreV1().Pods("").List(context.TODO(), v1.ListOptions{})
	fmt.Print(pods)
	//pod_msg, _ := fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	pod_msg := fmt.Sprintf("There are %d pods in the cluster\n", len(pods.Items))
	// items, err := GetDeployments(clientset, ctx, namespace)
	// if err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	for _, item := range items {
	// 		fmt.Printf("%+v\n", item)
	// 	}
	// }
	hello := widget.NewLabel("KUI!")
	w.SetContent(container.NewVBox(
		hello,
		widget.NewButton("run", func() {
			hello.SetText(pod_msg)
		}),
	))
	w.ShowAndRun()
}

// func GetDeployments(clientset *kubernetes.Clientset, ctx context.Context,
// 	namespace string) ([]v1.Deployment, error) {

// 	list, err := clientset.AppsV1().Deployments(namespace).
// 		List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return list.Items, nil
// }
