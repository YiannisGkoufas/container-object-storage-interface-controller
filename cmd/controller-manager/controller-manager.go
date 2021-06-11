package main

import (
	"context"
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	bucketcontroller "sigs.k8s.io/container-object-storage-interface-api/controller"
	"sigs.k8s.io/container-object-storage-interface-controller/pkg/bucketaccessrequest"
	"sigs.k8s.io/container-object-storage-interface-controller/pkg/bucketrequest"
	"syscall"

	"k8s.io/klog/v2"
)

var cmd = &cobra.Command{
	Use:           "controller-manager",
	Short:         "central controller for managing bucket* and bucketAccess* API objects",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(c *cobra.Command, args []string) error {
		return run(c.Context(), args)
	},
	DisableFlagsInUseLine: true,
}

var kubeConfig string
var verbosity int

func init() {
	viper.AutomaticEnv()

	flag.Set("alsologtostderr", "true")
	kflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kflags)

	cmd.PersistentFlags().AddGoFlagSet(kflags)
	cmd.PersistentFlags().StringVarP(&kubeConfig, "kubeconfig", "", kubeConfig, "path to kubeconfig file")

	//flag.CommandLine.Parse([]string{})
	viper.BindPFlags(cmd.PersistentFlags())
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Just in case

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	if err := cmd.ExecuteContext(ctx); err != nil {
		klog.Error(err)
	}
}

func run(ctx context.Context, args []string) error {
	ctrl, err := bucketcontroller.NewDefaultObjectStorageController("cosi-controller-manager", "leader-lock", 40)
	if err != nil {
		return err
	}
	ctrl.AddBucketRequestListener(bucketrequest.NewBucketRequestListener())
	ctrl.AddBucketAccessRequestListener(bucketaccessrequest.NewListener())
	return ctrl.Run(ctx)
}
