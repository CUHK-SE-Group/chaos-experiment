package controllers

import (
	"chaos-expriment/chaos"
	"context"
	"fmt"
	"strings"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	"github.com/k0kubun/pp/v3"
	"github.com/sirupsen/logrus"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/util/rand"
)

func MakeCPUStressors(load int, worker int) v1alpha1.Stressors {
	return v1alpha1.Stressors{
		CPUStressor: &v1alpha1.CPUStressor{
			Load:     &load,
			Stressor: v1alpha1.Stressor{Workers: worker},
		},
	}
}
func MakeMemoryStressors(memorySize string, worker int) v1alpha1.Stressors {
	return v1alpha1.Stressors{
		MemoryStressor: &v1alpha1.MemoryStressor{
			Size:     memorySize,
			Stressor: v1alpha1.Stressor{Workers: worker},
		},
	}
}

func ScheduleStressChaos(cli client.Client, namespace string, appList []string, stressors v1alpha1.Stressors, stressType string) {
	workflowName := strings.ToLower(fmt.Sprintf("%s-%s-%s", namespace, stressType, rand.String(6)))
	workflowSpec := v1alpha1.WorkflowSpec{
		Entry: workflowName,
		Templates: []v1alpha1.Template{
			{
				Name:     workflowName,
				Type:     v1alpha1.TypeSerial,
				Children: nil,
			},
		},
	}
	for idx, appName := range appList {

		spec := chaos.GenerateStressChaosSpec(namespace, appName, stressors)

		workflowSpec.Templates = append(workflowSpec.Templates, v1alpha1.Template{
			Name: strings.ToLower(fmt.Sprintf("%s-%s-%s", namespace, appName, stressType)),
			Type: v1alpha1.TypeStressChaos,
			EmbedChaos: &v1alpha1.EmbedChaos{
				StressChaos: spec,
			},
			Deadline: pointer.String("5m"),
		})
		if idx < len(appList)-1 {
			workflowSpec.Templates = append(workflowSpec.Templates, v1alpha1.Template{
				Name:     fmt.Sprintf("%s-%d", "sleep", idx),
				Type:     v1alpha1.TypeSuspend,
				Deadline: pointer.String("10m"),
			})
		}
	}

	for i, template := range workflowSpec.Templates {
		if i == 0 {
			continue
		}
		workflowSpec.Templates[0].Children = append(workflowSpec.Templates[0].Children, template.Name)
	}

	workflowChaos, err := chaos.NewWorkflowChaos(chaos.WithName(workflowName), chaos.WithNamespace(namespace), chaos.WithWorkflowSpec(&workflowSpec))
	if err != nil {
		logrus.Errorf("Failed to create chaos workflow: %v", err)
	}

	if err != nil {
		logrus.Errorf("Failed to create chaos: %v", err)
	}

	pp.Print("%+v", workflowChaos)
	create, err := workflowChaos.ValidateCreate()
	if err != nil {
		logrus.Errorf("Failed to validate create chaos: %v", err)
	}
	logrus.Infof("create warning: %v", create)
	err = cli.Create(context.Background(), workflowChaos)
	if err != nil {
		logrus.Errorf("Failed to create chaos: %v", err)
	}
}