package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"strings"

	"github.com/kubicorn/kubicorn/pkg/logger"

	"github.com/weaveworks/eksctl/pkg/eks"
	"github.com/weaveworks/eksctl/pkg/eks/api"
	"github.com/weaveworks/eksctl/pkg/utils/kubeconfig"
)

func deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resource(s)",
		Run: func(c *cobra.Command, _ []string) {
			c.Help()
		},
	}

	cmd.AddCommand(deleteClusterCmd())

	return cmd
}

func deleteClusterCmd() *cobra.Command {
	cfg := &api.ClusterConfig{}

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Delete a cluster",
		Run: func(_ *cobra.Command, args []string) {
			if err := doDeleteCluster(cfg, getNameArg(args)); err != nil {
				logger.Critical("%s\n", err.Error())
				os.Exit(1)
			}
		},
	}

	fs := cmd.Flags()

	fs.StringVarP(&cfg.ClusterName, "name", "n", "", "EKS cluster name (required)")

	fs.StringVarP(&cfg.Region, "region", "r", api.DEFAULT_EKS_REGION, "AWS region")
	fs.StringVarP(&cfg.Profile, "profile", "p", "", "AWS credentials profile to use (overrides the AWS_PROFILE environment variable)")

	fs.DurationVar(&cfg.WaitTimeout, "timeout", api.DefaultWaitTimeout, "max wait time in any polling operations")

	return cmd
}

// TODO: comment
func deleteClusterAndResources(ctl *eks.ClusterProvider) {
	// TODO: should I think of a better way to handle this?
	var deletedResources []string
	handleIfError := func(err error, name string) bool {
		if err != nil {
			logger.Debug("continue despite error: %v", err)
			return true
		}
		logger.Debug("deleted %q", name)
		deletedResources = append(deletedResources, name)
		return false
	}

	stackManager := ctl.NewStackManager()

	// TODO: is the handleIfError(...) thing super ugly? Would it be better to handle this with the
	// more traditional 'if err := <method>; if err != nil { ... }' thing?
	handleIfError(stackManager.WaitDeleteNodeGroup(), "node group")
	if handleIfError(stackManager.DeleteCluster(), "cluster") {
		if handleIfError(ctl.DeprecatedDeleteControlPlane(),"control plane") {
			handleIfError(stackManager.DeprecatedDeleteStackControlPlane(), "stack control plane")
		}
	}
	handleIfError(stackManager.DeprecatedDeleteStackServiceRole(), "node group")
	handleIfError(stackManager.DeprecatedDeleteStackVPC(), "stack VPC")
	handleIfError(stackManager.DeprecatedDeleteStackDefaultNodeGroup(), "default node group")

	if len(deletedResources) == 0 {
		logger.Warning("No EKS cluster resource were found for %q", ctl.Spec.ClusterName)
	} else {
		logger.Success("The following EKS cluster resource for %q will be deleted (if in doubt, check CloudFormation console): %q", ctl.Spec.ClusterName, strings.Join(deletedResources, ", "))
	}
}

// TODO: comment
func doDeleteCluster(cfg *api.ClusterConfig, name string) error {
	ctl := eks.New(cfg)

	if err := ctl.CheckAuth(); err != nil {
		return err
	}

	if cfg.ClusterName != "" && name != "" {
		return fmt.Errorf("--name=%s and argument %s cannot be used at the same time", cfg.ClusterName, name)
	}

	if name != "" {
		cfg.ClusterName = name
	}

	if cfg.ClusterName == "" {
		return fmt.Errorf("--name must be set")
	}

	deleteClusterAndResources(ctl)

	ctl.MaybeDeletePublicSSHKey()

	kubeconfig.MaybeDeleteConfig(cfg.ClusterName)

	logger.Success("all EKS cluster resource for %q will be deleted (if in doubt, check CloudFormation console)", cfg.ClusterName)

	return nil
}

// TODO: ask someone if closures are used a lot in Go.
// AND DELETE ALL OF THIS!
//
//func _deleteClusterAndResources(ctl *eks.ClusterProvider ) []string {
//
//	deletedList, hasError := errorHandling()
//
//	stackManager := ctl.NewStackManager()
//
//	hasError(stackManager.WaitDeleteNodeGroup(), "node group")
//
//	if hasError(stackManager.DeleteCluster(), "cluster") {
//		if hasError(ctl.DeprecatedDeleteControlPlane(), "control plane") {
//			hasError(stackManager.DeprecatedDeleteStackControlPlane(), "stack control plane")
//		}
//	}
//
//	hasError(stackManager.DeprecatedDeleteStackServiceRole(), "stack service role")
//	hasError(stackManager.DeprecatedDeleteStackVPC(), "stack VPC")
//	hasError(stackManager.DeprecatedDeleteStackDefaultNodeGroup(), "default node group")
//	return deletedList()
//}
//
//func errorHandling() (func() []string, func(err error, resourceName string) bool) {
//	var deletedResources []string
//
//	return func() []string {
//		return deletedResources
//	},
//		func(err error, resourceName string) bool {
//			logger.Info("this shit:", deletedResources)
//			if err != nil {
//				logger.Debug("continue despite error: %v", err)
//				return true
//			}
//			deletedResources = append(deletedResources, resourceName)
//			return false
//		}
//}